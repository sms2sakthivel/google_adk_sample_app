package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// InteractiveWizard implements Loader using CLI prompts.
type InteractiveWizard struct{}

// NewInteractiveWizard creates a new wizard instance.
func NewInteractiveWizard() *InteractiveWizard {
	return &InteractiveWizard{}
}

// SelectInterfaceMode prompts the user to choose the interface mode (Console/Web).
// This is structurally separate from loading the Agent's configuration.
func (w *InteractiveWizard) SelectInterfaceMode() ([]string, error) {
	fmt.Println("=== Interface Selection ===")
	fmt.Println("1. Console (Interact via terminal)")
	fmt.Println("2. Web GUI (Interact via browser)")
	fmt.Print("Enter choice [1]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		if choice == "2" {
			return []string{"web", "api", "webui"}, nil
		}
	}
	return []string{"console"}, nil
}

// LoadLLMConfig prompts the user to configure the LLM Provider.
func (w *InteractiveWizard) LoadLLMConfig() (*LLMConfig, error) {
	reader := bufio.NewReader(os.Stdin)
	config := &LLMConfig{}

	fmt.Println("\n=== Agent Configuration ===")
	// Ask for Provider (Local/Corporate)
	w.askProviderDetails(reader, config)

	fmt.Println("\nConfiguration Complete.")
	fmt.Println("------------------------")
	return config, nil
}

func (w *InteractiveWizard) askProviderDetails(reader *bufio.Reader, config *LLMConfig) {
	fmt.Println("Select LLM Provider:")
	fmt.Println("1. Local Ollama (Default)")
	fmt.Println("2. Corporate / Private LLM (OpenAI Compatible)")
	fmt.Print("Enter choice [1]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "2" {
		// Corporate LLM
		fmt.Print("Enter Base URL (e.g. http://llm.corp.net/v1): ")
		url, _ := reader.ReadString('\n')
		config.BaseURL = strings.TrimSpace(url)

		fmt.Print("Enter API Key: ")
		key, _ := reader.ReadString('\n')
		config.APIKey = strings.TrimSpace(key)

		fmt.Print("Enter Model Name (e.g. gpt-4, llama3): ")
		model, _ := reader.ReadString('\n')
		config.ModelName = strings.TrimSpace(model)
	} else {
		// Local Ollama (Default)
		config.BaseURL = "http://localhost:11434/v1"
		config.APIKey = "ollama"
		config.ModelName = "qwen2.5:latest"

		fmt.Print("\nUse default Ollama settings (http://localhost:11434/v1, qwen2.5:latest)? [Y/n]: ")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))

		if confirm == "n" || confirm == "no" {
			fmt.Printf("Enter Base URL [%s]: ", config.BaseURL)
			url, _ := reader.ReadString('\n')
			if val := strings.TrimSpace(url); val != "" {
				config.BaseURL = val
			}

			fmt.Printf("Enter Model Name [%s]: ", config.ModelName)
			model, _ := reader.ReadString('\n')
			if val := strings.TrimSpace(model); val != "" {
				config.ModelName = val
			}
		}
	}
}
