package llm

type ReviewConfig struct {
	UserPromptTemplate string
	SystemMessage      string
	MaxPerReview       int
	ActiveProvider     string
}

type ReplyConfig struct {
	SystemMessage  string
	ActiveProvider string
}
