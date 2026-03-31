package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dafahan/unila-ai/internal/domain"
)

// Named vector keys used in the Qdrant collection.
const (
	vecDense  = "dense"
	vecSparse = "bm25"
)

type QdrantRepository struct {
	client      pb.PointsClient
	collections pb.CollectionsClient
	collection  string
}

func NewQdrantRepository(host string, port int, collection string) (*QdrantRepository, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("qdrant connect: %w", err)
	}
	return &QdrantRepository{
		client:      pb.NewPointsClient(conn),
		collections: pb.NewCollectionsClient(conn),
		collection:  collection,
	}, nil
}

func (r *QdrantRepository) CollectionExists() (bool, error) {
	resp, err := r.collections.List(context.Background(), &pb.ListCollectionsRequest{})
	if err != nil {
		return false, err
	}
	for _, c := range resp.Collections {
		if c.Name == r.collection {
			return true, nil
		}
	}
	return false, nil
}

// CreateCollection creates the collection with named dense + sparse vectors.
// Dense: "dense" (Cosine similarity). Sparse: "bm25" (Dot product via RRF fusion).
//
// NOTE: If upgrading from a single-vector collection, delete the old collection
// from the Qdrant dashboard (/dashboard) and re-upload all documents.
func (r *QdrantRepository) CreateCollection(dimension int) error {
	_, err := r.collections.Create(context.Background(), &pb.CreateCollection{
		CollectionName: r.collection,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_ParamsMap{
				ParamsMap: &pb.VectorParamsMap{
					Map: map[string]*pb.VectorParams{
						vecDense: {
							Size:     uint64(dimension),
							Distance: pb.Distance_Cosine,
						},
					},
				},
			},
		},
		SparseVectorsConfig: &pb.SparseVectorConfig{
			Map: map[string]*pb.SparseVectorParams{
				vecSparse: {},
			},
		},
	})
	return err
}

// SaveChunks upserts chunks with both dense and BM25 sparse named vectors.
func (r *QdrantRepository) SaveChunks(chunks []domain.Chunk) error {
	points := make([]*pb.PointStruct, 0, len(chunks))
	for _, c := range chunks {
		id := uuid.New().String()

		namedVecs := map[string]*pb.Vector{
			vecDense: {
				Vector: &pb.Vector_Dense{
					Dense: &pb.DenseVector{Data: c.Vector},
				},
			},
		}
		if len(c.SparseIndices) > 0 {
			namedVecs[vecSparse] = &pb.Vector{
				Vector: &pb.Vector_Sparse{
					Sparse: &pb.SparseVector{
						Values:  c.SparseValues,
						Indices: c.SparseIndices,
					},
				},
			}
		}

		points = append(points, &pb.PointStruct{
			Id: &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: id}},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vectors{
					Vectors: &pb.NamedVectors{Vectors: namedVecs},
				},
			},
			Payload: map[string]*pb.Value{
				"document_id": {Kind: &pb.Value_StringValue{StringValue: c.DocumentID}},
				"filename":    {Kind: &pb.Value_StringValue{StringValue: c.Filename}},
				"page":        {Kind: &pb.Value_IntegerValue{IntegerValue: int64(c.PageNumber)}},
				"text":        {Kind: &pb.Value_StringValue{StringValue: c.Text}},
			},
		})
	}

	_, err := r.client.Upsert(context.Background(), &pb.UpsertPoints{
		CollectionName: r.collection,
		Points:         points,
	})
	return err
}

