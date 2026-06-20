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
       │    ├─ internal/review/generator.go      iCommentGenerator.generate() — format prompt, call LLM, parse JSON
       │    │    ├─ internal/review/agentic_generator.go  — agent mode: clone repo + run agent CLI
       │    │    └─ pkg/llm/                     CreateLLMProvider() — pick SDK provider from config
       │    │         └─ openai.go | gemini.go | anthropic.go | codex.go
       │    └─ pkg/upsource             CreateDiscussion() — post inline or general comments
       └─ reply pass (open threads)
            ├─ pkg/upsource             ListReviewedReviews() — reviews already labelled reviewedLabel
            ├─ pkg/upsource             ListReviewDiscussions() — discussions for one review
            ├─ pkg/upsource             ShouldReplyToDiscussion() — label + not-resolved + last-author + bot-cap
            ├─ internal/replies/replier.go  Replier.Reply() — threaded replies to discussions
            │    ├─ internal/replies/generator.go         — LLM reply generation
            │    └─ internal/replies/review_reply_generator.go  — build discussion context, call generator
            └─ pkg/upsource             AddDiscussionComment() — post threaded reply to the last comment
```

## Directory Structure

```
cmd/reviewer/main.go              entry point, config load, signal handling
configs/                          configuration examples (claude-settings.json, codex.config.toml, reviewer.config.yaml)
pkg/config/                       configuration parsing and validation
pkg/llm/                          LLM provider implementations (openai, gemini, anthropic, codex)
pkg/llm/provider.go               provider factory — CreateLLMProvider()
pkg/upsource/                     Upsource API client (reviews, discussions, reply predicate)
pkg/upsource/client.go            Upsource HTTP client wrapper
internal/review/                  code review generation
  reviewer.go                     main orchestration loop, Reviewer.Run()
  generator.go                    comment generation interface & SDK-based implementation
  agentic_generator.go            agent mode — clone repo + run agent CLI
  comment.go                       ReviewComment struct, severity constants
  diff_validator.go               validateCommentsAgainstDiff — line number verification
internal/replies/                 discussion reply generation
  replier.go                      reply orchestration, Replier.Reply()
  generator.go                    LLM-based reply generation
  review_reply_generator.go        build discussion context + call generator
internal/git/gitlab.go            GitLab diff fetching
internal/metrics/metrics.go       metric recording
```

## LLM Provider Selection

Priority order in `pkg/llm.CreateLLMProvider()`: **Agent → OpenAI → Gemini → Anthropic**.
Only one provider is active per run, determined by which API key / command is set.
At least one must be configured or startup fails.

### Agentic mode

When `review.activeProvider: agent`, the review runs in fully agentic mode
(`internal/review/agentic_generator.go`): for each review the repo is cloned at its
branch into a fresh dir under `agent.cloneDir` (or the OS temp dir) and the
agent command runs with that clone as its working directory, so an agentic CLI
can explore the real code. The diff is still passed in the prompt; the command
must read the prompt from stdin and print the JSON `ReviewComment` array to
stdout. The clone is removed after the run. Requires `git` on the host. SDK
providers (OpenAI/Gemini/Anthropic) keep the stateless diff-in-prompt flow.

## Configuration Overview

| Section     | Key fields                                                                                                                          |
|-------------|-------------------------------------------------------------------------------------------------------------------------------------|
| `polling`   | `intervalSeconds`                                                                                                                   |
| `review`    | `maxPerReview`, `activeProvider`, `systemMessageIntro`, `systemMessageGuidelines`, `systemMessageOutputFormat`, `userPromptTemplate` |
| `upsource`  | `baseUrl`, `username`, `password`, `query`, `reviewedLabel`, `invitationLabel`                                                      |
| `gitlab`    | `baseUrl`, `accessToken`                                                                                                            |
| `openai`    | `apiKey`, `endpoint`, `model`, `maxTokens`, `temperature`, `requestTimeout`                                                         |
| `gemini`    | `apiKey`, `model`, `maxTokens`                                                                                                      |
| `anthropic` | `apiKey`, `model`, `maxTokens`, `requestTimeout`                                                                                    |
| `agent`     | `command`, `cloneDir`, `requestTimeout`                                                                                             |
| `replies`   | `enabled`, `maxPerThread`, `activeProvider`, `systemMessage`                                                                         |

## Review Flow

1. Fetch open reviews matching `upsource.query`
2. Skip reviews already labeled with `reviewedLabel`
3. If `invitationLabel` is set, skip reviews that do not have it (opt-in mode)
4. Fetch the GitLab diff for the review's branch
5. Send diff + commit messages to the LLM
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
6. Build a thread transcript + diff context via `reviewReplyGenerator`, call `generator.reply()` (plain prose)
7. Empty response ⇒ skip posting (the LLM may choose silence on "ok/thanks")
8. Otherwise post via `AddDiscussionComment` with `parentId` = last comment's id

Idempotency comes from "the last comment in the thread is from the AI user" — no DB needed. Errors per discussion are logged and never abort the loop.

## Comment Posting

Comments are sorted by severity and capped to `review.maxPerReview`, then split into two groups:

* **Inline** — `lineVerified=true` AND `lineNumber > 0` AND `filePath` is set; posted as individual per-line Upsource discussions
* **General** — remaining comments (no line info or unverified lines); batched into one discussion

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
