package provider

import (
	"context"
	"time"

	"github.com/sashabaranov/go-openai"
)

type CloudProvider struct {
	client     *openai.Client
	model      string
	maxRetries int
}

func NewCloudProvider(apiKey, baseURL, model string) *CloudProvider {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	return &CloudProvider{
		client:     openai.NewClientWithConfig(config),
		model:      model,
		maxRetries: 3,
	}
}

func (p *CloudProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	var resp openai.EmbeddingResponse
	var err error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		resp, err = p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
			Input: texts,
			Model: openai.EmbeddingModel(p.model),
		})

		if err == nil {
			break
		}

		if attempt < p.maxRetries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	if err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}
	return embeddings, nil
}
