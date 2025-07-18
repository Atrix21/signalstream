package alerter


type Strategy struct {
	ID          string
	Description string 
	OwnerEmail  string 


	SearchQuery string


	SourceFilter  []string 
	TickersFilter []string 

	SimilarityThreshold float32
}
