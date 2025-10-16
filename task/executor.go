package task

import (
	"context"
	"dwight/api"
	"dwight/config"
	"dwight/prompts"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Executor struct {
	client *api.OpenAIClient
	cfg    *config.Config
}

func NewExecutor(client *api.OpenAIClient, cfg *config.Config) *Executor {
	return &Executor{
		client: client,
		cfg:    cfg,
	}
}

func (e *Executor) Execute(task string) error {
	structure, err := e.getProjectStructure()
	if err != nil {
		return err
	}

	tools := e.getTools()
	messages := e.createInitialMessages(structure, task)

	for {
		startTime := time.Now()
		fullResponse, err := e.createChatCompletion(messages, tools, startTime)
		if err != nil {
			return err
		}

		if len(fullResponse.Choices) == 0 {
			return fmt.Errorf("no choices returned by the model")
		}

		choice := fullResponse.Choices[0]
		messages = append(messages, choice.Message)

		if len(choice.Message.ToolCalls) == 0 {
			if e.isTaskComplete(choice.Message.Content) {
				fmt.Println("Task completed!")
				break
			}

			fmt.Println("Empty response from AI, exiting...")
			break
		}

		if e.handleToolCalls(choice.Message.ToolCalls, &messages) {
			fmt.Println("Task completed!")
			break
		}
	}

	return nil
}

func (e *Executor) createChatCompletion(messages []openai.ChatCompletionMessage, tools []openai.Tool, startTime time.Time) (openai.ChatCompletionResponse, error) {
	done := make(chan bool)
	var response openai.ChatCompletionResponse
	var err error

	go func() {
		response, err = e.client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model:    e.cfg.Model,
			Messages: messages,
			Tools:    tools,
		})
		done <- true
	}()

	e.showSpinnerUntilDone(done, startTime)

	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	fmt.Printf("\r\x1b[32mExecuting AI request... ✓ (%.1f s)\x1b[0m\n", time.Since(startTime).Seconds())

	if len(response.Choices) > 0 {
		message := response.Choices[0].Message
		if message.Content != "" {
			fmt.Printf("\x1b[34m\nModel message:\n\x1b[0m%s\n\n", message.Content)
		}
	}

	return response, nil
}

func (e *Executor) showSpinnerUntilDone(done chan bool, startTime time.Time) {
	spinnerChars := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		select {
		case <-done:
			return
		default:
			elapsed := time.Since(startTime).Seconds()
			fmt.Printf("\r\x1b[36mExecuting AI request... %s (%.1f s)\x1b[0m", spinnerChars[i%len(spinnerChars)], elapsed)
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}

func (e *Executor) getTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_file_contents",
				Description: "Get contents of multiple files",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"files": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"type": "string"},
						},
					},
					"required": []string{"files"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "modify_files",
				Description: "Modify or create multiple files with new content",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"files": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"file_path": map[string]interface{}{
										"type":        "string",
										"description": "Path to the file to modify or create",
									},
									"content": map[string]interface{}{
										"type":        "string",
										"description": "Complete new content for the file",
									},
								},
								"required": []string{"file_path", "content"},
							},
						},
					},
					"required": []string{"files"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_command",
				Description: "Execute a shell command (sh -c <your_command>)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Shell command to execute",
						},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "ask_question",
				Description: "Ask the user a clarifying question",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"question": map[string]interface{}{
							"type":        "string",
							"description": "Question to ask the user",
						},
					},
					"required": []string{"question"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "task_complete",
				Description: "Mark the task as completed",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
}

func (e *Executor) createInitialMessages(structure, task string) []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.TaskExecutionSP,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Project structure:\n%s\n\nTask: %s", structure, task),
		},
	}
}

func (e *Executor) isTaskComplete(content string) bool {
	lowerContent := strings.ToLower(content)
	return strings.Contains(lowerContent, "done") ||
		strings.Contains(lowerContent, "complete") ||
		strings.Contains(lowerContent, "finished")
}

func (e *Executor) handleToolCalls(toolCalls []openai.ToolCall, messages *[]openai.ChatCompletionMessage) bool {
	taskComplete := false
	for _, toolCall := range toolCalls {
		if toolCall.Function.Name == "task_complete" {
			taskComplete = true
			continue
		}

		if err := e.handleToolCall(toolCall, messages); err != nil {
			fmt.Fprintf(os.Stderr, "Error handling tool call: %v\n", err)
		}
	}
	return taskComplete
}
