package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/dafahan/unila-ai/internal/domain"
	"github.com/dafahan/unila-ai/pkg/bm25"
	"github.com/dafahan/unila-ai/pkg/config"
)

type IngestionUseCase struct {
	llm  domain.LLMProvider
	repo domain.DocumentRepository
	cfg  *config.Config
	bm25 *bm25.Index
}

func NewIngestionUseCase(llm domain.LLMProvider, repo domain.DocumentRepository, cfg *config.Config, bm25Idx *bm25.Index) *IngestionUseCase {
	return &IngestionUseCase{llm: llm, repo: repo, cfg: cfg, bm25: bm25Idx}
}

// IngestText splits text into chunks, computes dense + BM25 sparse vectors, and saves to Qdrant.
func (uc *IngestionUseCase) IngestText(ctx context.Context, filename, text string) (int, error) {
	if err := uc.ensureCollection(ctx); err != nil {
		return 0, err
	}

	rawChunks := deduplicateChunks(splitIntoChunks(text, uc.cfg.ChunkSize, uc.cfg.ChunkOverlap))
	docID := uuid.New().String()

	// Update BM25 corpus statistics from this batch of chunks before computing vectors,
	// so IDF values include the current document's term frequencies.
	uc.bm25.AddChunks(rawChunks)
	if err := uc.bm25.Save(); err != nil {
		// Non-fatal: BM25 stats will be recomputed next time if save fails.
		fmt.Printf("warn: failed to save BM25 stats: %v\n", err)
	}

	chunks := make([]domain.Chunk, len(rawChunks))
	errCh := make(chan error, 1)

	// Worker pool — limit concurrency to avoid OOM on 16GB RAM.
	const workers = 4
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, raw := range rawChunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, text string) {
			defer wg.Done()
			defer func() { <-sem }()

			vec, err := uc.llm.GenerateEmbedding(ctx, text)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("embed chunk %d: %w", i, err):
				default:
				}
				return
			}

			sparseIdx, sparseVal := uc.bm25.VectorizeDoc(text)
			chunks[i] = domain.Chunk{
				ID:            uuid.New().String(),
				DocumentID:    docID,
				Filename:      filename,
				Text:          text,
				Vector:        vec,
				SparseIndices: sparseIdx,
				SparseValues:  sparseVal,
			}
		}(i, raw)
	}

	wg.Wait()
	close(errCh)

	if err := <-errCh; err != nil {
		return 0, err
	}

	if err := uc.repo.SaveChunks(chunks); err != nil {
		return 0, fmt.Errorf("save chunks: %w", err)
	}
	return len(chunks), nil
}

func (uc *IngestionUseCase) ensureCollection(ctx context.Context) error {
	exists, err := uc.repo.CollectionExists()
	if err != nil {
		return err
	}
	if !exists {
		return uc.repo.CreateCollection(uc.llm.EmbeddingDimension())
	}
	return nil
}

// deduplicateChunks removes chunks that are too similar to previously seen ones
// using Jaccard similarity on word sets (threshold 0.75).
func deduplicateChunks(chunks []string) []string {
	seen := make([]map[string]struct{}, 0, len(chunks))
	result := make([]string, 0, len(chunks))

	for _, chunk := range chunks {
		words := strings.Fields(strings.ToLower(chunk))
		wordSet := make(map[string]struct{}, len(words))
		for _, w := range words {
			wordSet[w] = struct{}{}
		}

		duplicate := false
		for _, prev := range seen {
			if jaccardSimilarity(wordSet, prev) > 0.75 {
				duplicate = true
				break
			}
		}

		if !duplicate {
			seen = append(seen, wordSet)
			result = append(result, chunk)
		}
	}
	return result
}

func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	intersection := 0
	for w := range a {
		if _, ok := b[w]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	return float64(intersection) / float64(union)
}

// splitIntoChunks splits text by words into chunks of roughly `size` chars
// with `overlap` chars carried over to the next chunk.
func splitIntoChunks(text string, size, overlap int) []string {
	words := strings.Fields(text)
	var chunks []string
	var buf strings.Builder
	pos := 0

	for pos < len(words) {
		buf.Reset()
		for buf.Len() < size && pos < len(words) {
			if buf.Len() > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(words[pos])
			pos++
		}
		chunk := buf.String()
		chunks = append(chunks, chunk)

		if overlap > 0 && pos < len(words) {
			overlapWords := 0
			for i := len(chunk) - 1; i >= 0 && overlapWords*5 < overlap; i-- {
				if chunk[i] == ' ' {
					overlapWords++
				}
			}
			pos -= overlapWords
			if pos < 0 {
				pos = 0
			}
		}
	}
	return chunks
}
