package api

import (
	"context"
	"dwight/config"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient(config *config.Config) *OpenAIClient {
	clientConfig := openai.DefaultConfig(config.Token)
	clientConfig.BaseURL = config.BaseURL
	client := openai.NewClientWithConfig(clientConfig)
	return &OpenAIClient{client: client}
}

func (o *OpenAIClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	return o.client.CreateChatCompletion(ctx, req)
}
