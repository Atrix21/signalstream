package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), DefaultConfig(), "test-op", func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	cfg := Config{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
	}

	calls := 0
	err := Do(context.Background(), cfg, "test-op", func() error {
		calls++
		if calls < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_AllRetriesExhausted(t *testing.T) {
	cfg := Config{
		MaxRetries: 2,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   5 * time.Millisecond,
		Multiplier: 2.0,
	}

	sentinel := errors.New("persistent failure")
	calls := 0
	err := Do(context.Background(), cfg, "test-op", func() error {
		calls++
		return sentinel
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got: %v", err)
	}
	// 1 initial + 2 retries = 3 total calls
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ContextCancelledBeforeAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Do(ctx, DefaultConfig(), "test-op", func() error {
		t.Fatal("function should not be called")
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestDo_ContextCancelledDuringBackoff(t *testing.T) {
	cfg := Config{
		MaxRetries: 5,
		BaseDelay:  1 * time.Second, // Long delay to ensure context cancels during it
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	calls := 0
	err := Do(ctx, cfg, "test-op", func() error {
		calls++
		return errors.New("fail")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls < 1 {
		t.Fatal("expected at least 1 call")
	}
}

func TestDo_MaxDelayRespected(t *testing.T) {
	cfg := Config{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   150 * time.Millisecond,
		Multiplier: 10.0, // Would produce 1s, 10s delays without cap
	}

	start := time.Now()
	calls := 0
	_ = Do(context.Background(), cfg, "test-op", func() error {
		calls++
		return errors.New("fail")
	})

	elapsed := time.Since(start)
	// 3 delays, each capped at ~150ms (with jitter). Should be well under 1s.
	if elapsed > 2*time.Second {
		t.Fatalf("max delay not respected, total elapsed: %v", elapsed)
	}
}

func TestDo_ZeroRetries(t *testing.T) {
	cfg := Config{
		MaxRetries: 0,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   5 * time.Millisecond,
		Multiplier: 2.0,
	}

	calls := 0
	err := Do(context.Background(), cfg, "test-op", func() error {
		calls++
		return errors.New("fail")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call with 0 retries, got %d", calls)
	}
}
