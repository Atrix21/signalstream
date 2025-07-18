package notification

type AlertData struct {
	StrategyDescription string
	Recipient           string
	EventTitle          string
	EventSource         string
	EventURL            string
	SimilarityScore     float32
	SimilarityThreshold float32
}
