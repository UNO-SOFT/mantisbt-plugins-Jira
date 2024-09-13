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
	"time"

	"github.com/google/renameio/v2"

	"github.com/UNO-SOFT/mantisbt-plugins-Jira/cmd/mantisbt-jira/dirq"
)

const configFileName = "jira-config.json"

type task struct {
	Name               string
	IssueID, Comment   string
	FileName, MIMEType string
	TransitionID       string
	MantisID           int
	Data               []byte
}

func (svc *SVC) GetMantisID(ctx context.Context, issueID string) (string, error) {
	issue, err := svc.IssueGet(ctx, issueID, []string{"customfield_15902"})
	if err != nil {
		logger.Error("IssueGet", "issueID", issueID, "error", err)
		return "", err
	}
	logger.Info("issue MantisID", "issueID", issueID, "mantisID", issue.Fields.MantisID)
	// fmt.Println(issue.Fields.MantisID)
	return issue.Fields.MantisID, nil
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
	if err != nil {
		logger.Error("IssueGet", "issueID", issueID, "error", err)
		return false, err
	}
	logger.Info("checkMantisIssueID", "mantisID", mantisID, "issueID", issueID, "issueMantisID", issueMantisID)
	// fmt.Println(issue.Fields.MantisID)
	i, err := strconv.Atoi(issueMantisID)
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
		_ = os.MkdirAll(dir, 0750)
		fn := filepath.Join(dir, configFileName)
		logger.Info("write config", "file", fn)
		if err = renameio.WriteFile(fn, b, 0400); err != nil {
			logger.Error("Write config", "file", fn, "error", err)
			return fmt.Errorf("write %q: %w", fn, err)
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

func serve(ctx context.Context, dir string, alertEmails []string) error {
	logger.Debug("serve", "dir", dir)

	sendAlert := func(err error) error { return nil }
	if len(alertEmails) != 0 {
		var buf bytes.Buffer
		sendAlert = func(alert error) error {
			cmd := exec.CommandContext(ctx, "sendmail", alertEmails...)
			buf.Reset()
			fmt.Fprintf(&buf, "From: mantisbt-jira@lnx-web-uno\r\nSubject: Mantis->JIRA hiba\r\n\r\n%+v", alert)
			cmd.Stdin = bytes.NewReader(buf.Bytes())
			if b, err := cmd.CombinedOutput(); err != nil {
				logger.Error("sendmail", "args", cmd.Args, "output", string(b), "error", err)
				return err
			}
			logger.Info("sendmail", "args", cmd.Args)
			return nil
		}
	}

	f := func(ctx context.Context, svc *SVC, p []byte, logger *slog.Logger) error {
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
			err = svc.IssueDoTransition(ctx, t.IssueID, t.TransitionID)

		default:
			return fmt.Errorf("%q: %w", t.Name, errUnknownCommand)
		}
		if err != nil {
			logger.Error(t.Name, "task", t, "error", err)
			if sendAlert != nil {
				if saErr := sendAlert(err); saErr != nil {
					logger.Error("sendAlert", "task", t, "sendAlert", saErr)
				}
			}
		}
		return err
	}

	seen := make(map[string]struct{})
	F := func() error {
		dis, err := os.ReadDir(dir)
		if len(dis) == 0 && err != nil {
			return fmt.Errorf("ReadDir(%q): %w", dir, err)
		}
		for _, di := range dis {
			if !di.Type().IsDir() {
				continue
			}
			if _, ok := seen[di.Name()]; ok {
				continue
			}
			seen[di.Name()] = struct{}{}
			dir := filepath.Join(dir, di.Name())
			logger := logger.With("queue", dir)
			fn := filepath.Join(dir, configFileName)
			logger.Info("Read config", "file", fn)
			var svc SVC
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
				return f(ctx, &svc, msg, logger)
			}
			if err := Q.Dequeue(ctx, g); err != nil && !errors.Is(err, dirq.ErrEmpty) {
				if errors.Is(err, errAuthenticate) {
					logger.Warn("Dequeue", "error", err)
					sendAlert(err)
					continue
				}
				return err
			}

			ticker := time.NewTicker(time.Minute)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := Q.Dequeue(ctx, g); err != nil {
							if errors.Is(err, dirq.ErrEmpty) {
								logger.Info("Dequeue empty")
							} else if errors.Is(err, errAuthenticate) {
								logger.Warn("Dequeue", "error", err)
								return
							} else {
								logger.Error("Dequeue", "error", err)
							}
						}
					}
				}
			}()
			go Q.Watch(ctx, g)
		}
		return nil
	}

	if err := F(); err != nil {
		return err
	}
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := F(); err != nil {
				return err
			}
		}
	}
}

var errUnknownCommand = errors.New("unknown command")
