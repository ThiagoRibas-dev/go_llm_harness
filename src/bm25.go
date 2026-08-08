package main

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BM25Document represents a single indexed text document
type BM25Document struct {
	Path      string
	TermFreqs map[string]int
	DocLength int
}

// BM25Engine maintains the document corpus indices and score parameters
type BM25Engine struct {
	K1        float64 // Term frequency saturation (usually 1.2 - 2.0)
	B         float64 // Length normalization (usually 0.75)
	Documents []BM25Document
	DocFreqs  map[string]int // Number of documents containing each term
	TotalDocs int
	AvgDocLen float64
}

// SearchResult represents a single scored query result
type SearchResult struct {
	Path  string  `json:"path"`
	Score float64 `json:"score"`
}

// NewBM25Engine instantiates a standard-compliant BM25 engine
func NewBM25Engine() *BM25Engine {
	return &BM25Engine{
		K1:        1.5,
		B:         0.75,
		DocFreqs:  make(map[string]int),
		Documents: []BM25Document{},
	}
}

// Tokenize converts a raw string into a clean lowercase term slice
func (e *BM25Engine) Tokenize(text string) []string {
	text = strings.ToLower(text)
	// Replace punctuation with spaces to prevent indexing artifacts
	puncs := []string{".", ",", "!", "?", "\"", "'", "(", ")", "[", "]", "{", "}", "-", "_", ";", ":", "*", "/", "\\", "`", "<", ">", "=", "+"}
	for _, p := range puncs {
		text = strings.ReplaceAll(text, p, " ")
	}
	return strings.Fields(text)
}

// AddDocument processes and indexes a single document's text
func (e *BM25Engine) AddDocument(path, content string) {
	terms := e.Tokenize(content)
	if len(terms) == 0 {
		return
	}

	tf := make(map[string]int)
	uniqueTerms := make(map[string]bool)

	for _, t := range terms {
		tf[t]++
		uniqueTerms[t] = true
	}

	// Update global document frequencies
	for t := range uniqueTerms {
		e.DocFreqs[t]++
	}

	doc := BM25Document{
		Path:      path,
		TermFreqs: tf,
		DocLength: len(terms),
	}

	e.Documents = append(e.Documents, doc)
	e.TotalDocs++

	// Recalculate average document length dynamically
	totalLength := 0
	for _, d := range e.Documents {
		totalLength += d.DocLength
	}
	e.AvgDocLen = float64(totalLength) / float64(e.TotalDocs)
}

// Search calculates BM25 relevance scores and returns the sorted top results
func (e *BM25Engine) Search(query string, limit int) []SearchResult {
	queryTerms := e.Tokenize(query)
	if len(queryTerms) == 0 || e.TotalDocs == 0 {
		return []SearchResult{}
	}

	var results []SearchResult

	for _, doc := range e.Documents {
		score := 0.0

		for _, term := range queryTerms {
			tf := doc.TermFreqs[term]
			if tf == 0 {
				continue
			}

			df := e.DocFreqs[term]
			// Standard BM25 IDF formula with 0.5 smoothing
			idf := math.Log((float64(e.TotalDocs)-float64(df)+0.5)/(float64(df)+0.5) + 1.0)
			if idf < 0 {
				idf = 0.0001 // Prevent negative IDF weighting
			}

			// Standard BM25 Term Frequency Saturation & Length normalization formula
			numerator := float64(tf) * (e.K1 + 1.0)
			denominator := float64(tf) + e.K1*(1.0-e.B+e.B*(float64(doc.DocLength)/e.AvgDocLen))

			score += idf * (numerator / denominator)
		}

		if score > 0.0 {
			results = append(results, SearchResult{
				Path:  doc.Path,
				Score: score,
			})
		}
	}

	// Sort results descending by BM25 score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// IndexDirectory recursively scans and indexes all text/JSON files in a target directory
func (e *BM25Engine) IndexDirectory(rootPath string, ignoreSubDirs []string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Calculate relative path
		rel, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		// Skip ignored subdirectories
		if info.IsDir() {
			for _, ignore := range ignoreSubDirs {
				if strings.Contains(rel, ignore) || strings.HasPrefix(rel, ignore) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Only index common code, text, and JSON logs
		ext := strings.ToLower(filepath.Ext(path))
		isText := ext == ".txt" || ext == ".json" || ext == ".md" || ext == ".py" || ext == ".go" || ext == ".js" || ext == ".ts" || ext == ".html" || ext == ".css" || ext == ".sh" || ext == ".yaml" || ext == ".yml"

		if isText {
			bytes, err := os.ReadFile(path)
			if err == nil {
				e.AddDocument(path, string(bytes))
			}
		}

		return nil
	})
}
