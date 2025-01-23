// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/UNO-SOFT/mantisbt-plugins-Jira/cmd/mantisbt-jira/dirq"
	"github.com/UNO-SOFT/zlog/v2"
	"github.com/UNO-SOFT/zlog/v2/loghttp"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
	"github.com/tgulacsi/go/version"
)

var verbose zlog.VerboseVar = 1
var logger = zlog.NewLogger(zlog.MaybeConsoleHandler(&verbose, os.Stderr)).SLog()

// https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/openapi.json

const (
	// DefaultJiraURL is the default JIRA URL
	DefaultJiraURL = "https://partnerapi-test.aegon.hu/partner/v1/ticket/update"
)

func main() {
	if err := Main(); err != nil {
		logger.Error("Main", "error", err)
		var jerr *JIRAError
		if errors.As(err, &jerr) {
			//logger.Info("as jiraerr", "error", jerr, "code", jerr.Code)
			if s, _, ok := strings.Cut(jerr.Code, " "); ok && s != "" {
				//logger.Info("cut", "s", s)
				if i, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
					if 401 <= i && i < 500 {
						os.Exit(i - 400)
					}
				}
			}
		}
		os.Exit(1)
	}
}

type SVC struct {
	Jira
	BaseURL                      string
	BasicUser, BasicUserPassword string
	TokensFile                   string
	JIRAUser, JIRAPassword       string
	queueName                    string
	queue                        dirq.Queue
}

