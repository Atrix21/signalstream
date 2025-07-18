package alerter

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	openai "github.com/sashabaranov/go-openai"

	"github.com/Atrix21/signalstream/internal/config"
	"github.com/Atrix21/signalstream/internal/notification"
	"github.com/Atrix21/signalstream/internal/platform"
)

const (
	qdrantCollectionName = "financial_events"
	embeddingModel       = openai.SmallEmbedding3
)

type Service struct {
	qdrantClient *qdrant.Client
	openAIClient *openai.Client
	notifier     notification.Notifier
	strategies   []Strategy
}

func NewService(cfg config.AppConfig, notifier notification.Notifier, strategies []Strategy) (*Service, error) {
	qClient, err := qdrant.NewClient(&qdrant.Config{Host: "localhost", Port: 6334})
	if err != nil {
		return nil, err
	}
	openAIClient := openai.NewClient(cfg.OpenAIAPIKey)

	return &Service{
		qdrantClient: qClient,
		openAIClient: openAIClient,
		notifier:     notifier,
		strategies:   strategies,
	}, nil
}

func (s *Service) CheckEventAgainstStrategies(ctx context.Context, event platform.NormalizedEvent) {
	log.Printf("[ALERTER_DEBUG] Received event: ID=%s, Title='%s', Source=%s, Tickers=%v", event.ID, event.Title, event.Source, event.Tickers)
	for _, strategy := range s.strategies {
		log.Printf("[ALERTER_DEBUG] -> Checking against strategy: '%s'", strategy.Description)

		if !matchesSimpleFilters(event, strategy) {
			log.Printf("[ALERTER_DEBUG] |-> Event REJECTED by simple filters.")
			continue // Skip to the next strategy.
		}
		log.Printf("[ALERTER_DEBUG] |-> Event PASSED simple filters.")

		queryVector, err := s.createEmbedding(ctx, strategy.SearchQuery)
		if err != nil {
			log.Printf("[ALERTER_DEBUG] |-> ERROR creating embedding for strategy query: %v", err)
			continue
		}
		log.Printf("[ALERTER_DEBUG] |-> Generated strategy query embedding successfully.")

		searchResult, err := s.querySimilar(ctx, queryVector, strategy)
		if err != nil {
			log.Printf("[ALERTER_DEBUG] |-> ERROR during Qdrant query: %v", err)
			continue
		}
		log.Printf("[ALERTER_DEBUG] |-> Qdrant query returned %d results.", len(searchResult))

		matchFound := false
		for _, point := range searchResult {
			pointID := point.GetId().GetUuid()
			eventUUID := uuid.NewSHA1(uuid.Nil, []byte(event.ID)).String()

			log.Printf("[ALERTER_DEBUG] | |-> Comparing result ID '%s' with event UUID '%s'. Score: %.4f", pointID, eventUUID, point.Score)
			if pointID == eventUUID {
				log.Printf("[ALERTER_DEBUG] | | |-> IDs MATCH.")
				if point.Score >= strategy.SimilarityThreshold {
					log.Printf("[ALERTER_DEBUG] | | | |-> Score threshold PASSED (%.4f >= %.2f). TRIGGERING ALERT.", point.Score, strategy.SimilarityThreshold)
					alert := notification.AlertData{
						StrategyDescription: strategy.Description,
						Recipient:           strategy.OwnerEmail,
						EventTitle:          event.Title,
						EventSource:         event.Source,
						EventURL:            event.ContentURL,
						SimilarityScore:     point.Score,
						SimilarityThreshold: strategy.SimilarityThreshold,
					}
					s.notifier.Send(alert)
					matchFound = true
					break
				} else {
					log.Printf("[ALERTER_DEBUG] | | | |-> Score threshold FAILED (%.4f < %.2f).", point.Score, strategy.SimilarityThreshold)
				}
			}
		}
		if !matchFound {
			log.Printf("[ALERTER_DEBUG] |-> Event was not found in top search results or did not meet score threshold.")
		}
	}
}

func (s *Service) querySimilar(ctx context.Context, vector []float32, strategy Strategy) ([]*qdrant.ScoredPoint, error) {
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{},
	}

	if len(strategy.SourceFilter) > 0 {
		var sourceConditions []*qdrant.Condition
		for _, source := range strategy.SourceFilter {
			sourceConditions = append(sourceConditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "source",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: source},
						},
					},
				},
			})
		}
		var sourceFilterCond *qdrant.Condition
		if len(sourceConditions) == 1 {
			sourceFilterCond = sourceConditions[0]
		} else {
			sourceFilterCond = &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Filter{
					Filter: &qdrant.Filter{
						Should: sourceConditions,
					},
				},
			}
		}
		filter.Must = append(filter.Must, sourceFilterCond)
	}

	if len(strategy.TickersFilter) > 0 {
		var tickerConditions []*qdrant.Condition
		for _, ticker := range strategy.TickersFilter {
			tickerConditions = append(tickerConditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "tickers",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: ticker},
						},
					},
				},
			})
		}
		var tickerFilterCond *qdrant.Condition
		if len(tickerConditions) == 1 {
			tickerFilterCond = tickerConditions[0]
		} else {
			tickerFilterCond = &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Filter{
					Filter: &qdrant.Filter{
						Should: tickerConditions,
					},
				},
			}
		}
		filter.Must = append(filter.Must, tickerFilterCond)
	}

	limit := uint64(10)
	queryResponse, err := s.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: qdrantCollectionName,
		Query:          qdrant.NewQuery(vector...),
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	return queryResponse, nil
}

func matchesSimpleFilters(event platform.NormalizedEvent, strategy Strategy) bool {
	log.Printf("[ALERTER_DEBUG] | |-> Evaluating simple filters...")
	if len(strategy.SourceFilter) > 0 {
		sourceMatch := false
		for _, source := range strategy.SourceFilter {
			if event.Source == source {
				sourceMatch = true
				break
			}
		}
		if !sourceMatch {
			log.Printf("[ALERTER_DEBUG] | | |-> REJECTED on source. Event source '%s' not in strategy filter %v.", event.Source, strategy.SourceFilter)
			return false
		}
		log.Printf("[ALERTER_DEBUG] | | |-> PASSED source filter.")
	}

	if len(strategy.TickersFilter) > 0 {
		if len(event.Tickers) == 0 {
			log.Printf("[ALERTER_DEBUG] | | |-> REJECTED on ticker. Strategy requires tickers, but event has none.")
			return false
		}
		tickerMatch := false
		for _, eventTicker := range event.Tickers {
			for _, strategyTicker := range strategy.TickersFilter {
				if eventTicker == strategyTicker {
					tickerMatch = true
					break
				}
			}
			if tickerMatch {
				break
			}
		}
		if !tickerMatch {
			log.Printf("[ALERTER_DEBUG] | | |-> REJECTED on ticker. Event tickers %v not in strategy filter %v.", event.Tickers, strategy.TickersFilter)
			return false
		}
		log.Printf("[ALERTER_DEBUG] | | |-> PASSED ticker filter.")
	}
	return true
}
func (s *Service) createEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{Input: []string{text}, Model: embeddingModel}
	resp, err := s.openAIClient.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Data[0].Embedding, nil
}
