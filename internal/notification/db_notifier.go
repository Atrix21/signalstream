package notification

import (
	"context"
	"log/slog"

	"github.com/Atrix21/signalstream/internal/database"
	"github.com/Atrix21/signalstream/internal/sse"
)

// DatabaseNotifier persists alerts to PostgreSQL and optionally broadcasts via SSE.
type DatabaseNotifier struct {
	db     *database.DB
	broker *sse.Broker
}

// NewDatabaseNotifier creates a notifier that saves alerts to the database and broadcasts via SSE.
func NewDatabaseNotifier(db *database.DB, broker *sse.Broker) *DatabaseNotifier {
	return &DatabaseNotifier{db: db, broker: broker}
}

func (n *DatabaseNotifier) Send(data AlertData) error {
	alert := &database.Alert{
		UserID:          data.UserID,
		StrategyID:      data.StrategyID,
		EventID:         data.EventID,
		Title:           data.EventTitle,
		Content:         data.StrategyDescription,
		URL:             data.EventURL,
		SimilarityScore: float64(data.SimilarityScore),
		IsRead:          false,
	}

	if err := n.db.CreateAlert(context.Background(), alert); err != nil {
		slog.Error("failed to persist alert",
			"user_id", data.UserID,
			"event_title", data.EventTitle,
			"error", err,
		)
		return err
	}

	slog.Info("alert persisted",
		"alert_id", alert.ID,
		"user_id", data.UserID,
		"event_title", data.EventTitle,
	)

	// Broadcast to connected SSE clients.
	if n.broker != nil {
		n.broker.Broadcast(data.UserID.String(), alert)
	}

	return nil
}
