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

	"github.com/UNO-SOFT/zlog/v2"
	"github.com/klauspost/compress/gzhttp"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var verbose zlog.VerboseVar
var logger = zlog.NewLogger(zlog.MaybeConsoleHandler(&verbose, os.Stderr)).SLog()

// https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/openapi.json

// DefaultJiraURL is the default JIRA URL
const DefaultJiraURL = "https://partnerapi-uat.aegon.hu/partner/v1/ticket/update"

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

// Main is the main function
func Main() error {
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
	svc := Jira{HTTPClient: &client}

	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	flagAttachFileName := fs.String("filename", "", "override file name")
	addAttachmentCmd := ffcli.Command{Name: "attach",
		FlagSet: fs,
		Exec: func(ctx context.Context, args []string) error {
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
			return svc.IssueAddAttachment(ctx, issueID, fileName, mimeType, io.MultiReader(bytes.NewReader(b), r))
		},
	}

	addCommentCmd := ffcli.Command{Name: "comment",
		Exec: func(ctx context.Context, args []string) error {
			issueID := args[0]
			body := strings.Join(args[1:], " ")
			if len(args) < 2 || (len(args) == 2 && (args[1] == "" || args[1] == "-")) {
				var buf strings.Builder
				if _, err := io.Copy(&buf, os.Stdin); err != nil {
					return err
				}
				body = buf.String()
			}
			return svc.IssueAddComment(ctx, issueID, body)
		},
	}

	issueGetCmd := ffcli.Command{Name: "get",
		Exec: func(ctx context.Context, args []string) error {
			issueID := args[0]
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
			issueID := args[0]
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

	issueCmd := ffcli.Command{Name: "issue",
		Subcommands: []*ffcli.Command{&issueGetCmd, &issueExistsCmd},
		Exec:        issueExistsCmd.Exec,
	}
	fs = flag.NewFlagSet("jira", flag.ContinueOnError)
	flagBaseURL := fs.String("jira-base", DefaultJiraURL, "JIRA base URL (with basic auth!)")
	flagJiraUser := fs.String("jira-user", os.Getenv("SVC_USER"), "service user")
	flagJiraPassword := fs.String("jira-password", os.Getenv("SVC_PASSWORD"), "service password")
	flagTimeout := fs.Duration("timeout", 30*time.Second, "timeout")
	flagBasicUser := fs.String("basic-user", os.Getenv("JIRA_USER"), "JIRA user")
	flagBasicPassword := fs.String("basic-password", os.Getenv("JIRA_PASSWORD"), "JIRA password")
	fs.Var(&verbose, "v", "verbose logging")
	ucd, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	// nosemgrep: go.lang.correctness.permissions.file_permission.incorrect-default-permission
	_ = os.MkdirAll(ucd, 0750)
	flagTokensFile := fs.String("token", filepath.Join(ucd, "jira-token.json"), "JIRA token file")
	app := ffcli.Command{Name: "jira", FlagSet: fs, Options: []ff.Option{ff.WithEnvVarNoPrefix()},
		Subcommands: []*ffcli.Command{
			&addAttachmentCmd, &addCommentCmd,
			&issueCmd,
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
	svcURL, err := url.Parse(*flagBaseURL)
	if err != nil {
		return fmt.Errorf("parse %q: %w", *flagBaseURL, err)
	}
	svc.URL = svcURL
	if *flagBasicUser != "" {
		svc.URL.User = url.UserPassword(*flagBasicUser, *flagBasicPassword)
	}
	svc.Load(*flagTokensFile, *flagJiraUser, *flagJiraPassword)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, *flagTimeout)
	defer cancel()

	start := time.Now()
	err = app.Run(ctx)
	logger.Info("run", "dur", time.Since(start).String())
	return err
}
