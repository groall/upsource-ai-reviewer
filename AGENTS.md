This project is an AI-based code review tool for Upsource.

Upsource AI Reviewer polls JetBrains Upsource for open reviews, fetches the corresponding diff from GitLab, sends it to an LLM, and posts the review comments back to Upsource.

## Tech

* Go
* OpenAI API
* Gemini API
* Anthropic API
* Upsource API
* GitLab API
* Codex

## Architecture

```
main.go (ticker)
  └─ internal/review/reviewer.go  Reviewer.Run()
       ├─ pkg/upsource             ListReviews() — filter by reviewedLabel / invitationLabel
       ├─ internal/git/gitlab.go   GetReviewChanges() — fetch diff via GitLab branch compare
       ├─ internal/llm/reviewer.go Reviewer.Do() — format prompt, call LLM, parse JSON
       │    └─ internal/llm/lllm_provider.go  createLLMProvider() — pick provider from config
       │         └─ pkg/llm/       openai.go | gemini.go | anthropic.go | codex.go
       └─ pkg/upsource             CreateDiscussion() — post inline or general comments
```

## Directory Structure

```
cmd/reviewer/main.go           entry point, config load, signal handling
pkg/config/config.go           Config struct, LoadConfig, ValidateConfig
pkg/llm/                       LLM provider implementations
pkg/upsource/                  Upsource API client
internal/review/reviewer.go    main orchestration loop
internal/llm/reviewer.go       prompt formatting, LLM call, JSON parsing
internal/llm/lllm_provider.go  provider factory
internal/llm/comment.go        ReviewComment struct, severity constants
internal/llm/diff_validator.go validateCommentsAgainstDiff — line number verification
internal/git/gitlab.go         GitLab diff fetching
config.yaml.example            reference configuration
```

## LLM Provider Selection

Priority order in `createLLMProvider()`: **Codex → OpenAI → Gemini → Anthropic**.
Only one provider is active per run, determined by which API key / command is set.
At least one must be configured or startup fails.

## Configuration Overview

| Section | Key fields |
|---|---|
| `polling` | `intervalSeconds` |
| `comments` | `maxPerReview`, `postInLine` (high / mid / low / none) |
| `upsource` | `baseUrl`, `username`, `password`, `query`, `reviewedLabel`, `invitationLabel` |
| `gitlab` | `baseUrl`, `accessToken` |
| `openai` | `apiKey`, `endpoint`, `model`, `maxTokens`, `temperature`, `requestTimeout` |
| `gemini` | `apiKey`, `model`, `maxTokens` |
| `anthropic` | `apiKey`, `model`, `maxTokens`, `requestTimeout` |
| `codex` | `command`, `workdir`, `requestTimeout` |
| `promts` | `systemMessage` (`%d` for maxPerReview), `userPromptTemplate` (`%s` diff, `%s` commits) |

## Review Flow

1. Fetch open reviews matching `upsource.query`
2. Skip reviews already labeled with `reviewedLabel`
3. If `invitationLabel` is set, skip reviews that do not have it (opt-in mode)
4. Fetch the GitLab diff for the review's branch
5. Send diff + commit messages to the LLM using `promts` templates
6. Parse the JSON array response: `[{filePath, lineNumber, lineVerified, comment, severity}]`
7. Validate reported line numbers against the actual diff (`diff_validator.go`)
8. Post comments to Upsource; add `reviewedLabel` to prevent re-processing

## Comment Posting

Comments are split into two groups based on `comments.postInLine`:

* **Inline** — severity meets the threshold AND `lineVerified=true` AND `lineNumber > 0`; posted as individual per-line Upsource discussions
* **General** — remaining comments batched into one discussion formatted as `### Low-Medium Priority Comments (AI generated)`

## LLM Response Format

The LLM must return a JSON array only (no prose). Each element:
```json
{
  "filePath": "path/to/file.go",
  "lineNumber": 42,
  "lineVerified": true,
  "comment": "...",
  "severity": "high"
}
```
Severity values: `low`, `medium`, `high`.
JSON is extracted from the raw response by finding the outermost `[...]` block.

## Features

* AI-based code review using configurable LLM providers
* Automatic inline and general comment generation
* Line number verification against the actual diff
* Severity-based comment filtering
* Invitation label opt-in model for selective review processing
* Duplicate prevention via reviewed label

## RULES

* do not explain the code
* do not explain the code changes