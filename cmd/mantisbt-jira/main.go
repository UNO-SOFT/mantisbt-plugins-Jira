// Copyright 2022 Tamás Gulácsi. All rights reserved.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/gzhttp"
	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/tgulacsi/go/zlog"
)

var logger = zlog.New(os.Stderr)

const DefaultJiraURL = "https://partnerapi-uat.aegon.hu/partner/v1/ticket/update"

func main() {
	if err := Main(); err != nil {
		logger.Error(err, "Main")
		os.Exit(1)
	}
}

func Main() error {
	client := *http.DefaultClient
	client.Transport = gzhttp.Transport(client.Transport)
	var err error
	if client.Jar, err = cookiejar.New(nil); err != nil {
		return err
	}
	svc := Jira{HTTPClient: &client}

	addAttachmentCmd := ffcli.Command{Name: "attach",
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
			fileName := r.Name()
			var a [1024]byte
			n, err := r.Read(a[:])
			if n == 0 {
				return err
			}
			b := a[:n]
			mimeType := http.DetectContentType(b)
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

	fs := flag.NewFlagSet("jira", flag.ExitOnError)
	flagBaseURL := fs.String("jira-base", DefaultJiraURL, "JIRA base URL (with basic auth!)")
	fs.StringVar(&svc.Token.Username, "jira-user", "", "service user")
	fs.StringVar(&svc.Token.Password, "jira-password", "", "service password")
	flagBasicUser := fs.String("basic-user", "", "JIRA user")
	flagBasicPassword := fs.String("basic-password", "", "JIRA password")
	flagVerbose := fs.Bool("v", false, "verbose logging")
	ucd, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(ucd, 0750)
	fs.StringVar(&svc.Token.FileName, "token", filepath.Join(ucd, "jira-token.json"), "JIRA token file")
	app := ffcli.Command{Name: "jira", FlagSet: fs, Options: []ff.Option{ff.WithEnvVarNoPrefix()},
		Subcommands: []*ffcli.Command{&addAttachmentCmd, &addCommentCmd},
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
	if *flagVerbose {
		zlog.SetLevel(logger, zlog.TraceLevel)
	}
	if svc.URL, err = url.Parse(*flagBaseURL); err != nil {
		return fmt.Errorf("parse %q: %w", svc.URL, err)
	}
	if *flagBasicUser != "" {
		svc.URL.User = url.UserPassword(*flagBasicUser, *flagBasicPassword)
	}
	svc.Token.AuthURL = svc.URL.JoinPath("auth").String()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	return app.Run(ctx)
}
