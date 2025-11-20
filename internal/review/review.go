package review

type Review interface {
	GetDefaultBranch() string
	GetBranch() string
	GetGitNamespaceAndName() (string, string)
}
