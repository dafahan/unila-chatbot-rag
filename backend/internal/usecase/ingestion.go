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
	pdfextract "github.com/dafahan/unila-ai/pkg/pdf"
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

// IngestPages ingests a PDF that has already been split into pages (with page numbers).
// Each page is chunked independently so that chunk.PageNumber is accurate.
func (uc *IngestionUseCase) IngestPages(ctx context.Context, filename string, pages []pdfextract.PageText) (int, error) {
	if err := uc.ensureCollection(ctx); err != nil {
		return 0, err
	}

	// Build flat list of (text, pageNum) pairs after chunking per page
	type rawChunk struct {
		text string
		page int
	}
	var raws []rawChunk
	for _, p := range pages {
		for _, chunk := range splitIntoChunks(cleanText(p.Text), uc.cfg.ChunkSize, uc.cfg.ChunkOverlap) {
			raws = append(raws, rawChunk{text: chunk, page: p.Page})
		}
	}

	// Deduplicate on text only
	texts := make([]string, len(raws))
	for i, r := range raws {
		texts[i] = r.text
	}
	kept := deduplicateChunksIndexed(texts)
	dedupedRaws := make([]rawChunk, 0, len(kept))
	for _, i := range kept {
		dedupedRaws = append(dedupedRaws, raws[i])
	}

	// Update BM25 corpus stats
	dedupedTexts := make([]string, len(dedupedRaws))
	for i, r := range dedupedRaws {
		dedupedTexts[i] = r.text
	}
	uc.bm25.AddChunks(dedupedTexts)
	if err := uc.bm25.Save(); err != nil {
		fmt.Printf("warn: failed to save BM25 stats: %v\n", err)
	}

	docID := uuid.New().String()
	chunks := make([]domain.Chunk, len(dedupedRaws))
	errCh := make(chan error, 1)

	const workers = 4
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, raw := range dedupedRaws {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, text string, page int) {
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
				PageNumber:    page,
				Vector:        vec,
				SparseIndices: sparseIdx,
				SparseValues:  sparseVal,
			}
		}(i, raw.text, raw.page)
	}

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return 0, err
	}

	if len(chunks) == 0 {
		return 0, fmt.Errorf("no text extracted from PDF — file may be scanned/image-only; upload as .md instead")
	}
	if err := uc.repo.SaveChunks(chunks); err != nil {
		return 0, fmt.Errorf("save chunks: %w", err)
	}
	return len(chunks), nil
}

// IngestText splits text into chunks, computes dense + BM25 sparse vectors, and saves to Qdrant.
// For markdown files it uses contextual chunking that prepends the nearest section heading to each
// chunk, which significantly improves retrieval accuracy for structured academic documents.
func (uc *IngestionUseCase) IngestText(ctx context.Context, filename, text string) (int, error) {
	if err := uc.ensureCollection(ctx); err != nil {
		return 0, err
	}

	var rawChunks []string
	if isMarkdown(text) {
		rawChunks = deduplicateChunks(splitMarkdownWithContext(cleanText(text), uc.cfg.ChunkSize, uc.cfg.ChunkOverlap))
	} else {
		rawChunks = deduplicateChunks(splitIntoChunks(cleanText(text), uc.cfg.ChunkSize, uc.cfg.ChunkOverlap))
	}
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
// deduplicateChunksIndexed returns the indices of non-duplicate chunks.
func deduplicateChunksIndexed(chunks []string) []int {
	seen := make([]map[string]struct{}, 0, len(chunks))
	var result []int
	for i, chunk := range chunks {
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
			result = append(result, i)
		}
	}
	return result
}

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

// cleanText removes known PDF extraction noise before chunking.
// "BUKU – " is a recurring header artifact in the SIAKAD guide PDFs.
func cleanText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "BUKU –"); idx != -1 {
			lines[i] = strings.TrimSpace(line[idx+len("BUKU –"):])
		}
	}
	return strings.Join(lines, "\n")
}

// isMarkdown returns true if the text looks like a markdown document
// (has at least two lines starting with a `#` heading marker).
func isMarkdown(text string) bool {
	count := 0
	for _, line := range strings.SplitN(text, "\n", 200) {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			count++
			if count >= 2 {
				return true
			}
		}
	}
	return false
}

// splitMarkdownWithContext splits a markdown document into chunks while
// prepending the most recent section heading hierarchy to each chunk.
// This ensures every chunk is self-contained for retrieval purposes.
func splitMarkdownWithContext(text string, size, overlap int) []string {
	lines := strings.Split(text, "\n")
	var result []string

	// headings[0]=h1, headings[1]=h2, headings[2]=h3
	headings := make([]string, 0, 3)
	var buf strings.Builder

	flush := func() {
		body := strings.TrimSpace(buf.String())
		if body == "" {
			return
		}
		prefix := strings.Join(headings, " > ")
		var full string
		if prefix != "" {
			full = prefix + "\n\n" + body
		} else {
			full = body
		}
		// If the full section fits within 2× chunk size, keep it as one chunk
		// so that key facts (e.g. a tariff table) are never split from their heading.
		if len(full) <= size*2 {
			result = append(result, full)
		} else {
			result = append(result, splitIntoChunks(full, size, overlap)...)
		}
		buf.Reset()
	}

	headingLevel := func(line string) (int, string) {
		trimmed := strings.TrimSpace(line)
		for lvl := 1; lvl <= 4; lvl++ {
			prefix := strings.Repeat("#", lvl) + " "
			if strings.HasPrefix(trimmed, prefix) {
				return lvl, strings.TrimPrefix(trimmed, prefix)
			}
		}
		return 0, ""
	}

	for _, line := range lines {
		lvl, title := headingLevel(line)
		if lvl > 0 && lvl <= 3 {
			flush()
			// Trim headings slice to parent level and append new heading
			if lvl-1 < len(headings) {
				headings = headings[:lvl-1]
			}
			headings = append(headings, title)
		} else {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	flush()
	return result
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
