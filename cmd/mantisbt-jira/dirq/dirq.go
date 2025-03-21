// Copyright 2024, 2025 Tamás Gulácsi. All rights reserved.

// Packge dirq provides a directory+files based multiple provider,
// single consumer persistent queue.
package dirq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/google/renameio/v2"
	"github.com/oklog/ulid/v2"
	"github.com/rjeczalik/notify"
)

const ext = ".dirq-item.dat"

type Queue struct {
	Dir string

	mu sync.Mutex
	fh *os.File
}

func (Q *Queue) Close() error {
	fh := Q.fh
	Q.fh = nil
	if fh == nil {
		return nil
	}
	defer fh.Close()
	return syscall.Flock(int(fh.Fd()), syscall.LOCK_UN)
}

func New(dir string) (*Queue, error) {
	return &Queue{Dir: dir}, nil
}

// Enqueue a message.
//
// Does not lock (not needed).
func (Q *Queue) Enqueue(p []byte) error {
	return renameio.WriteFile(
		filepath.Join(Q.Dir, ulid.MustNew(ulid.Now(), ulid.DefaultEntropy()).String()+ext),
		p,
		0400)
}

// DequeueOne will call f on the first dequeueable message.
// The message will be deleted iff f returns nil,
// otherwise it remains in the queue.
func (Q *Queue) Dequeue(ctx context.Context, f func(context.Context, []byte) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	Q.mu.Lock()
	defer Q.mu.Unlock()
	if err := Q.lock(); err != nil {
		return err
	}
	dis, err := Q.fh.ReadDir(-1)
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

func (Q *Queue) lock() error {
	if Q.fh != nil {
		return nil
	}
	var err error
	if Q.fh, err = os.Open(Q.Dir); err != nil {
		return err
	}
	if err = syscall.Flock(int(Q.fh.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		Q.Close()
		return fmt.Errorf("cannot flock directory %s - %s (possibly in use by another instance of dirq)", Q.Dir, err)
	}
	dis, err := Q.fh.ReadDir(-1)
	if len(dis) == 0 && err != nil {
		Q.Close()
		return err
	}
	for _, di := range dis {
		if nm := di.Name(); len(nm) == 26+len(ext)+2 && strings.HasSuffix(nm, ext+".y") {
			_ = os.Rename(filepath.Join(Q.Dir, nm), filepath.Join(Q.Dir, nm[:len(nm)-2]))
		}
	}
	return nil
}

var ErrEmpty = errors.New("queue is empty")

// Dequeue all the incoming messages, continuously.
//
// Calls dequeueOne when a new message arrives (based on notification).
func (Q *Queue) Watch(ctx context.Context, f func(context.Context, []byte) error) error {
	Q.mu.Lock()
	err := Q.lock()
	Q.mu.Unlock()
	if err != nil {
		return err
	}
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
				_ = Q.DequeueOne(ctx, f, ei.Path())
			}
		}
	}
}

func (Q *Queue) DequeueOne(ctx context.Context, f func(context.Context, []byte) error, fn string) error {
	Q.mu.Lock()
	defer Q.mu.Unlock()
	return Q.dequeueOne(ctx, f, fn)
}

func (Q *Queue) dequeueOne(ctx context.Context, f func(context.Context, []byte) error, fn string) error {
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
