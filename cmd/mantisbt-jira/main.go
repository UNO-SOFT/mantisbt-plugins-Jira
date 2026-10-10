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

	"flag"
	"github.com/UNO-SOFT/cli"
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
	queue                        *dirq.Queue
}

// Main is the main function
func Main() error {
	slog.SetDefault(logger)
	var queuesDir string
	timeout := time.Minute

	svc := SVC{BaseURL: os.Getenv("JIRA_URL")}
	if svc.BaseURL == "" {
		svc.BaseURL = DefaultJiraURL
	}
	defer svc.Close()

	var mantisID int
	FS := flag.NewFlagSet("attach", flag.ContinueOnError)
	FS.IntVar(&mantisID, "mantisid", 0, "mantisID")
	flagAttachFileName := FS.String("filename", "", "override file name")
	addAttachmentCmd := cli.Command{Name: "attach", Flags: FS,
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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

	FS = flag.NewFlagSet("attach", flag.ContinueOnError)
	FS.IntVar(&mantisID, "mantisid", 0, "mantisID")
	addCommentCmd := cli.Command{Name: "comment", Flags: FS,
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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

	issueGetCmd := cli.Command{Name: "get",
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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

	issueExistsCmd := cli.Command{Name: "exists",
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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

	issueMantisIDCmd := cli.Command{Name: "mantisID",
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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

	transitionsGetCmd := cli.Command{Name: "get",
		Usage: "get <issueID>",
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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
	FS = flag.NewFlagSet("transition to", flag.ContinueOnError)
	FS.StringVar(&comment, "comment", "", "comment")
	transitionToCmd := cli.Command{Name: "to", Flags: FS,
		Usage: "to <issueID> <targetStatusID>",
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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
		}, FlagConfigs: []cli.FlagConfig{cli.FlagConfig{Name: "comment", Short: "m"}},
	}

	FS = flag.NewFlagSet("transition", flag.ContinueOnError)
	FS.StringVar(&comment, "comment", "", "comment")
	issueDoTransitionCmd := cli.Command{Name: "transition", Flags: FS,
		Usage:       "transition <issueID> <transitionID>",
		SubCommands: []*cli.Command{&transitionToCmd, &transitionsGetCmd},
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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
		}, FlagConfigs: []cli.FlagConfig{cli.FlagConfig{Name: "comment", Short: "m"}},
	}

	issueCmd := cli.Command{Name: "issue",
		SubCommands: []*cli.Command{
			&issueGetCmd, &issueExistsCmd,
			&issueMantisIDCmd,
			&issueDoTransitionCmd,
		},
		Exec: issueExistsCmd.Exec,
	}

	FS = flag.NewFlagSet("serve", flag.ContinueOnError)
	flagServeEmail := FS.String("alert", "t.gulacsi+jira@unosoft.hu", "comma-separated list of emails to send alerts to")
	serveCmd := cli.Command{Name: "serve", Flags: FS,
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
			if len(args) != 0 {
				queuesDir = args[0]
			}
			return serve(ctx, queuesDir, strings.Split(*flagServeEmail, ","))
		},
	}

	FS = flag.NewFlagSet("jira", flag.ContinueOnError)
	FS.StringVar(&svc.BaseURL, "jira-base", svc.BaseURL, "JIRA base URL (with basic auth!)")
	FS.StringVar(&svc.BasicUser, "basic-user", os.Getenv("JIRA_USER"), "JIRA user")
	FS.StringVar(&svc.BasicUserPassword, "basic-password", os.Getenv("JIRA_PASSWORD"), "JIRA password")
	FS.StringVar(&svc.JIRAUser, "jira-user", os.Getenv("SVC_USER"), "service user")
	FS.StringVar(&svc.JIRAPassword, "jira-password", os.Getenv("SVC_PASSWORD"), "service password")
	FS.DurationVar(&timeout, "timeout", 1*time.Minute, "timeout")
	FS.Var(&verbose, "verbose", "verbose logging")
	flagVersion := FS.Bool("version", false, "print version")
	FS.StringVar(&queuesDir, "queues", "", "queues directory")
	ucd, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	// nosemgrep: go.lang.correctness.permissions.file_permission.incorrect-default-permission
	_ = os.MkdirAll(ucd, 0750)
	FS.StringVar(&svc.TokensFile, "token", filepath.Join(ucd, "jira-token.json"), "JIRA token file")
	app := cli.Command{Name: "jira", Flags: FS,
		SubCommands: []*cli.Command{
			&addAttachmentCmd, &addCommentCmd,
			&issueCmd,
			&serveCmd,
		},
		Exec: func(ctx context.Context, state *cli.State) error {
			args := state.Args
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
		}, FlagConfigs: []cli.FlagConfig{cli.FlagConfig{Name: "verbose", Short: "v"}},
	}
	if err := cli.Parse(&app, os.Args[1:]); err != nil {
		cli.PrintHelp(os.Stderr, &app)
		if errors.Is(err, flag.ErrHelp) {
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
		client.Transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	if verbose > 1 {
		client.Transport = loghttp.Transport(client.Transport)
	}
	// logger.Debug("Main", "logtransport", client.Transport)
	clientJar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client.Jar = clientJar
	svc.Jira = Jira{HTTPClient: &client}

	start := time.Now()
	err = cli.Run(ctx, &app, nil)
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
