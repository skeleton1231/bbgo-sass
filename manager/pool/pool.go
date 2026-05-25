package pool

import (
	"context"

	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

type Pool struct {
	pool *ants.Pool
	wg   sync.WaitGroup
}

func New(size int) *Pool {
	p, err := ants.NewPool(size, ants.WithPreAlloc(true))
	if err != nil {
		panic("pool: " + err.Error())
	}
	return &Pool{pool: p}
}

func (p *Pool) Submit(fn func()) error {
	p.wg.Add(1)
	return p.pool.Submit(func() {
		defer p.wg.Done()
		fn()
	})
}

func (p *Pool) SubmitWithTimeout(fn func(), timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return p.SubmitWithContext(ctx, fn)
}

func (p *Pool) SubmitWithContext(ctx context.Context, fn func()) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	errCh := make(chan error, 1)
	p.wg.Add(1)
	go func() {
		err := p.pool.Submit(func() {
			defer p.wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				fn()
			}
		})
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) Running() int {
	return p.pool.Running()
}

func (p *Pool) Release() {
	p.pool.Release()
}

func (p *Pool) Size() int {
	return p.pool.Cap()
}

func (p *Pool) Waiting() int {
	return p.pool.Waiting()
}
