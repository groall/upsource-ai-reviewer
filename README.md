# Upsource AI Reviewer

## Description

Upsource AI Reviewer is a Go application that automatically reviews code changes in Upsource using AI models from OpenAI, Gemini, Anthropic, or a local Codex command. It fetches code reviews from Upsource, generates review comments using an AI model, and posts them back to Upsource.

In addition, when `replies.enabled` is set, the bot scans the discussions it previously authored and posts a threaded follow-up whenever a human commented after its last word. A per-thread cap (`replies.maxPerThread`) prevents runaway loops, and an empty LLM response is treated as a deliberate "stay silent".

## Getting Started

### Prerequisites

- Go 1.2x installed
- Access to an Upsource instance
- Access to a GitLab instance
- An API key from OpenAI, Google Gemini, or Anthropic (or a Codex command)

### Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd upsource-ai-reviewer
   ```

2. Build the application:
   ```bash
   go build -o reviewer ./cmd/reviewer
   ```

## Usage

1. Create a `config.yaml` file. You can use the example below as a template.

2. Run the application with the path to your configuration file:
   ```bash
   ./reviewer -config path/to/your/config.yaml
   ```

## Configuration

The application is configured using a YAML file. An example of the `config.yaml` file you can find in `configs/config.example.yaml`.
You can copy this file and modify it according to your needs.

The main review behavior is configured under the `review` section (prompt templates plus `maxPerReview` and `postInLine`).
