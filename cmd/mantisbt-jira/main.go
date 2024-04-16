// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
	"github.com/klauspost/compress/gzhttp"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
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
			if s, _, ok := strings.Cut(jerr.Code, " "); ok {
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

	client := *http.DefaultClient
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}
	client.Transport = gzhttp.Transport(client.Transport)
	clientJar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client.Jar = clientJar
	svc := SVC{
		Jira: Jira{HTTPClient: &client},
	}
	if svc.BaseURL = os.Getenv("JIRA_URL"); svc.BaseURL == "" {
		svc.BaseURL = DefaultJiraURL
	}

	var mantisID int
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.IntVar(&mantisID, "mantisid", 0, "mantisID")
	flagAttachFileName := fs.String("filename", "", "override file name")
	addAttachmentCmd := ffcli.Command{Name: "attach", FlagSet: fs,
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

	fs = flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.IntVar(&mantisID, "mantisid", 0, "mantisID")
	addCommentCmd := ffcli.Command{Name: "comment", FlagSet: fs,
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
				if err = svc.Enqueue(ctx, queuesDir, task{
					Name:     "IssueAddComment",
					MantisID: mantisID, IssueID: issueID, Comment: body,
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
			return svc.IssueAddComment(ctx, issueID, body)
		},
	}

	issueGetCmd := ffcli.Command{Name: "get",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			if err = svc.init(); err != nil {
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

	issueExistsCmd := ffcli.Command{Name: "exists",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			if err = svc.init(); err != nil {
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
	issueMantisIDCmd := ffcli.Command{Name: "mantisID",
		Exec: func(ctx context.Context, args []string) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			issueID := args[0]
			if err = svc.init(); err != nil {
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

	issueCmd := ffcli.Command{Name: "issue",
		Subcommands: []*ffcli.Command{&issueGetCmd, &issueExistsCmd, &issueMantisIDCmd},
		Exec:        issueExistsCmd.Exec,
	}

	serveCmd := ffcli.Command{Name: "serve",
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 0 {
				queuesDir = args[0]
			}
			return serve(ctx, queuesDir)
		},
	}

	fs = flag.NewFlagSet("jira", flag.ContinueOnError)
	fs.StringVar(&svc.BaseURL, "jira-base", svc.BaseURL, "JIRA base URL (with basic auth!)")
	fs.StringVar(&svc.BasicUser, "basic-user", os.Getenv("JIRA_USER"), "JIRA user")
	fs.StringVar(&svc.BasicUserPassword, "basic-password", os.Getenv("JIRA_PASSWORD"), "JIRA password")
	fs.StringVar(&svc.JIRAUser, "jira-user", os.Getenv("SVC_USER"), "service user")
	fs.StringVar(&svc.JIRAPassword, "jira-password", os.Getenv("SVC_PASSWORD"), "service password")
	fs.DurationVar(&timeout, "timeout", 1*time.Minute, "timeout")
	fs.Var(&verbose, "v", "verbose logging")
	fs.StringVar(&queuesDir, "queues", "", "queues directory")
	ucd, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	// nosemgrep: go.lang.correctness.permissions.file_permission.incorrect-default-permission
	_ = os.MkdirAll(ucd, 0750)
	fs.StringVar(&svc.TokensFile, "token", filepath.Join(ucd, "jira-token.json"), "JIRA token file")
	app := ffcli.Command{Name: "jira", FlagSet: fs, Options: []ff.Option{ff.WithEnvVarNoPrefix()},
		Subcommands: []*ffcli.Command{
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
	if err := app.Parse(os.Args[1:]); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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
