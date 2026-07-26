package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPollStepsResetsIntervalAfterSlowPoll(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var active int32
	var calls int32

	done := make(chan struct{})
	go func() {
		defer close(done)
		runPollSteps(ctx, time.Millisecond, []pollStep{func(context.Context) error {
			if atomic.AddInt32(&active, 1) != 1 {
				t.Error("poll step overlapped itself")
			}
			atomic.AddInt32(&calls, 1)
			select {
			case entered <- struct{}{}:
			default:
			}
			<-release
			atomic.AddInt32(&active, -1)
			return nil
		}})
	}()

	<-entered
	time.Sleep(10 * time.Millisecond)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls while first poll is blocked = %d, want 1", got)
	}
	close(release)
	<-entered
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runPollSteps did not stop after cancellation")
	}
}
