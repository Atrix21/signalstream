package notification

import (
	"fmt"
	"log"
)

// Notifier now uses the decoupled AlertData struct.
type Notifier interface {
	Send(data AlertData) error
}

type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

// Send now takes the AlertData struct.
func (n *LogNotifier) Send(data AlertData) error {
	alertMsg := fmt.Sprintf(
		"\n"+
			"========================= 🚨 ALERT TRIGGERED 🚨 ========================\n"+
			"Strategy: %s\n"+
			"Recipient: %s\n"+
			"Event: %s\n"+
			"Source: %s\n"+
			"URL: %s\n"+
			"Similarity Score: %.4f (Threshold: %.2f)\n"+
			"========================================================================\n",
		data.StrategyDescription,
		data.Recipient,
		data.EventTitle,
		data.EventSource,
		data.EventURL,
		data.SimilarityScore,
		data.SimilarityThreshold,
	)
	log.Println(alertMsg)
	return nil
}