// Main is the main function
func Main() error {
	var queuesDir string
	timeout := time.Minute

	svc := SVC{BaseURL: os.Getenv("JIRA_URL")}
	if svc.BaseURL == "" {
		svc.BaseURL = DefaultJiraURL
	}

	var mantisID int
	FS := ff.NewFlagSet("attach")
	FS.IntVar(&mantisID, 0, "mantisid", 0, "mantisID")
	flagAttachFileName := FS.StringLong("filename", "", "override file name")
	addAttachmentCmd := ff.Command{Name: "attach", Flags: FS,
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			r := os.Stdin
			if !(len(args) < 2 || args[1] == "" || args[1] == "-") {
				var err error
				if r, err = os.Open(args[1]); err != nil {
					return fmt.Errorf("open %q: %w", args[1], err)
				}
			}
			defer r.Close()
			fileName := *flagAttachFileName
			if fileName == "" {
				fileName = r.Name()
			}
			var a [1024]byte
			n, err := r.Read(a[:])
			if n == 0 {
				logger.Error("read", "file", r, "error", err)
				return err
			}
			b := a[:n]
			mimeType := http.DetectContentType(b)
			logger.Info("IssueAddAttachment", "issueID", issueID, "fileName", fileName, "mimeType", mimeType)
			if queuesDir != "" {
				if b, err = io.ReadAll(io.MultiReader(bytes.NewReader(b), r)); err != nil {
					return err
				}
				if err = svc.Enqueue(ctx, queuesDir, task{
					Name:    "IssueAddAttachment",
					IssueID: issueID, MantisID: mantisID,
					FileName: fileName, MIMEType: mimeType, Data: b,
				}); err != nil {
					logger.Error("queue", "error", err)
				} else {
					return nil
				}
			}
			if err = svc.init(); err != nil {
				return err
			}
			if ok, err := svc.checkMantisIssueID(ctx, issueID, mantisID); err != nil {
				return err
			} else if !ok {
				return nil
			}
			return svc.IssueAddAttachment(ctx, issueID, fileName, mimeType, io.MultiReader(bytes.NewReader(b), r))
		},
	}

	FS = ff.NewFlagSet("attach")
	FS.IntVar(&mantisID, 0, "mantisid", 0, "mantisID")
	addCommentCmd := ff.Command{Name: "comment", Flags: FS,
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			body := strings.Join(args[1:], " ")
			if len(args) < 2 || (len(args) == 2 && (args[1] == "" || args[1] == "-")) {
				var buf strings.Builder
				if _, err := io.Copy(&buf, os.Stdin); err != nil {
					return err
				}
				body = buf.String()
			}
			if queuesDir != "" {
				if err := svc.Enqueue(ctx, queuesDir, task{
					Name:     "IssueAddComment",
					MantisID: mantisID, IssueID: issueID, Comment: body,
				}); err != nil {
					logger.Error("queue", "error", err)
				} else {
					return nil
				}
			}
			if err := svc.init(); err != nil {
				return err
			}
			if ok, err := svc.checkMantisIssueID(ctx, issueID, mantisID); err != nil {
				return err
			} else if !ok {
				return nil
			}
			return svc.IssueAddComment(ctx, issueID, body)
		},
	}

	issueGetCmd := ff.Command{Name: "get",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			if err := svc.init(); err != nil {
				return err
			}
			issue, err := svc.IssueGet(ctx, issueID, nil)
			if err != nil {
				fmt.Println("ERR", err)
				return err
			}
			fmt.Println(issue)
			return nil
		},
	}

	issueExistsCmd := ff.Command{Name: "exists",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			if err := svc.init(); err != nil {
				return err
			}
			issue, err := svc.IssueGet(ctx, issueID, []string{"status"})
			if err != nil {
				fmt.Println("ERR", err)
				return err
			}
			logger.Info("issue exists", "issueID", issueID, "status", issue.Fields.Status)
			fmt.Println(issue.Fields.Status.StatusCategory.Name)
			return nil
		},
	}

	issueMantisIDCmd := ff.Command{Name: "mantisID",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			if err := svc.init(); err != nil {
				return err
			}
			issueMantisID, err := svc.GetMantisID(ctx, issueID)
			if err != nil {
				fmt.Println("ERR", err)
				return err
			}
			logger.Info("issue MantisID", "issueID", issueID, "mantisID", issueMantisID)
			fmt.Println(issueMantisID)
			return nil
		},
	}

	transitionsGetCmd := ff.Command{Name: "get",
		Usage: "get <issueID>",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			if err := svc.init(); err != nil {
				return err
			}
			transitions, err := svc.IssueTransitions(ctx, issueID, true)
			if err != nil {
				fmt.Println("ERR", err)
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(transitions); err != nil {
				return err
			}
			for _, t := range transitions {
				for _, f := range t.Fields {
					if f.Required {
						fmt.Println("required:", f)
					}
				}
			}
			return nil
		},
	}

	var comment string
	FS = ff.NewFlagSet("transition to")
	FS.StringVar(&comment, 'm', "comment", "", "comment")
	transitionToCmd := ff.Command{Name: "to", Flags: FS,
		Usage: "to <issueID> <targetStatusID>",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			targetStatusID := args[1]
			if queuesDir != "" {
				if err := svc.Enqueue(ctx, queuesDir, task{
					Name:    "IssueDoTransitionTo",
					IssueID: issueID, Comment: comment,
					TargetStatusID: targetStatusID,
				}); err != nil {
					logger.Error("queue", "error", err)
				} else {
					return nil
				}
			}
			if err := svc.init(); err != nil {
				return err
			}
			err := svc.IssueDoTransitionTo(ctx, issueID, targetStatusID, comment)
			if err != nil {
				fmt.Println("ERR", err)
				return err
			}
			return nil
		},
	}

	FS = ff.NewFlagSet("transition")
	FS.StringVar(&comment, 'm', "comment", "", "comment")
	issueDoTransitionCmd := ff.Command{Name: "transition", Flags: FS,
		Usage:       "transition <issueID> <transitionID>",
		Subcommands: []*ff.Command{&transitionToCmd, &transitionsGetCmd},
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			transitionID := args[1]
			if queuesDir != "" {
				if err := svc.Enqueue(ctx, queuesDir, task{
					Name:    "IssueDoTransition",
					IssueID: issueID, Comment: comment,
					TransitionID: transitionID,
				}); err != nil {
					logger.Error("queue", "error", err)
				} else {
					return nil
				}
			}
			if err := svc.init(); err != nil {
				return err
			}
			err := svc.IssueDoTransition(ctx, issueID, transitionID, comment)
			if err != nil {
				fmt.Println("ERR", err)
				return err
			}
			return nil
		},
	}

	issueCmd := ff.Command{Name: "issue",
		Subcommands: []*ff.Command{
			&issueGetCmd, &issueExistsCmd,
			&issueMantisIDCmd,
			&issueDoTransitionCmd,
		},
		Exec: issueExistsCmd.Exec,
	}

	FS = ff.NewFlagSet("serve")
	flagServeEmail := FS.StringLong("alert", "t.gulacsi+jira@unosoft.hu", "comma-separated list of emails to send alerts to")
	serveCmd := ff.Command{Name: "serve", Flags: FS,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				queuesDir = args[0]
			}
			return serve(ctx, queuesDir, strings.Split(*flagServeEmail, ","))
		},
	}

	FS = ff.NewFlagSet("jira")
	FS.StringVar(&svc.BaseURL, 0, "jira-base", svc.BaseURL, "JIRA base URL (with basic auth!)")
	FS.StringVar(&svc.BasicUser, 0, "basic-user", os.Getenv("JIRA_USER"), "JIRA user")
	FS.StringVar(&svc.BasicUserPassword, 0, "basic-password", os.Getenv("JIRA_PASSWORD"), "JIRA password")
	FS.StringVar(&svc.JIRAUser, 0, "jira-user", os.Getenv("SVC_USER"), "service user")
	FS.StringVar(&svc.JIRAPassword, 0, "jira-password", os.Getenv("SVC_PASSWORD"), "service password")
	FS.DurationVar(&timeout, 0, "timeout", 1*time.Minute, "timeout")
	FS.Value('v', "verbose", &verbose, "verbose logging")
	flagVersion := FS.BoolLongDefault("version", false, "print version")
	FS.StringVar(&queuesDir, 0, "queues", "", "queues directory")
	ucd, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	// nosemgrep: go.lang.correctness.permissions.file_permission.incorrect-default-permission
	_ = os.MkdirAll(ucd, 0750)
	FS.StringVar(&svc.TokensFile, 0, "token", filepath.Join(ucd, "jira-token.json"), "JIRA token file")
	app := ff.Command{Name: "jira", Flags: FS,
		Subcommands: []*ff.Command{
			&addAttachmentCmd, &addCommentCmd,
			&issueCmd,
			&serveCmd,
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				args = append(args, "INCIDENT-6508")
			}
			issue, err := svc.IssueGet(ctx, args[0], nil)
			if err != nil {
				return err
			}
			fmt.Println("issue", issue)

			comments, err := svc.IssueComments(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Println("comments:", comments)
			return nil
		},
	}
	if err := app.Parse(
		os.Args[1:],
		ff.WithEnvVars(),
	); err != nil {
		ffhelp.Command(&app).WriteTo(os.Stderr)
		if errors.Is(err, ff.ErrHelp) {
			return nil
		}
		return err
	}
	if *flagVersion {
		fmt.Println(version.Main())
		return nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx = zlog.NewSContext(ctx, logger)

	client := *http.DefaultClient
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}
	if logger.Enabled(ctx, slog.LevelDebug) {
		client.Transport = loghttp.Transport(client.Transport)
	}
	logger.Debug("Main", "logtransport", client.Transport)
	clientJar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client.Jar = clientJar
	svc.Jira = Jira{HTTPClient: &client}

	start := time.Now()
	err = app.Run(ctx)
	logger.Info("run", "dur", time.Since(start).String())
	return err
}

func (svc *SVC) init() error {
	svcURL, err := url.Parse(svc.BaseURL)
	if err != nil {
		return fmt.Errorf("parse %q: %w", svc.BaseURL, err)
	}
	svc.Jira.URL = svcURL
	if svc.BasicUser != "" {
		svc.URL.User = url.UserPassword(svc.BasicUser, svc.BasicUserPassword)
	}
	svc.Load(svc.TokensFile, svc.JIRAUser, svc.JIRAPassword)
	return nil
}
