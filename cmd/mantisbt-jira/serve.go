package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/UNO-SOFT/mantisbt-plugins-Jira/cmd/mantisbt-jira/dirq"
)

type queue struct {
	dirq.Queue
}

type task struct {
	Name               string
	IssueID, Comment   string
	FileName, MIMEType string
	Data               []byte
}

func (Q queue) Enqueue(ctx context.Context, t task) error {
	body, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return Q.Queue.Enqueue(body)
}

func serve(ctx context.Context, svc Jira, dir string) error {
	logger.Debug("serve", "svc", svc, "dir", dir)
	f := func(ctx context.Context, p []byte) error {
		logger.Debug("Dequeue", "data", p)
		var t task
		if err := json.Unmarshal(p, &t); err != nil {
			return err
		}
		logger.Debug("dequeued", slog.String("name", t.Name))
		switch t.Name {
		case "IssueAddComment":

			return svc.IssueAddComment(ctx, t.IssueID, t.Comment)

		case "IssueAddAttachment":
			return svc.IssueAddAttachment(ctx, t.IssueID, t.FileName, t.MIMEType, bytes.NewReader(t.Data))

		default:
			return fmt.Errorf("%q: %w", t.Name, errUnknownCommand)
		}
		return nil
	}
	Q := queue{Queue: dirq.Queue{Dir: dir}}
	if err := Q.Dequeue(ctx, f); err != nil && !errors.Is(err, dirq.ErrEmpty) {
		return err
	}
	return Q.Watch(ctx, f)
}

var errUnknownCommand = errors.New("unknown command")
