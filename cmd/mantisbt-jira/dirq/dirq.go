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
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/google/renameio/v2"
	"github.com/oklog/ulid/v2"
	"github.com/rjeczalik/notify"
	"golang.org/x/time/rate"
)

const ext = ".dirq-item.dat"

type Queue struct {
	fh      *os.File
	limiter *rate.Limiter
	Dir     string

	mu sync.Mutex
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
	return &Queue{Dir: dir, limiter: rate.NewLimiter(1, 1)}, nil
}

// Enqueue a message.
//
// Does not lock (not needed).
func (Q *Queue) Enqueue(p []byte) error {
	fn := filepath.Join(
		Q.Dir,
		ulid.MustNew(ulid.Now(), ulid.DefaultEntropy()).String()+ext)
	slog.Debug("Enqueue", "file", fn)
	return renameio.WriteFile(fn, p, 0400)
}

// DequeueOne will call f on the first dequeueable message.
// The message will be deleted iff f returns nil,
// otherwise it remains in the queue.
func (Q *Queue) Dequeue(ctx context.Context, f func(context.Context, []byte) error) error {
	if err := ctx.Err(); err != nil {
		slog.Error("Dequeue", "error", err)
		return err
	}
	Q.mu.Lock()
	defer Q.mu.Unlock()
	if err := Q.lock(); err != nil {
		slog.Error("lock", "error", err)
		return err
	}
	dis, err := os.ReadDir(Q.Dir)
	if len(dis) == 0 {
		if err != nil {
			slog.Error("ReadDir", "dir", Q.Dir, "error", err)
		}
		// slog.Debug("empty", "dir", Q.Dir)
		return err
	}
	names := make([]string, 0, len(dis))
	for _, di := range dis {
		nm := di.Name()
		if di.Type().IsRegular() && strings.HasSuffix(nm, ext) &&
			len(nm) == 26+len(ext) {
			names = append(names, nm)
		}
	}
	// slog.Debug("ReadDir2", "names", names)
	if len(names) == 0 {
		return ErrEmpty
	}
	slices.Sort(names)
	for _, nm := range names {
		if err := Q.dequeueOne(ctx, f, filepath.Join(Q.fh.Name(), nm)); err != nil {
			slog.Error("dequeueOne", "name", nm, "error", err)
			return err
		}
		slog.Info("dequeueOne", "name", nm)
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

	evts := make(chan error, 1)
	go func() {
		for err := range evts {
			if err != nil {
				slog.Warn("evts EXIT", "error", err)
				return
			}
			if err = Q.Dequeue(ctx, f); err != nil {
				slog.Error("Dequeue", "error", err)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ei, ok := <-c:
			if !ok {
				return ctx.Err()
			}
			bn := filepath.Base(ei.Path())
			slog.Debug("notify", "path", ei.Path(), "length", len(bn)-len(ext))
			if len(bn) == 26+len(ext) && strings.HasSuffix(bn, ext) {
				if err := Q.limiter.Wait(ctx); err != nil {
					select {
					case evts <- err:
					default:
					}
					return err
				}
				select {
				case evts <- nil:
				default:
				}
			}
		}
	}
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
