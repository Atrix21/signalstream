package platform

import "time"

type NormalizedEvent struct {
	ID          string
	Source      string
	Tickers     []string
	Title       string
	ContentURL  string
	RawText     string
	PublishedAt time.Time
	IngestedAt  time.Time
}
