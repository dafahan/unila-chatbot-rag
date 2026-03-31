// Package bm25 implements a BM25 sparse vector index for use with Qdrant's
// native sparse-dense hybrid search. Corpus statistics (vocabulary, document
// frequencies, total chunk count, average document length) are persisted to a
// JSON file so they survive process restarts.
//
// BM25 parameters: k1 = 1.5, b = 0.75 (standard Robertson et al. defaults).
package bm25

import (
	"encoding/json"
	"math"
	"os"
	"sync"

	"github.com/dafahan/unila-ai/pkg/nlp"
)

const (
	k1 = 1.5
	b  = 0.75
)

// Stats holds all corpus-level data needed to compute BM25 weights.
type Stats struct {
	Vocab  map[string]uint32 `json:"vocab"`   // term → sparse dimension index
	DF     map[string]int    `json:"df"`      // document (chunk) frequency per term
	N      int               `json:"n"`       // total number of chunks in corpus
	AvgDL  float64           `json:"avg_dl"`  // average document (chunk) length in tokens
	NextID uint32            `json:"next_id"` // next free vocab index
}

// Index is a thread-safe BM25 corpus index.
type Index struct {
	mu        sync.RWMutex
	stats     Stats
	statsPath string
}

// Load reads an existing stats file, or returns a fresh empty index if the
// file does not exist yet.
func Load(path string) (*Index, error) {
	idx := &Index{
		statsPath: path,
		stats: Stats{
			Vocab: make(map[string]uint32),
			DF:    make(map[string]int),
		},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return idx, nil
		}
		return nil, err
	}
	return idx, json.Unmarshal(data, &idx.stats)
}

// Save persists the current corpus stats to disk.
func (idx *Index) Save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	data, err := json.Marshal(idx.stats)
	if err != nil {
		return err
	}
	return os.WriteFile(idx.statsPath, data, 0644)
}

// AddChunks updates corpus statistics (vocabulary, DF, N, AvgDL) from a
// slice of raw chunk texts. Must be called before VectorizeDoc for these
// chunks so that IDF values are correct.
func (idx *Index) AddChunks(texts []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	totalTokens := 0
	for _, text := range texts {
		tokens := nlp.Tokenize(text)
		totalTokens += len(tokens)

		seen := make(map[string]bool, len(tokens))
		for _, t := range tokens {
			if !seen[t] {
				idx.stats.DF[t]++
				seen[t] = true
			}
			if _, ok := idx.stats.Vocab[t]; !ok {
				idx.stats.Vocab[t] = idx.stats.NextID
				idx.stats.NextID++
			}
		}
	}

	// Update running average document length
	prevTotal := idx.stats.AvgDL * float64(idx.stats.N)
	idx.stats.N += len(texts)
	if idx.stats.N > 0 {
		idx.stats.AvgDL = (prevTotal + float64(totalTokens)) / float64(idx.stats.N)
	}
}

// VectorizeDoc returns a sparse BM25 vector (indices, values) for a document
// chunk. The weight of each term is: IDF(t) × TF_normalized(t, d).
func (idx *Index) VectorizeDoc(text string) (indices []uint32, values []float32) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	tokens := nlp.Tokenize(text)
	dl := float64(len(tokens))

	tf := make(map[string]int, len(tokens))
	for _, t := range tokens {
		tf[t]++
	}

	for term, freq := range tf {
		termID, ok := idx.stats.Vocab[term]
		if !ok {
			continue
		}
		df := idx.stats.DF[term]
		if df == 0 {
			continue
		}
		idf := math.Log((float64(idx.stats.N)-float64(df)+0.5)/(float64(df)+0.5) + 1)
		avgdl := idx.stats.AvgDL
		if avgdl == 0 {
			avgdl = 1
		}
		tfNorm := (float64(freq) * (k1 + 1)) / (float64(freq) + k1*(1-b+b*dl/avgdl))
		weight := idf * tfNorm

		indices = append(indices, termID)
		values = append(values, float32(weight))
	}
	return
}

// VectorizeQuery returns a sparse IDF-weighted vector for a query string.
// Each unique stemmed query term is weighted by its IDF in the corpus.
func (idx *Index) VectorizeQuery(text string) (indices []uint32, values []float32) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	tokens := nlp.Tokenize(text)
	seen := make(map[string]bool, len(tokens))

	for _, term := range tokens {
		if seen[term] {
			continue
		}
		seen[term] = true

		termID, ok := idx.stats.Vocab[term]
		if !ok {
			continue
		}
		df := idx.stats.DF[term]
		if df == 0 {
			continue
		}
		idf := math.Log((float64(idx.stats.N)-float64(df)+0.5)/(float64(df)+0.5) + 1)
		indices = append(indices, termID)
		values = append(values, float32(idf))
	}
	return
}
