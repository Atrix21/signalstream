package sec

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
)

const cikMapURL = "https://www.sec.gov/files/company_tickers.json"

// CIKEntry represents a single entry in the SEC's company_tickers.json.
type CIKEntry struct {
	CIK    int    `json:"cik_str"`
	Ticker string `json:"ticker"`
	Title  string `json:"title"`
}

// CIKMap holds the mapping from CIK numbers to ticker symbols.
type CIKMap struct {
	mu   sync.RWMutex
	data map[string]string
}

var (
	once   sync.Once
	cikMap *CIKMap
)

// GetCIKMap returns a singleton instance of the CIKMap, initializing it on first use.
func GetCIKMap() *CIKMap {
	once.Do(func() {
		slog.Info("initializing CIK-to-ticker mapping")
		m := &CIKMap{data: make(map[string]string)}

		client := &http.Client{}
		req, err := http.NewRequest("GET", cikMapURL, nil)
		if err != nil {
			slog.Error("failed to create CIK map request", "error", err)
			cikMap = m
			return
		}
		req.Header.Set("User-Agent", "SignalStream App v0.1 user@example.com")

		resp, err := client.Do(req)
		if err != nil {
			slog.Error("failed to download CIK map", "error", err)
			cikMap = m
			return
		}
		defer resp.Body.Close()

		var allEntries map[string]CIKEntry
		if err := json.NewDecoder(resp.Body).Decode(&allEntries); err != nil {
			slog.Error("failed to decode CIK map JSON", "error", err)
			cikMap = m
			return
		}

		for _, entry := range allEntries {
			m.data[strconv.Itoa(entry.CIK)] = entry.Ticker
		}

		cikMap = m
		slog.Info("CIK-to-ticker mapping initialized", "entries", len(m.data))
	})
	return cikMap
}

// Ticker returns the ticker symbol for a given CIK string.
func (m *CIKMap) Ticker(cik string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ticker, found := m.data[cik]
	return ticker, found
}
