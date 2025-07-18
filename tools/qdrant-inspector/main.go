package main

import (
	"context"
	"fmt"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

func main() {
	log.Println("Connecting to Qdrant...")
	qClient, err := qdrant.NewClient(&qdrant.Config{Host: "localhost", Port: 6334})
	if err != nil {
		log.Fatalf("Failed to create Qdrant client: %v", err)
	}

	collectionName := "financial_events"
	log.Printf("Fetching details for collection: '%s'", collectionName)

	// Check if the collection exists
	exists, err := qClient.CollectionExists(context.Background(), collectionName)
	if err != nil {
		log.Fatalf("Failed to check if collection exists: %v", err)
	}
	if !exists {
		log.Fatalf("Collection '%s' not found.", collectionName)
	}

	// Get detailed collection info
	info, err := qClient.GetCollectionInfo(context.Background(), collectionName)
	if err != nil {
		log.Fatalf("Failed to get collection info: %v", err)
	}

	fmt.Println("\n--- QDRANT COLLECTION DETAILS ---")
	fmt.Printf("Collection Name: %s\n", collectionName)
	fmt.Printf("Points Count:    %d\n", info.GetPointsCount())
	fmt.Printf("Vector Size:     %d\n", info.GetConfig().GetParams().GetVectorsConfig().GetParams().GetSize())
	fmt.Printf("Distance Metric: %s\n", info.GetConfig().GetParams().GetVectorsConfig().GetParams().GetDistance())
	fmt.Println("---------------------------------")
}
