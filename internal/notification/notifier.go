package notification

import (
	"fmt"
	"log/slog"
)

// Notifier sends alert notifications through a delivery channel.
type Notifier interface {
	Send(data AlertData) error
}

// LogNotifier logs alerts to structured output.
type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) Send(data AlertData) error {
	slog.Info("alert triggered",
		"strategy_id", data.StrategyID,
		"recipient", data.Recipient,
		"event_title", data.EventTitle,
		"event_source", data.EventSource,
		"similarity_score", fmt.Sprintf("%.4f", data.SimilarityScore),
		"threshold", fmt.Sprintf("%.2f", data.SimilarityThreshold),
	)
	return nil
}

// MultiNotifier fans out alert delivery to multiple notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) Send(data AlertData) error {
	var firstErr error
	for _, n := range m.notifiers {
		if err := n.Send(data); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
