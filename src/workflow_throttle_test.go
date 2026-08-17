package main

import (
	"context"
	"testing"
	"time"
)

// TestAcquireThrottleCapsConcurrency verifies that parallel nodes sharing a
// profile with MaxConcurrency=2 never have more than 2 slots held at once, and
// that a 3rd caller blocks until a slot is released.
func TestAcquireThrottleCapsConcurrency(t *testing.T) {
	e := &WorkflowExecutor{}
	cfg := APIConfig{Provider: "openai", Model: "m", MaxConcurrency: 2}

	r1 := e.acquireThrottle(context.Background(), "smart", cfg)
	r2 := e.acquireThrottle(context.Background(), "smart", cfg)

	// Third acquire must block (both slots taken). Use a short timeout to prove it.
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	got := make(chan struct{})
	go func() {
		_ = e.acquireThrottle(ctx, "smart", cfg)
		close(got)
	}()
	select {
	case <-got:
		t.Fatal("third acquire returned before a slot was released; concurrency not capped")
	case <-time.After(40 * time.Millisecond):
		// expected: still blocked
	}

	// Releasing one slot should unblock the waiter.
	r1()
	select {
	case <-got:
		// good
	case <-time.After(200 * time.Millisecond):
		t.Fatal("waiter did not unblock after release")
	}
	r2()
}

// TestAcquireThrottleUnlimited ensures MaxConcurrency<=0 means no queuing.
func TestAcquireThrottleUnlimited(t *testing.T) {
	e := &WorkflowExecutor{}
	cfg := APIConfig{MaxConcurrency: 0}
	// Many acquires should return immediately without blocking.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			r := e.acquireThrottle(context.Background(), "p", cfg)
			defer r()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("unlimited throttle appears to block")
	}
}

// TestAcquireThrottleBucketsArePerProfile verifies two different profiles get
// independent semaphores (so a slow profile doesn't block an unrelated one).
func TestAcquireThrottleBucketsArePerProfile(t *testing.T) {
	e := &WorkflowExecutor{}
	a := APIConfig{MaxConcurrency: 1}
	b := APIConfig{MaxConcurrency: 1}
	ra := e.acquireThrottle(context.Background(), "profile-a", a)
	defer ra()

	// A different profile with limit 1 must still acquire immediately even
	// though profile-a's single slot is held.
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	got := make(chan struct{})
	go func() {
		rb := e.acquireThrottle(ctx, "profile-b", b)
		rb()
		close(got)
	}()
	select {
	case <-got:
	case <-time.After(40 * time.Millisecond):
		t.Fatal("different profiles should not share a throttle bucket")
	}
}
