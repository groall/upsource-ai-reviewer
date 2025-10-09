package git

type Review interface {
	GetDefaultBranch() string
	GetBranch() string
	GetGitGroupAndName() (string, string)
}

type Provider interface {
	GetReviewChanges(review Review) (string, string, error)
}
