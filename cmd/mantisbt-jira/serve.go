// Copyright 2025 Tamas Gulacsi. All rights reserved.

package main

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/renameio/v2"
	"github.com/rogpeppe/retry"

	"github.com/UNO-SOFT/mantisbt-plugins-Jira/cmd/mantisbt-jira/dirq"
)

const configFileName = "jira-config.json"

type task struct {
	Name               string
	IssueID, Comment   string
	FileName, MIMEType string
	TransitionID       string
	TargetStatusID     string
	Data               []byte
	MantisID           int
}

func (svc *SVC) Close() error {
	q := svc.queue
	svc.queue = nil
	if q != nil {
		return q.Close()
	}
	return nil
}

func (svc *SVC) GetMantisID(ctx context.Context, issueID string) (string, error) {
	//  A Mantis külső azonosítót jelenleg a fieldID 15902 mezőben kapjuk teszten, ezt kérlek cseréld le a 19001-es fieldID mezőre, élesen pedig a 18913 fieldID-ban kell érkezzen.
	fld := "18913"
	if svc.isTest() {
		fld = "19001"
	}
	logger := logger.With("issueID", issueID, "field", fld)
	issue, err := svc.IssueGet(ctx, issueID, []string{"customfield_" + fld})
	if err != nil {
		logger.Error("IssueGet", "error", err)
		return "", err
	}
	mID := issue.Fields.MantisID()
	if mID == "" {
		logger.Warn("issue MantisID", "issue", issue)
	} else {
		logger.Info("issue MantisID", "mantisID", mID)
	}
	// fmt.Println(issue.Fields.MantisID)
	return mID, nil
}

func (svc *SVC) checkMantisIssueID(ctx context.Context, issueID string, mantisID int) (bool, error) {
	if mantisID == 0 {
		logger.Warn("checkMantisIssueID", "mantisID", mantisID)
		return true, nil
	}
	// $t_mantis_id = trim(
	// 	$this->call("issue", array( "mantisID", $t_issueid ) )[1]
	// );
	// if( $t_mantis_id != $p_bug_id ) {
	// 	$this->log("mantisID=$t_mantis_id bugID=$p_bug_id");
	// 	return;
	// }
	issueMantisID, err := svc.GetMantisID(ctx, issueID)
	if err != nil || issueMantisID == "" {
		logger.Error("IssueGet", "issueID", issueID, "error", err)
		return false, err
	}
	logger.Info("checkMantisIssueID", "mantisID", mantisID, "issueID", issueID, "issueMantisID", issueMantisID)
	// fmt.Println(issue.Fields.MantisID)
	i, err := strconv.Atoi(issueMantisID)
	if err != nil {
		err = fmt.Errorf("%w: %w", err, errSkip)
	}
	return i == mantisID, err

}

func (svc *SVC) Enqueue(ctx context.Context, queuesDir string, t task) error {
	logger.Info("Enqueue", "queuesDir", queuesDir, "queue", svc.queueName)
	if svc.queueName == "" || svc.queue.Dir == "" {
		b, err := json.Marshal(svc)
		if err != nil {
			return err
		}
		hsh := sha512.Sum512_224(b)
		svc.queueName = base64.URLEncoding.EncodeToString(hsh[:])
		dir := filepath.Join(queuesDir, svc.queueName)
		mkdErr := os.MkdirAll(dir, 0750)
		fn := filepath.Join(dir, configFileName)
		if fi, err := os.Stat(fn); !(err == nil && fi.Size() == int64(len(b))) {
			logger.Info("write config", "file", fn)
			if err = renameio.WriteFile(fn, b, 0400); err != nil {
				logger.Error("Write config", "file", fn, "error", err, "MkdirAll", mkdErr)
				return fmt.Errorf("write %q: %w", fn, err)
			}
		}
		if svc.queue, err = dirq.New(dir); err != nil {
			logger.Error("new queue", "dir", dir, "error", err)
			return err
		}
	}

	body, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return svc.queue.Enqueue(body)
}

var errSkip = errors.New("skip")

