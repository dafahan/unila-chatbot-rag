package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dafahan/unila-ai/internal/domain"
)

type QdrantRepository struct {
	client     pb.PointsClient
	collections pb.CollectionsClient
	collection string
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

func (r *QdrantRepository) CreateCollection(dimension int) error {
	_, err := r.collections.Create(context.Background(), &pb.CreateCollection{
		CollectionName: r.collection,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     uint64(dimension),
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})
	return err
}

func (r *QdrantRepository) SaveChunks(chunks []domain.Chunk) error {
	points := make([]*pb.PointStruct, 0, len(chunks))
	for _, c := range chunks {
		id := uuid.New().String()
		points = append(points, &pb.PointStruct{
			Id: &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: id}},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vector{
					Vector: &pb.Vector{Data: c.Vector},
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

func (r *QdrantRepository) ListDocuments() ([]domain.DocumentInfo, error) {
	// Scroll all points, collect unique filenames and count chunks
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
	resp, err := r.client.Delete(context.Background(), &pb.DeletePoints{
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
	_ = resp
	return 0, nil
}

func ptr[T any](v T) *T { return &v }

// SearchSimilar does hybrid search: vector search + keyword boost reranking.
func (r *QdrantRepository) SearchSimilar(vector []float32, keywords []string, topK int) ([]domain.Chunk, error) {
	// Fetch more candidates than needed so reranker has room to work
	candidates := topK * 4
	if candidates < 20 {
		candidates = 20
	}

	resp, err := r.client.Search(context.Background(), &pb.SearchPoints{
		CollectionName: r.collection,
		Vector:         vector,
		Limit:          uint64(candidates),
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search: %w", err)
	}

	type scored struct {
		chunk domain.Chunk
		score float64
	}

	results := make([]scored, 0, len(resp.Result))
	for _, pt := range resp.Result {
		p := pt.Payload
		text := p["text"].GetStringValue()
		textLower := strings.ToLower(text)

		// Keyword boost: +0.1 per keyword found in the chunk
		boost := 0.0
		for _, kw := range keywords {
			if strings.Contains(textLower, strings.ToLower(kw)) {
				boost += 0.1
			}
		}

		results = append(results, scored{
			chunk: domain.Chunk{
				DocumentID: p["document_id"].GetStringValue(),
				Filename:   p["filename"].GetStringValue(),
				PageNumber: int(p["page"].GetIntegerValue()),
				Text:       text,
			},
			score: float64(pt.Score) + boost,
		})
	}

	// Sort by combined score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	chunks := make([]domain.Chunk, 0, topK)
	for i := 0; i < topK && i < len(results); i++ {
		chunks = append(chunks, results[i].chunk)
	}
	return chunks, nil
}
