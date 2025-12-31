package config

// LLMConfig holds the configuration for the LLM provider.
// It is independent of the user interface mode.
type LLMConfig struct {
	BaseURL   string
	APIKey    string
	ModelName string
}

// Loader defines how configuration is loaded.
type Loader interface {
	LoadLLMConfig() (*LLMConfig, error)
}
