# Upsource AI Reviewer

## Description

Upsource AI Reviewer is a Go application that automatically reviews code changes in Upsource using AI models from OpenAI or Gemini. It fetches code reviews from Upsource, generates review comments using an AI model, and posts them back to Upsource.

## Getting Started

### Prerequisites

- Go 1.2x installed
- Access to an Upsource instance
- Access to a GitLab instance
- An API key from OpenAI or Google Gemini

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
