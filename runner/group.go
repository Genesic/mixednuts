package runner

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"
)

type runner struct {
	fn         RunFunc
	shutdownFn ShutdownFunc
}

type RunFunc func() error
type ShutdownFunc func(context.Context) error

type Group struct {
	runners []runner
	errors  []error

	shutdownTimeout time.Duration
}

func NewGroup(shutdownTimeout time.Duration) *Group {
	if shutdownTimeout < 2*time.Second {
		shutdownTimeout = 2 * time.Second
	}
	return &Group{
		shutdownTimeout: shutdownTimeout,
	}
}

func (g *Group) Register(fn RunFunc, shutdownFn ShutdownFunc) *Group {
	g.runners = append(g.runners, runner{
		fn:         fn,
		shutdownFn: shutdownFn,
	})

	return g
}

// Wait starts runners and returns if
//  1. The process receives SIGTERM or SIGINT.
//  2. One of the runner returns with non-nil error.
//  3. The input context is done.
//
// When this function returns, the process no longer masks signal handling.
func (g *Group) Wait(ctx context.Context) *Group {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errCh := make(chan error, len(g.runners))
	for _, executor := range g.runners {
		exec := executor
		go func() {
			if err := exec.fn(); err != nil {
				errCh <- err
			}
		}()
	}

	// wait for either signal marks the context as done, or one of the runner
	// returns with error.
	select {
	case err := <-errCh:
		g.errors = append(g.errors, err)
		return g
	case <-ctx.Done():
		return g
	}
}

func (g *Group) Errors() error {
	if len(g.errors) != 0 {
		return ErrGroup{Errs: g.errors}
	}
	return nil
}

// Shutdown runs shutdown handlers provided by each runner during registration,
// and exits if the shutdown timeout reached or all of the shutdown handler
// return. It returns non-nil error if one of the shutdown handlers returns
// error, or if the timeout is reached. The returned error type is ErrGroup
// (if it is non-nil).
func (g *Group) Shutdown() error {
	shutDownCtx, cancel := context.WithTimeout(context.Background(), g.shutdownTimeout)
	defer cancel()

	var shutdownCount int
	shutdownErrCh := make(chan error, len(g.runners))

	// Setting inputShutdownCtx shorter than shutDownCtx so that runner's shutdownFunc can return earlier
	// than the deadline of shutDownCtx if it respects the deadline of inputShutdownCtx.
	inputShutdownCtx, cancel := context.WithTimeout(shutDownCtx, g.shutdownTimeout-time.Second)
	defer cancel()
	for _, executor := range g.runners {
		exec := executor
		if exec.shutdownFn == nil {
			continue
		}

		shutdownCount += 1
		go func() {
			err := exec.shutdownFn(inputShutdownCtx)
			shutdownErrCh <- err
		}()
	}

	isTimeout, shutdownNilOrErrs := shutdown(shutDownCtx, shutdownErrCh, shutdownCount)

	var groupedErr ErrGroup
	if isTimeout {
		groupedErr.Errs = append(groupedErr.Errs, errors.New("shutdown timeout exceeded"))
	}
	for _, err := range shutdownNilOrErrs {
		if err == nil {
			continue
		}
		groupedErr.Errs = append(groupedErr.Errs, err)
	}
	if len(groupedErr.Errs) == 0 {
		return nil
	}
	return groupedErr
}

func shutdown(shutDownCtx context.Context, shutdownErrCh chan error, shutdownCount int) (bool, []error) {
	var shutdownNilOrErrs []error
	for {
		select {
		case err := <-shutdownErrCh:
			shutdownNilOrErrs = append(shutdownNilOrErrs, err)
		case <-shutDownCtx.Done():
			return true, shutdownNilOrErrs
		}

		if len(shutdownNilOrErrs) == shutdownCount {
			return false, shutdownNilOrErrs
		}
	}
}
