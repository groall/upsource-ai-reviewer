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

### Configuration Options
- **comments**: Configuration for the review comments.
  - `maxPerReview`: The maximum number of comments per review.
  - `highSeveritySplit`: Whether to split comments into multiple messages for high severity issues.
- **upsource**: Configuration for your Upsource instance.
  - `baseUrl`: The base URL of your Upsource instance.
  - `username`: Your Upsource username.
  - `password`: Your Upsource password.
  - `query`: The Upsource search query to find reviews to be processed.
  - `reviewedLabel`: The label to add to a review after it has been reviewed.
- **gitlab**: Configuration for your GitLab instance.
  - `baseUrl`: The base URL of your GitLab instance.
  - `accessToken`: Your GitLab personal access token.
- **codex**: Configuration for running Codex locally via a shell command.
  - `command`: Shell command that reads the prompt from stdin and prints the completion to stdout (e.g., `codex ask --quiet --input -`).
  - `workdir`: Optional working directory in which to execute the command.
  - `requestTimeout`: Timeout for the Codex command.
- **openai**: Configuration for the OpenAI API.
  - `endpoint`: The OpenAI API endpoint.
  - `model`: The OpenAI model to use for generating reviews.
  - `maxTokens`: The maximum number of tokens to generate in the completion.
  - `temperature`: The sampling temperature to use.
  - `apiKey`: Your OpenAI API key.
  - `requestTimeout`: The request timeout for the OpenAI API.
- **gemini**: Configuration for the Google Gemini API.
    - `apiKey`: Your Gemini API key.
    - `model`: The Gemini model to use.
    - `maxTokens`: The maximum number of tokens to generate.
- **polling**: Configuration for the polling interval.
  - `intervalSeconds`: The interval in seconds at which to poll Upsource for new reviews.
- **promts**: Configuration for the AI prompts.
  - `systemMessage`: The system message to send to the AI model. Please, modify it carefully to fit your needs. The message should be concise and clear, and should reflect your specific requirements for code review. Please keep the part about format and severity.
  - `userPromptTemplate`: The user prompt template. The `{diff}` placeholder will be replaced with the code diff.
