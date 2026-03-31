package domain

// Chunk is a piece of text extracted from a document, stored in Qdrant.
type Chunk struct {
	ID            string    `json:"-"`
	DocumentID    string    `json:"-"`
	Filename      string    `json:"filename"`
	PageNumber    int       `json:"page"`
	Text          string    `json:"text"`
	Vector        []float32 `json:"-"` // dense embedding (768-dim)
	SparseIndices []uint32  `json:"-"` // BM25 sparse vector indices
	SparseValues  []float32 `json:"-"` // BM25 sparse vector weights
}

// DocumentInfo is a summary of an ingested document.
type DocumentInfo struct {
	Filename   string `json:"filename"`
	ChunkCount int    `json:"chunk_count"`
}

// DocumentRepository handles persistence of document chunks and vectors.
type DocumentRepository interface {
	SaveChunks(chunks []Chunk) error
	SearchSimilar(denseVec []float32, sparseIndices []uint32, sparseValues []float32, topK int) ([]Chunk, error)
	CollectionExists() (bool, error)
	CreateCollection(dimension int) error
	ListDocuments() ([]DocumentInfo, error)
	DeleteByFilename(filename string) (int, error)
}
