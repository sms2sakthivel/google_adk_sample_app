package tools

import (
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// SearchArgs defines the arguments for the search tool.
type SearchArgs struct {
	Query string `json:"query" description:"The search query to find information about."`
}

// SearchResult defines the output of the search tool.
type SearchResult struct {
	Results []string `json:"results"`
}

// NewSearchTool creates a new simple search tool.
func NewSearchTool() (tool.Tool, error) {
	return functiontool.New(
		functiontool.Config{
			Name:        "search",
			Description: "Search the web for information. Use this tool finding real-time data or facts.",
		},
		func(ctx tool.Context, args SearchArgs) (SearchResult, error) {
			// In a real implementation, this would call Google Search API or similar.
			// For this local Ollama demo, we simulate a response.

			query := args.Query
			results := []string{
				fmt.Sprintf("Result 1 for '%s': This is a simulated search result.", query),
				fmt.Sprintf("Result 2: details about %s found on the web.", query),
			}

			// Mock specific data if needed for the "Capital of France" query
			if query == "Capital of France" || query == "capital of France" {
				results = []string{"The capital of France is Paris."}
			}

			return SearchResult{Results: results}, nil
		},
	)
}
