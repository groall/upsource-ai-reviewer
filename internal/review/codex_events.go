package review

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/groall/upsource-ai-reviewer/internal/metrics"
)

// parseCodexJSONOutput extracts the final agent message from Codex's --json
// JSONL stream and records the command and Knowledge MCP outcomes observed in
// that stream. Plain-text agent commands keep their existing behavior.
func parseCodexJSONOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return output
	}

	lines := strings.Split(trimmed, "\n")
	parsedAny := false
	finalMessage := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var event struct {
			Type  string          `json:"type"`
			Item  json.RawMessage `json:"item"`
			Error json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil || event.Type == "" {
			return output
		}
		parsedAny = true

		var item map[string]json.RawMessage
		if len(event.Item) > 0 {
			_ = json.Unmarshal(event.Item, &item)
		}
		itemType := jsonString(item["type"])
		status := strings.ToLower(jsonString(item["status"]))
		failed := event.Type == "item.failed" || isFailureStatus(status) || hasError(event.Error) || hasItemError(item)

		switch itemType {
		case "command_execution":
			if event.Type == "item.completed" || event.Type == "item.failed" {
				result := metrics.CodexCapabilitySuccess
				if failed {
					result = metrics.CodexCapabilityFailure
				}
				metrics.DefaultRecorder.RecordCodexCapability(metrics.CodexCapabilityCommandExecution, result)
			}
		case "mcp_tool_call":
			if isNodaKnowledgeCall(item) && (event.Type == "item.completed" || event.Type == "item.failed") {
				result := metrics.CodexCapabilitySuccess
				if failed {
					result = metrics.CodexCapabilityFailure
				}
				metrics.DefaultRecorder.RecordCodexCapability(metrics.CodexCapabilityNodaKnowledgeMCP, result)
			}
		case "agent_message":
			if event.Type == "item.completed" {
				finalMessage = jsonString(item["text"])
			}
		}
	}

	if parsedAny && finalMessage != "" {
		return finalMessage
	}
	return output
}

func jsonString(value json.RawMessage) string {
	var result string
	_ = json.Unmarshal(value, &result)
	return result
}

func isFailureStatus(status string) bool {
	switch status {
	case "failed", "incomplete", "declined", "denied", "error":
		return true
	default:
		return false
	}
}

func hasError(value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) && !bytes.Equal(trimmed, []byte(`""`))
}

func hasItemError(item map[string]json.RawMessage) bool {
	if hasError(item["error"]) {
		return true
	}
	if hasNestedMCPError(item["result"]) {
		return true
	}
	for _, key := range []string{"is_error", "isError"} {
		var isError bool
		if json.Unmarshal(item[key], &isError) == nil && isError {
			return true
		}
	}
	return false
}

func isNodaKnowledgeCall(item map[string]json.RawMessage) bool {
	server := jsonString(item["server"])
	if server == "" {
		server = jsonString(item["server_name"])
	}
	if strings.EqualFold(server, "noda_knowledge") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(jsonString(item["name"])), "mcp__noda_knowledge__")
}

func hasNestedMCPError(value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return false
	}

	var object map[string]json.RawMessage
	if json.Unmarshal(trimmed, &object) == nil {
		if hasError(object["error"]) {
			return true
		}
		for _, key := range []string{"is_error", "isError"} {
			var isError bool
			if json.Unmarshal(object[key], &isError) == nil && isError {
				return true
			}
		}
		for _, child := range object {
			if hasNestedMCPError(child) {
				return true
			}
		}
		return false
	}

	var values []json.RawMessage
	if json.Unmarshal(trimmed, &values) == nil {
		for _, child := range values {
			if hasNestedMCPError(child) {
				return true
			}
		}
	}
	return false
}