func serve(ctx context.Context, dir string, alertEmails []string) error {
	logger.Debug("serve", "dir", dir)

	sendAlert := func(err error) error { return nil }
	sendAlerts := func() error { return nil }
	if len(alertEmails) != 0 {
		hostname, _ := os.Hostname()
		var (
			buf       bytes.Buffer
			errs      []error
			errsSeen  = make(map[string]struct{})
			errsTimer *time.Timer
			errsMu    sync.Mutex
		)
		sendAlert = func(alert error) error {
			if alert == nil {
				return nil
			}
			var doSend bool
			errS := alert.Error()
			errsMu.Lock()
			if _, ok := errsSeen[errS]; !ok {
				errsSeen[errS] = struct{}{}
				errs = append(errs, alert)
				if doSend = len(errs) >= 100; !doSend {
					logger.Debug("sendAlert", "len", len(errs), "timer", errsTimer != nil)
					if errsTimer == nil {
						doSend = true
						errsTimer = time.AfterFunc(15*time.Minute, func() { sendAlerts() })
					}
				}
			}
			errsMu.Unlock()
			if doSend {
				return sendAlerts()
			}
			return nil
		}
		sendAlerts = func() error {
			var alert error
			errsMu.Lock()
			if len(errs) != 0 {
				alert = errors.Join(errs...)
				errs = errs[:0]
				clear(errsSeen)
				if errsTimer != nil {
					errsTimer.Stop()
					errsTimer = nil
				}
			}
			errsMu.Unlock()
			logger.Debug("sendAlerts", "alert", alert)
			if alert == nil {
				return nil
			}
			cmd := exec.CommandContext(ctx, "sendmail", alertEmails...)
			buf.Reset()
			fmt.Fprintf(&buf, "From: mantisbt-jira@"+hostname+"\r\nSubject: Mantis->JIRA hiba\r\n\r\n%+v", alert)
			cmd.Stdin = bytes.NewReader(buf.Bytes())
			if b, err := cmd.CombinedOutput(); err != nil {
				logger.Error("sendmail", "args", cmd.Args, "output", string(b), "error", err)
				return err
			}
			logger.Info("sendmail", "args", cmd.Args)
			return nil
		}
		defer sendAlerts()
	}

	processOne := func(ctx context.Context, svc *SVC, p []byte, logger *slog.Logger) error {
		logger.Debug("Dequeue", "data", p)
		var t task
		if err := json.Unmarshal(p, &t); err != nil {
			return err
		}
		logger.Debug("dequeued", slog.String("name", t.Name))
		if ok, err := svc.checkMantisIssueID(ctx, t.IssueID, t.MantisID); err != nil {
			return err
		} else if !ok {
			logger.Warn("not a JIRA issue", "issueID", t.IssueID, "mantisID", t.MantisID, "task", t)
			return nil
		}
		var err error
		switch t.Name {
		case "IssueAddComment":
			err = svc.IssueAddComment(ctx, t.IssueID, t.Comment)

		case "IssueAddAttachment":
			err = svc.IssueAddAttachment(ctx, t.IssueID, t.FileName, t.MIMEType, bytes.NewReader(t.Data))

		case "IssueDoTransition":
			err = svc.IssueDoTransition(ctx, t.IssueID, t.TransitionID, t.Comment)

		case "IssueDoTransitionTo":
			err = svc.IssueDoTransitionTo(ctx, t.IssueID, t.TargetStatusID, t.Comment)

		default:
			return fmt.Errorf("%q: %w", t.Name, errUnknownCommand)
		}
		if err != nil {
			logger.Error("DO", "name", t.Name, "task", t, "error", err)
			if sendAlert != nil {
				if saErr := sendAlert(err); saErr != nil {
					logger.Error("sendAlert", "task", t, "sendAlert", saErr)
				}
			}
		}
		return err
	}

	services := make(map[string]*SVC)
	defer func() {
		for _, svc := range services {
			svc.Close()
		}
	}()
	batch := func() error {
		dis, err := os.ReadDir(dir)
		if len(dis) == 0 && err != nil {
			return fmt.Errorf("ReadDir(%q): %w", dir, err)
		}
		for _, di := range dis {
			if !di.Type().IsDir() {
				continue
			}
			// Init only once per service
			if _, ok := services[di.Name()]; ok {
				continue
			}
			svc := new(SVC)
			services[di.Name()] = svc
			dir := filepath.Join(dir, di.Name())
			logger := logger.With("queue", dir)
			fn := filepath.Join(dir, configFileName)
			logger.Info("Read config", "file", fn)
			if b, err := os.ReadFile(fn); err != nil {
				if os.IsNotExist(err) {
					logger.Info("Read config", "file", fn, "error", err)
				} else {
					logger.Warn("Read config", "file", fn, "error", err)
				}
				continue
			} else if err = json.Unmarshal(b, &svc); err != nil {
				logger.Error("unmarshal %q: %w", string(b), err)
				continue
			}
			// svc.TokensFile = filepath.Join(dir, "jira-token.json")
			if err = svc.init(); err != nil {
				return err
			}
			Q, err := dirq.New(dir)
			if err != nil {
				return err
			}
			g := func(ctx context.Context, msg []byte) error {
				logger.Warn("processOne", "msg", string(msg))
				if err := processOne(ctx, svc, msg, logger); err != nil {
					logger.Error("processOne", "msg", msg, "error", err)
					if !errors.Is(err, errSkip) {
						return err
					}
				}
				return nil
			}

			go func() {
				normalStrategy := retry.Strategy{Delay: 15 * time.Second, MaxDelay: time.Hour, Factor: 1.5}
				authErrStrategy := retry.Strategy{Delay: time.Hour, Factor: 2, MaxDelay: 6 * time.Hour}
				var lastErrIsAuth bool
				for iter := normalStrategy.Start(); ; {
					if err := Q.Dequeue(ctx, g); err != nil {
						var isAuth bool
						if errors.Is(err, dirq.ErrEmpty) {
							logger.Debug("Dequeue empty")
						} else if errors.Is(err, errAuthenticate) {
							logger.Warn("Dequeue", "error", err)
							sendAlert(err)
							isAuth = true
						} else {
							logger.Error("Dequeue", "error", err)
						}
						if lastErrIsAuth && !isAuth {
							iter.Reset(&normalStrategy, nil)
						} else if !lastErrIsAuth && isAuth {
							iter.Reset(&authErrStrategy, nil)
						}
						lastErrIsAuth = isAuth
					}
					if !iter.Next(ctx.Done()) {
						break
					}
				}
			}()
			go Q.Watch(ctx, g)
		}
		return nil
	}

	if err := batch(); err != nil {
		return err
	}
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := batch(); err != nil {
				return err
			}
		}
	}
}

var errUnknownCommand = errors.New("unknown command")
