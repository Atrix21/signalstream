package notification

import "github.com/google/uuid"

type AlertData struct {
	UserID              uuid.UUID
	StrategyID          uuid.UUID
	StrategyDescription string
	Recipient           string
	EventID             string
	EventTitle          string
	EventSource         string
	EventURL            string
	EventContent        string
	SimilarityScore     float32
	SimilarityThreshold float32
}
