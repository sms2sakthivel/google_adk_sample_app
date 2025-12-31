package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/loadartifactstool"

	"example.com/adk-agent/pkg/config"
	"example.com/adk-agent/pkg/openaiadapter"
	"example.com/adk-agent/pkg/services"
	"example.com/adk-agent/pkg/tools"
)

func main() {
	// Load .env file (optional)
	_ = godotenv.Load()
	ctx := context.Background()

	wizard := config.NewInteractiveWizard()
	var runArgs []string
	var llmConfig *config.LLMConfig
	var err error

	// Step 1: Determine Interface Mode (Run Mode)
	// Independent of Agent Configuration
	args := os.Args[1:]
	if len(args) == 0 {
		// No args -> Ask User
		runArgs, err = wizard.SelectInterfaceMode()
		if err != nil {
			log.Fatalf("Failed to select mode: %v", err)
		}
	} else {
		// Args present -> Map shortcuts or use directly
		runArgs = args
		if len(args) == 1 {
			if args[0] == "1" || args[0] == "console" {
				runArgs = []string{"console"}
			} else if args[0] == "2" || args[0] == "webui" {
				runArgs = []string{"web", "api", "webui"}
			}
		}
	}

	// Step 2: Determine Agent Configuration (LLM Provider)
	// This happens regardless of the interface mode, though we might skip wizard if env vars are strict.
	// For this design, we'll invoke the wizard if no args were passed (Interactive Mode),
	// but if args WERE passed (Shortcut Mode), we'll try to use Environment Variables/Defaults to stay non-blocking.

	if len(args) == 0 {
		// Interactive Mode: Ask for LLM Config explicitly
		llmConfig, err = wizard.LoadLLMConfig()
		if err != nil {
			log.Fatalf("Configuration failed: %v", err)
		}
	} else {
		// Non-Interactive (Shortcut): Use defaults or Env Vars
		llmConfig = &config.LLMConfig{
			BaseURL:   os.Getenv("OLLAMA_HOST"),
			ModelName: os.Getenv("OLLAMA_MODEL"),
			APIKey:    "ollama",
		}
		if llmConfig.BaseURL == "" {
			llmConfig.BaseURL = "http://localhost:11434/v1"
		}
		if llmConfig.ModelName == "" {
			llmConfig.ModelName = "qwen2.5:latest"
		}
	}

	// Step 3: Initialize Components
	searchTool, err := tools.NewSearchTool()
	if err != nil {
		log.Fatalf("Failed to create search tool: %v", err)
	}

	// Create FileSystem Artifact Service
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	artifactService := services.NewFileSystemArtifactService(cwd)

	// Create Load Artifacts Tool
	loadArtifactsTool := loadartifactstool.New()

	model := openaiadapter.NewModel(llmConfig.BaseURL, llmConfig.ModelName, llmConfig.APIKey)

	searchAgent, err := llmagent.New(llmagent.Config{
		Name:        "search_agent",
		Model:       model,
		Description: "A helpful assistant.",
		Instruction: `You are a helpful AI assistant.
You have access to a search tool and a local file artifact tool.
If the user asks for information you don't know or real-time facts, YOU MUST use the 'search' tool.
If the user asks about files or artifacts, use the 'load_artifacts' tool to read them.`,
		Tools: []tool.Tool{
			searchTool,
			loadArtifactsTool,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	launcherConfig := &launcher.Config{
		AgentLoader:     agent.NewSingleLoader(searchAgent),
		ArtifactService: artifactService,
	}

	l := full.NewLauncher()

	// Step 4: Execute with selected Interface Mode
	if err = l.Execute(ctx, launcherConfig, runArgs); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
