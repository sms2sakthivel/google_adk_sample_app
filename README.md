# ADK Agent with Ollama

This agent uses the Google Agent Development Kit (ADK) with a local Ollama model (Qwen 3).
It includes a custom local Search Tool to demonstrate tool usage without Google Search API.

## Prerequisites

1.  **Ollama**: Install [Ollama](https://ollama.com/) and ensure it is running (`ollama serve`).
2.  **Model**: Pull the Qwen 3 model (or your preferred model):
    ```bash
    ollama pull qwen3:8b
    ```

## Usage

### Interactive Setup Wizard
Run the agent without arguments to launch the setup wizard:
```bash
go run main.go
```
The wizard will guide you through:
1.  **Run Mode**: Console vs Web GUI.
2.  **LLM Provider**:
    -   **Local Ollama**: Uses defaults (`http://localhost:11434/v1`, `qwen3:8b`) or custom values.
    -   **Corporate / Private LLM**: Enter your custom Base URL, API Key, and Model Name.

### Shortcuts
You can still use shortcuts to launch quickly with default settings (Ollama):
-   `go run main.go 1` or `console`: Launch Console Mode
-   `go run main.go 2` or `webui`: Launch Web GUI Mode

Run directly in console mode:
```bash
go run main.go console
```

Then type your query, for example:
> What is the capital of France?

### Configuration

You can configure the model and host via environment variables:

-   `OLLAMA_HOST`: Defaults to `http://localhost:11434/v1`
-   `OLLAMA_MODEL`: Defaults to `qwen3:8b`

Example using a different model:
```bash
OLLAMA_MODEL=qwen2.5 go run main.go console
```
