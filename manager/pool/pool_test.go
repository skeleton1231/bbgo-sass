package pool

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_Submit(t *testing.T) {
	p := New(4)
	defer p.Release()

	var count atomic.Int32
	for i := 0; i < 20; i++ {
		if err := p.Submit(func() {
			count.Add(1)
		}); err != nil {
			t.Fatalf("submit: %v", err)
		}
	}
	p.Wait()

	if got := count.Load(); got != 20 {
		t.Fatalf("expected 20, got %d", got)
	}
}

func TestPool_MaxConcurrency(t *testing.T) {
	const max = 3
	p := New(max)
	defer p.Release()

	var running atomic.Int32
	var maxObserved atomic.Int32

	for i := 0; i < 10; i++ {
		if err := p.Submit(func() {
			cur := running.Add(1)
			for {
				old := maxObserved.Load()
				if cur <= old || maxObserved.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			running.Add(-1)
		}); err != nil {
			t.Fatalf("submit: %v", err)
		}
	}
	p.Wait()

	if got := maxObserved.Load(); got > max {
		t.Fatalf("max concurrency exceeded: observed %d, limit %d", got, max)
	}
}

func TestPool_SubmitWithContext_Cancelled(t *testing.T) {
	p := New(1)
	defer p.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.SubmitWithContext(ctx, func() {})
	if err == nil {
		t.Fatal("expected context cancelled error, got nil")
	}
}

func TestPool_SubmitWithContext_ExpiredWhileWaiting(t *testing.T) {
	p := New(1)
	defer p.Release()

	started := make(chan struct{})
	firstDone := make(chan struct{})

	err := p.Submit(func() {
		close(started)
		time.Sleep(200 * time.Millisecond)
		close(firstDone)
	})
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}

	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = p.SubmitWithContext(ctx, func() {})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	<-firstDone
	p.Wait()
}

func TestPool_Running(t *testing.T) {
	p := New(4)
	defer p.Release()

	if p.Running() != 0 {
		t.Fatalf("expected 0 running, got %d", p.Running())
	}

	done := make(chan struct{})
	_ = p.Submit(func() {
		<-done
	})

	time.Sleep(20 * time.Millisecond)
	if p.Running() == 0 {
		t.Fatal("expected at least 1 running task")
	}
	close(done)
	p.Wait()
}

func TestPool_ReleaseIdempotent(t *testing.T) {
	p := New(2)
	p.Release()
	p.Release()
}

func TestPool_Size(t *testing.T) {
	p := New(5)
	defer p.Release()
	if p.Size() != 5 {
		t.Errorf("expected size 5, got %d", p.Size())
	}
}

func TestPool_Waiting(t *testing.T) {
	p := New(1)
	defer p.Release()

	started := make(chan struct{})
	_ = p.Submit(func() {
		close(started)
		time.Sleep(300 * time.Millisecond)
	})
	<-started

	_ = p.Submit(func() {})
	time.Sleep(100 * time.Millisecond)

	w := p.Waiting()
	if w < 0 {
		t.Errorf("expected waiting >= 0, got %d", w)
	}
	p.Wait()
}

func TestPool_SubmitWithTimeout(t *testing.T) {
	p := New(1)
	defer p.Release()

	err := p.SubmitWithTimeout(func() {
		time.Sleep(10 * time.Millisecond)
	}, 5*time.Second)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	p.Wait()
}
