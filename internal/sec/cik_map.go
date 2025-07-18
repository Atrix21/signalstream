package sec

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
)

const cikMapURL = "https://www.sec.gov/files/company_tickers.json"

// CIKEntry represents the structure of each entry in the SEC's JSON data.
type CIKEntry struct {
	CIK    int    `json:"cik_str"`
	Ticker string `json:"ticker"`
	Title  string `json:"title"`
}

// CIKMap holds the mapping from CIK numbers to Ticker symbols.
type CIKMap struct {
	mu   sync.RWMutex
	Data map[string]string
}

var (
	once   sync.Once
	cikMap *CIKMap
)

// GetCIKMap returns a singleton instance of the CIKMap, initializing it on first use.
func GetCIKMap() *CIKMap {
	once.Do(func() {
		log.Println("Initializing CIK to Ticker mapping...")
		m := &CIKMap{
			Data: make(map[string]string),
		}

		// We use a custom client with a User-Agent, just like in the poller.
		client := &http.Client{}
		req, err := http.NewRequest("GET", cikMapURL, nil)
		if err != nil {
			log.Printf("Failed to create request for CIK map: %v", err)
			cikMap = m // return empty map
			return
		}
		req.Header.Set("User-Agent", "SignalStream App v0.1 user@example.com")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Failed to download CIK map: %v", err)
			cikMap = m
			return
		}
		defer resp.Body.Close()

		var allEntries map[string]CIKEntry
		if err := json.NewDecoder(resp.Body).Decode(&allEntries); err != nil {
			log.Printf("Failed to decode CIK map JSON: %v", err)
			cikMap = m
			return
		}

		// The JSON is structured as {"0": entry, "1": entry}. We need to transform it.
		for _, entry := range allEntries {
			// Convert the integer CIK to a string for mapping.
			m.Data[strconv.Itoa(entry.CIK)] = entry.Ticker
		}

		cikMap = m
		log.Printf("CIK to Ticker mapping initialized with %d entries.", len(m.Data))
	})
	return cikMap
}

// Ticker returns the ticker symbol for a given CIK string.
func (m *CIKMap) Ticker(cik string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ticker, found := m.Data[cik]
	return ticker, found
}
