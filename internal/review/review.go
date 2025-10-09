package review

type Review interface {
	GetDefaultBranch() string
	GetBranch() string
	GetGitGroupAndName() (string, string)
}
