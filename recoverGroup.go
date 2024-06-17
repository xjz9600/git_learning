package main

import (
	"context"
	"fmt"
	"sync"
)

type token struct{}

type Options func(group *RecoverGroup)

func WithRecoverFunc(f func(info any)) Options {
	return func(group *RecoverGroup) {
		group.recoverFunc = f
	}
}

type RecoverGroup struct {
	cancel func(error)

	wg sync.WaitGroup

	sem chan token

	errOnce sync.Once

	err error
	// 恢复调用方法
	recoverFunc func(any)
}

func (g *RecoverGroup) done() {
	if g.sem != nil {
		<-g.sem
	}
	g.wg.Done()
}

func withCancelCause(parent context.Context) (context.Context, func(error)) {
	return context.WithCancelCause(parent)
}

func WithContext(ctx context.Context, opts ...Options) (*RecoverGroup, context.Context) {
	ctx, cancel := withCancelCause(ctx)
	r := &RecoverGroup{cancel: cancel}
	for _, opt := range opts {
		opt(r)
	}
	return r, ctx
}

func (g *RecoverGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel(g.err)
	}
	return g.err
}

func (g *RecoverGroup) Go(f func() error) {
	if g.sem != nil {
		g.sem <- token{}
	}
	g.wg.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				if g.recoverFunc != nil {
					g.recoverFunc(err)
				}
			}
			g.done()
		}()
		if err := f(); err != nil {
			// 只执行一次
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel(g.err)
				}
			})
		}
	}()
}

func (g *RecoverGroup) TryGo(f func() error) bool {
	if g.sem != nil {
		select {
		case g.sem <- token{}:
			// Note: this allows barging iff channels in general allow barging.
		default:
			return false
		}
	}

	g.wg.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				if g.recoverFunc != nil {
					g.recoverFunc(err)
				}
			}
			g.done()
		}()
		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel(g.err)
				}
			})
		}
	}()
	return true
}

func (g *RecoverGroup) SetLimit(n int) {
	if n < 0 {
		g.sem = nil
		return
	}
	if len(g.sem) != 0 {
		panic(fmt.Errorf("errgroup: modify limit while %v goroutines in the group are still active", len(g.sem)))
	}
	g.sem = make(chan token, n)
}
