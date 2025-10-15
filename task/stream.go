package task

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

func (e *Executor) consumeStream(messages []openai.ChatCompletionMessage, tools []openai.Tool, startTime time.Time, fullResponse *openai.ChatCompletionResponse) error {
	stream, err := e.client.CreateChatCompletionStream(context.Background(), openai.ChatCompletionRequest{
		Model:    e.config.Model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	var response openai.ChatCompletionResponse
	var accumulatedContent string
	var toolCalls []openai.ToolCall

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		e.updateProgress(startTime)
		e.accumulateChunk(chunk, &accumulatedContent, &toolCalls)

		if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != "" {
			response = e.buildFinalResponse(chunk, accumulatedContent, toolCalls)
		}
	}

	*fullResponse = response
	fmt.Printf("\r\x1b[32mExecuting AI request... ✓ (%.1f s)\x1b[0m\n", time.Since(startTime).Seconds())

	if len(response.Choices) > 0 {
		message := response.Choices[0].Message
		if message.Content != "" {
			fmt.Printf("\x1b[34m\nModel message:\n\x1b[0m%s\n\n", message.Content)
		}
	}

	return nil
}

func (e *Executor) updateProgress(startTime time.Time) {
	elapsed := time.Since(startTime).Seconds()
	spinnerChars := []string{"|", "/", "-", "\\"}
	spinnerIndex := int(elapsed*2) % len(spinnerChars)
	fmt.Printf("\r\x1b[36mExecuting AI request... %s (%.1f s)\x1b[0m", spinnerChars[spinnerIndex], elapsed)
}

func (e *Executor) accumulateChunk(chunk openai.ChatCompletionStreamResponse, accumulatedContent *string, toolCalls *[]openai.ToolCall) {
	if len(chunk.Choices) == 0 {
		return
	}

	choice := chunk.Choices[0]
	if choice.Delta.Content != "" {
		*accumulatedContent += choice.Delta.Content
	}

	if len(choice.Delta.ToolCalls) > 0 {
		for _, toolCallDelta := range choice.Delta.ToolCalls {
			if *toolCallDelta.Index >= len(*toolCalls) {
				*toolCalls = append(*toolCalls, openai.ToolCall{
					ID:   toolCallDelta.ID,
					Type: toolCallDelta.Type,
					Function: openai.FunctionCall{
						Name:      toolCallDelta.Function.Name,
						Arguments: toolCallDelta.Function.Arguments,
					},
				})
			} else {
				(*toolCalls)[*toolCallDelta.Index].Function.Arguments += toolCallDelta.Function.Arguments
			}
		}
	}
}

func (e *Executor) buildFinalResponse(chunk openai.ChatCompletionStreamResponse, content string, toolCalls []openai.ToolCall) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		ID:      chunk.ID,
		Object:  chunk.Object,
		Created: chunk.Created,
		Model:   chunk.Model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: chunk.Choices[0].Index,
				Message: openai.ChatCompletionMessage{
					Role:      openai.ChatMessageRoleAssistant,
					Content:   content,
					ToolCalls: toolCalls,
				},
				FinishReason: chunk.Choices[0].FinishReason,
			},
		},
	}
}
