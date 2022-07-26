// Copyright 2022 Tamás Gulácsi. All rights reserved.

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"

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
	var err error
	if client.Jar, err = cookiejar.New(nil); err != nil {
		return err
	}
	svc := Jira{HTTPClient: &client}

	fs := flag.NewFlagSet("jira", flag.ExitOnError)
	flagBaseURL := fs.String("jira-base", DefaultJiraURL, "JIRA base URL (with basic auth!)")
	fs.StringVar(&svc.Token.Username, "svc-user", "", "service user")
	fs.StringVar(&svc.Token.Password, "svc-password", "", "service password")
	flagBasicUser := fs.String("jira-user", "", "JIRA user")
	flagBasicPassword := fs.String("jira-password", "", "JIRA password")
	ucd, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	fs.StringVar(&svc.Token.FileName, "token", filepath.Join(ucd, "jira-token.json"), "JIRA token file")
	app := ffcli.Command{Name: "jira", FlagSet: fs, Options: []ff.Option{ff.WithEnvVarNoPrefix()},
		Exec: func(ctx context.Context, args []string) error {
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
