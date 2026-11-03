package git

type Review interface {
	GetBranch() string
	GetGitNamespaceAndName() (string, string)
}

// RepoCloner is an optional capability for providers that can clone the
// review's repository locally so an agentic LLM can explore the real code.
type RepoCloner interface {
	// PrepareReviewRepo ensures the review's repository is available under
	// baseDir, checked out at its branch, reusing an existing clone when
	// present. It returns the path to the repository working tree.
	PrepareReviewRepo(review Review, baseDir string) (string, error)
}
