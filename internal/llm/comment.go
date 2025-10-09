package llm

// ReviewComment matches the JSON structure we requested from the LLM.
type ReviewComment struct {
	FilePath   string `json:"filePath"`   // Path to the file where the comment is made.
	LineNumber int    `json:"lineNumber"` // Line number in the file where the comment is made.
	Comment    string `json:"comment"`    // The actual comment text.
	Severity   string `json:"severity"`   // Severity of the comment, can be "low", "medium", or "high".
}

const (
	SeverityLow    = "low"
	SeverityMedium = "medium"
	SeverityHigh   = "high"
)
