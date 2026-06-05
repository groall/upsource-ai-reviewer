package config

import "fmt"

type Upsource struct {
	BaseURL         string `yaml:"baseUrl"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Query           string `yaml:"query"`
	ReviewedLabel   string `yaml:"reviewedLabel"`
	InvitationLabel string `yaml:"invitationLabel"`
}

func (u *Upsource) Validate() error {
	if u.BaseURL == "" {
		return fmt.Errorf("upsource.baseUrl is required")
	}
	if u.Username == "" {
		return fmt.Errorf("upsource.username is required")
	}
	if u.Password == "" {
		return fmt.Errorf("upsource.password is required")
	}
	if u.Query == "" {
		return fmt.Errorf("upsource.query is required")
	}
	if u.ReviewedLabel == "" {
		return fmt.Errorf("upsource.reviewedLabel is required")
	}

	return nil
}
