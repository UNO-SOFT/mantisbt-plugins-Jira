// Copyright 2024 Tamás Gulácsi. All rights reserved.

// Packge dirq provides a directory+files based multiple provider,
// single consumer persistent queue.
package dirq

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/renameio/v2"
	"github.com/oklog/ulid/v2"
	"github.com/rjeczalik/notify"
)

const ext = ".dirq-item.dat"

type Queue struct {
	Dir string
}

func (Q Queue) Enqueue(p []byte) error {
	return renameio.WriteFile(
		filepath.Join(Q.Dir, ulid.MustNew(ulid.Now(), ulid.DefaultEntropy()).String()+ext),
		p,
		0400)
}

// DequeueOne will call f on the first dequeueable message.
// The message will be deleted iff f returns nil,
// otherwise it remains in the queue.
func (Q Queue) Dequeue(ctx context.Context, f func(context.Context, []byte) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dis, err := os.ReadDir(Q.Dir)
	if len(dis) == 0 {
		return err
	}
	var haveAny bool
	for _, di := range dis {
		nm := di.Name()
		if !(di.Type().IsRegular() && strings.HasSuffix(nm, ext) &&
			len(nm) == 26+len(ext)) {
			continue
		}
		if err := Q.dequeueOne(ctx, f, filepath.Join(Q.Dir, nm)); err != nil {
			return err
		}
		haveAny = true
	}
	if !haveAny {
		return ErrEmpty
	}
	return nil
}

var ErrEmpty = errors.New("queue is empty")

// Dequeue all the incoming messages, continuously.
//
// Calls DequeueOne when a new message arrives (based on notification).
func (Q Queue) Watch(ctx context.Context, f func(context.Context, []byte) error) error {
	c := make(chan notify.EventInfo)
	if err := notify.Watch(Q.Dir, c, notify.InMoveSelf, notify.InMovedTo); err != nil {
		return err
	}
	defer notify.Stop(c)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ei, ok := <-c:
			if !ok {
				return ctx.Err()
			}
			slog.Debug("notify", "path", ei.Path())
			if bn := filepath.Base(ei.Path()); len(bn) == 26+len(ext) && strings.HasSuffix(bn, ext) {
				_ = Q.dequeueOne(ctx, f, ei.Path())
			}
		}
	}
}

func (Q Queue) dequeueOne(ctx context.Context, f func(context.Context, []byte) error, fn string) error {
	fny := fn + ".y"
	if err := os.Rename(fn, fny); err != nil {
		return err
	}
	b, err := os.ReadFile(fny)
	if err != nil {
		return err
	}
	if err := f(ctx, b); err != nil {
		_ = os.Rename(fny, fn)
		return err
	}
	return os.Remove(fny)
}
