package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// Config controls retry behavior.
type Config struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// DefaultConfig returns a sensible default: 3 retries, 500ms base, 10s max, 2x multiplier.
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   10 * time.Second,
		Multiplier: 2.0,
	}
}

// Do executes fn with exponential backoff and jitter. It respects context cancellation
// between retry attempts. The operation string is used for error wrapping.
func Do(ctx context.Context, cfg Config, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return fmt.Errorf("%s: %w (after %d attempts, last: %v)", operation, err, attempt, lastErr)
			}
			return fmt.Errorf("%s: %w", operation, err)
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt == cfg.MaxRetries {
			break
		}

		delay := time.Duration(float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt)))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		// Jitter: randomize between 50% and 100% of calculated delay.
		jitter := time.Duration(float64(delay) * (0.5 + rand.Float64()*0.5))

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w (during backoff, last: %v)", operation, ctx.Err(), lastErr)
		case <-time.After(jitter):
		}
	}

	return fmt.Errorf("%s: all %d retries exhausted: %w", operation, cfg.MaxRetries+1, lastErr)
}