// SearchSimilar performs sparse-dense hybrid search using Qdrant's native
// prefetch + Reciprocal Rank Fusion (RRF). Both the dense vector (semantic
// similarity via cosine) and the BM25 sparse vector (lexical relevance via
// dot product) are used as independent retrieval signals whose ranked lists
// are merged by RRF. Falls back to dense-only if no sparse vector is provided.
func (r *QdrantRepository) SearchSimilar(denseVec []float32, sparseIndices []uint32, sparseValues []float32, topK int, scoreThreshold float32) ([]domain.Chunk, error) {
	ctx := context.Background()
	candidateLimit := uint64(topK * 4)
	if candidateLimit < 20 {
		candidateLimit = 20
	}

	prefetches := []*pb.PrefetchQuery{
		{
			Query: &pb.Query{
				Variant: &pb.Query_Nearest{
					Nearest: &pb.VectorInput{
						Variant: &pb.VectorInput_Dense{
							Dense: &pb.DenseVector{Data: denseVec},
						},
					},
				},
			},
			Using: strPtr(vecDense),
			Limit: uint64Ptr(candidateLimit),
		},
	}

	if len(sparseIndices) > 0 {
		prefetches = append(prefetches, &pb.PrefetchQuery{
			Query: &pb.Query{
				Variant: &pb.Query_Nearest{
					Nearest: &pb.VectorInput{
						Variant: &pb.VectorInput_Sparse{
							Sparse: &pb.SparseVector{
								Indices: sparseIndices,
								Values:  sparseValues,
							},
						},
					},
				},
			},
			Using: strPtr(vecSparse),
			Limit: uint64Ptr(candidateLimit),
		})
	}

	resp, err := r.client.Query(ctx, &pb.QueryPoints{
		CollectionName: r.collection,
		Prefetch:       prefetches,
		Query: &pb.Query{
			Variant: &pb.Query_Fusion{
				Fusion: pb.Fusion_RRF,
			},
		},
		Limit:          uint64Ptr(uint64(topK)),
		ScoreThreshold: &scoreThreshold,
		WithPayload: &pb.WithPayloadSelector{
			SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant query: %w", err)
	}

	chunks := make([]domain.Chunk, 0, len(resp.Result))
	for _, pt := range resp.Result {
		p := pt.Payload
		chunks = append(chunks, domain.Chunk{
			DocumentID: p["document_id"].GetStringValue(),
			Filename:   p["filename"].GetStringValue(),
			PageNumber: int(p["page"].GetIntegerValue()),
			Text:       p["text"].GetStringValue(),
		})
	}
	return chunks, nil
}

func (r *QdrantRepository) ListDocuments() ([]domain.DocumentInfo, error) {
	var offset *pb.PointId
	counts := make(map[string]int)

	for {
		resp, err := r.client.Scroll(context.Background(), &pb.ScrollPoints{
			CollectionName: r.collection,
			Offset:         offset,
			Limit:          ptr(uint32(100)),
			WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
			WithVectors:    &pb.WithVectorsSelector{SelectorOptions: &pb.WithVectorsSelector_Enable{Enable: false}},
		})
		if err != nil {
			return nil, fmt.Errorf("qdrant scroll: %w", err)
		}
		for _, pt := range resp.Result {
			name := pt.Payload["filename"].GetStringValue()
			counts[name]++
		}
		if resp.NextPageOffset == nil {
			break
		}
		offset = resp.NextPageOffset
	}

	docs := make([]domain.DocumentInfo, 0, len(counts))
	for name, count := range counts {
		docs = append(docs, domain.DocumentInfo{Filename: name, ChunkCount: count})
	}
	return docs, nil
}

func (r *QdrantRepository) DeleteByFilename(filename string) (int, error) {
	_, err := r.client.Delete(context.Background(), &pb.DeletePoints{
		CollectionName: r.collection,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Filter{
				Filter: &pb.Filter{
					Must: []*pb.Condition{
						{
							ConditionOneOf: &pb.Condition_Field{
								Field: &pb.FieldCondition{
									Key: "filename",
									Match: &pb.Match{
										MatchValue: &pb.Match_Keyword{Keyword: filename},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("qdrant delete: %w", err)
	}
	return 0, nil
}

func ptr[T any](v T) *T        { return &v }
func strPtr(s string) *string  { return &s }
func uint64Ptr(n uint64) *uint64 { return &n }
