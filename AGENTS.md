This project is an AI-based code review tool for Upsource.

Upsource AI Reviewer polls JetBrains Upsource for open reviews, fetches the corresponding diff from GitLab, sends it to an LLM, and posts the review comments back to Upsource. It also follows up on threads it previously authored, replying when a human has commented after the bot's last word.

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
       ├─ review pass (new MRs)
       │    ├─ pkg/upsource             ListReviews() — filter by reviewedLabel / invitationLabel
       │    ├─ internal/git/gitlab.go   GetReviewChanges() — fetch diff via GitLab branch compare
       │    ├─ internal/llm/reviewer.go Reviewer.Do() — format prompt, call LLM, parse JSON
       │    │    └─ internal/llm/lllm_provider.go  createLLMProvider() — pick provider from config
       │    │         └─ pkg/llm/       openai.go | gemini.go | anthropic.go | codex.go
       │    └─ pkg/upsource             CreateDiscussion() — post inline or general comments
       └─ reply pass (open threads)
            ├─ pkg/upsource             ListReviewedReviews() — reviews already labelled reviewedLabel
            ├─ pkg/upsource             ListReviewDiscussions() — discussions for one review
            ├─ pkg/upsource             ShouldReplyToDiscussion() — label + not-resolved + last-author + bot-cap
            ├─ internal/llm/replier.go  Reviewer.Reply() — plain-prose reply via the same LLM provider
            └─ pkg/upsource             AddDiscussionComment() — threaded reply to the last comment
```

## Directory Structure

```
cmd/reviewer/main.go           entry point, config load, signal handling
pkg/config/config.go           Config struct, LoadConfig, ValidateConfig
pkg/llm/                       LLM provider implementations
pkg/upsource/                  Upsource API client (reviews, discussions, reply predicate)
internal/review/reviewer.go    main orchestration loop
internal/review/replier.go     reply pass — scan reviewed reviews, post follow-up replies
internal/llm/reviewer.go       prompt formatting, LLM call, JSON parsing
internal/llm/replier.go        reply-shaped LLM call (plain prose, no JSON)
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
| `replies` | `enabled`, `maxPerThread`, `systemMessage`, `userPromptTemplate` (`%s` code context, `%s` thread transcript) |

## Review Flow

1. Fetch open reviews matching `upsource.query`
2. Skip reviews already labeled with `reviewedLabel`
3. If `invitationLabel` is set, skip reviews that do not have it (opt-in mode)
4. Fetch the GitLab diff for the review's branch
5. Send diff + commit messages to the LLM using `promts` templates
6. Parse the JSON array response: `[{filePath, lineNumber, lineVerified, comment, severity}]`
7. Validate reported line numbers against the actual diff (`diff_validator.go`)
8. Post comments to Upsource; add `reviewedLabel` to prevent re-processing

## Reply Flow (when `replies.enabled: true`)

Runs after the review pass on every tick.

1. Resolve and cache the bot's own user id via `GetCurrentUser` (once per process)
2. List open reviews that already carry `reviewedLabel` (`ListReviewedReviews`)
3. For each review, list its discussions (`ListReviewDiscussions`)
4. For each discussion, apply `ShouldReplyToDiscussion`:
   - has `reviewedLabel`
   - not resolved
   - has at least one comment
   - last comment was **not** from the bot
   - bot has authored fewer than `replies.maxPerThread` comments in the thread
5. Lazily fetch the review diff once per review when at least one discussion qualifies
6. Build a thread transcript + diff context, call `Reviewer.Reply` (plain prose)
7. Empty response ⇒ skip posting (the LLM may choose silence on "ok/thanks")
8. Otherwise post via `AddDiscussionComment` with `parentId` = last comment's id

Idempotency comes from "the last comment in the thread is from the AI user" — no DB needed. Errors per discussion are logged and never abort the loop.

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
* Threaded follow-up replies — the bot answers humans who responded in its own threads, capped per-thread to avoid runaway loops

## RULES

* do not explain the code
* do not explain the code changes