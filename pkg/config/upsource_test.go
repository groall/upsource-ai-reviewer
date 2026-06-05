package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpsourceValidate(t *testing.T) {
	t.Run("succeeds for valid config", func(t *testing.T) {
		u := validUpsource()
		require.NoError(t, u.Validate())
	})

	testCases := []struct {
		name        string
		mutate      func(*Upsource)
		expectedErr string
	}{
		{
			name: "missing base url",
			mutate: func(u *Upsource) {
				u.BaseURL = ""
			},
			expectedErr: "upsource.baseUrl is required",
		},
		{
			name: "missing username",
			mutate: func(u *Upsource) {
				u.Username = ""
			},
			expectedErr: "upsource.username is required",
		},
		{
			name: "missing password",
			mutate: func(u *Upsource) {
				u.Password = ""
			},
			expectedErr: "upsource.password is required",
		},
		{
			name: "missing query",
			mutate: func(u *Upsource) {
				u.Query = ""
			},
			expectedErr: "upsource.query is required",
		},
		{
			name: "missing reviewed label",
			mutate: func(u *Upsource) {
				u.ReviewedLabel = ""
			},
			expectedErr: "upsource.reviewedLabel is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := validUpsource()
			tc.mutate(u)
			require.EqualError(t, u.Validate(), tc.expectedErr)
		})
	}
}

func validUpsource() *Upsource {
	return &Upsource{
		BaseURL:       "https://upsource.example",
		Username:      "user",
		Password:      "password",
		Query:         "state: open",
		ReviewedLabel: "AI-Reviewed",
	}
}
