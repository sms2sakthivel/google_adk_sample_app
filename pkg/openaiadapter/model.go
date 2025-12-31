package openaiadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ModelStruct implements the model.LLM interface via sashabaranov/go-openai.
type ModelStruct struct {
	client *openai.Client
	model  string
}

// NewModel creates a new OpenAI-compatible model instance.
func NewModel(baseURL, modelName, apiKey string) *ModelStruct {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	client := openai.NewClientWithConfig(config)
	return &ModelStruct{
		client: client,
		model:  modelName,
	}
}

func (m *ModelStruct) Name() string {
	return "ollama-" + m.model
}

func (m *ModelStruct) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		messages, err := toOpenAIMessages(req.Contents)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert messages: %w", err))
			return
		}

		if req.Config != nil && req.Config.SystemInstruction != nil {
			sysMsg, err := toOpenAISystemMessage(req.Config.SystemInstruction)
			if err != nil {
				yield(nil, fmt.Errorf("failed to convert system instruction: %w", err))
				return
			}
			// Prepend system message
			messages = append([]openai.ChatCompletionMessage{sysMsg}, messages...)
		}

		var tools []openai.Tool
		if len(req.Tools) > 0 {
			tools = toOpenAITools(req.Tools)
		}

		log.Printf("[Ollama] Sending request to model: %s with %d messages and %d tools", m.model, len(messages), len(tools))

		resp, err := m.client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:    m.model,
				Messages: messages,
				Tools:    tools,
			},
		)
		if err != nil {
			yield(nil, fmt.Errorf("ollama call failed: %w", err))
			return
		}

		adkResp, err := toADKResponse(resp)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert response: %w", err))
			return
		}

		yield(adkResp, nil)
	}
}

func toOpenAIMessages(contents []*genai.Content) ([]openai.ChatCompletionMessage, error) {
	var messages []openai.ChatCompletionMessage

	for _, c := range contents {
		role := c.Role
		if role == "model" {
			role = openai.ChatMessageRoleAssistant
		} else if role == "" {
			role = openai.ChatMessageRoleUser
		}

		var textContent string
		var toolCalls []openai.ToolCall

		// We need to handle mixed parts: text, calls, responses.
		// OpenAI expects:
		// - Assistant message with ToolCalls (and optional content)
		// - Tool messages (one per call)
		// ADK groups them in Content. We must split if necessary.
		// Strategy: Aggregate Text/ToolCalls, flush when hitting FunctionResponse, or at end.

		for _, p := range c.Parts {
			if p.FunctionResponse != nil {
				// If we have pending text/calls, flush them first as a message
				if textContent != "" || len(toolCalls) > 0 {
					msg := openai.ChatCompletionMessage{
						Role:    role,
						Content: textContent,
					}
					if len(toolCalls) > 0 {
						msg.ToolCalls = toolCalls
					}
					messages = append(messages, msg)
					// Reset
					textContent = ""
					toolCalls = nil
				}

				// Now append the Tool Response message
				toolResponseID := fmt.Sprintf("call_%s", p.FunctionResponse.Name)
				respJSON, _ := json.Marshal(p.FunctionResponse.Response)

				log.Printf("[Ollama] Converting FunctionResponse to ToolMessage. ID: %s, Payload Size: %d bytes", toolResponseID, len(respJSON))

				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    string(respJSON),
					ToolCallID: toolResponseID,
				})
			} else {
				// Accumulate Text / ToolCalls
				if p.Text != "" {
					textContent += p.Text
				}
				if p.InlineData != nil {
					textContent += string(p.InlineData.Data)
				}
				if p.FunctionCall != nil {
					argsJSON, _ := json.Marshal(p.FunctionCall.Args)
					toolCalls = append(toolCalls, openai.ToolCall{
						ID:   fmt.Sprintf("call_%s", p.FunctionCall.Name),
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      p.FunctionCall.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
		}

		// Final flush for any remaining text/calls
		if textContent != "" || len(toolCalls) > 0 {
			msg := openai.ChatCompletionMessage{
				Role:    role,
				Content: textContent,
			}
			if len(toolCalls) > 0 {
				msg.ToolCalls = toolCalls
			}
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

func toOpenAISystemMessage(c *genai.Content) (openai.ChatCompletionMessage, error) {
	var textContent string
	for _, p := range c.Parts {
		if p.Text != "" {
			textContent += p.Text
		}
	}
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: textContent,
	}, nil
}

// Declarer is an interface for tools that provide a GenAI FunctionDeclaration.
type Declarer interface {
	Declaration() *genai.FunctionDeclaration
}

func toOpenAITools(adkTools map[string]any) []openai.Tool {
	var tools []openai.Tool
	for _, v := range adkTools {
		var name, description string
		var parameters any

		// Try to get declaration if available
		if declarer, ok := v.(Declarer); ok {
			decl := declarer.Declaration()
			name = decl.Name
			description = decl.Description
			parameters = decl.Parameters
		} else {
			// Fallback: Try marshalling the object itself
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				continue
			}
			type ToolDef struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Parameters  any    `json:"parameters"`
			}
			var td ToolDef
			if err := json.Unmarshal(jsonBytes, &td); err == nil && td.Name != "" {
				name = td.Name
				description = td.Description
				parameters = td.Parameters
			}
		}

		if name != "" {
			tools = append(tools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        name,
					Description: description,
					Parameters:  parameters,
				},
			})
		}
	}
	return tools
}

func toADKResponse(resp openai.ChatCompletionResponse) (*model.LLMResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, errors.New("no choices returned from ollama")
	}

	choice := resp.Choices[0]
	var parts []*genai.Part

	if choice.Message.Content != "" {
		parts = append(parts, &genai.Part{Text: choice.Message.Content})
	}

	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			log.Printf("Failed to unmarshal tool arguments: %v", err)
			continue
		}

		// Workaround: Qwen/Ollama sometimes sends "artifact_names": "file.txt"
		// instead of ["file.txt"]. The ADK tool expects an array.
		if val, ok := args["artifact_names"]; ok {
			if strVal, ok := val.(string); ok {
				log.Printf("[Ollama] Fixing malformed artifact_names: %s -> [%s]", strVal, strVal)
				args["artifact_names"] = []string{strVal}
			}
		}

		parts = append(parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				Name: tc.Function.Name,
				Args: args,
			},
		})
	}

	return &model.LLMResponse{
		Content: &genai.Content{Role: "model", Parts: parts},
	}, nil
}
