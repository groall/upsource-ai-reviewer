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

func TestUpsourceValidateAcceptsWhitespace(t *testing.T) {
	t.Run("accepts baseUrl with whitespace", func(t *testing.T) {
		u := validUpsource()
		u.BaseURL = "   https://upsource.example   "
		require.NoError(t, u.Validate())
	})

	t.Run("accepts username with whitespace", func(t *testing.T) {
		u := validUpsource()
		u.Username = "   user   "
		require.NoError(t, u.Validate())
	})

	t.Run("accepts password with whitespace", func(t *testing.T) {
		u := validUpsource()
		u.Password = "   password   "
		require.NoError(t, u.Validate())
	})

	t.Run("accepts query with whitespace", func(t *testing.T) {
		u := validUpsource()
		u.Query = "   state: open   "
		require.NoError(t, u.Validate())
	})

	t.Run("accepts reviewedLabel with whitespace", func(t *testing.T) {
		u := validUpsource()
		u.ReviewedLabel = "   AI-Reviewed   "
		require.NoError(t, u.Validate())
	})
}

func TestUpsourceValidateWithVaryingValues(t *testing.T) {
	t.Run("succeeds with various valid URLs", func(t *testing.T) {
		urls := []string{
			"https://upsource.example",
			"http://upsource.example:8080",
			"https://upsource.example/path",
		}
		for _, url := range urls {
			u := validUpsource()
			u.BaseURL = url
			require.NoError(t, u.Validate())
		}
	})

	t.Run("succeeds with various queries", func(t *testing.T) {
		queries := []string{
			"state: open",
			"state: OPEN AND author:user",
			"created: today",
			"resolved: false",
		}
		for _, query := range queries {
			u := validUpsource()
			u.Query = query
			require.NoError(t, u.Validate())
		}
	})

	t.Run("succeeds with various labels", func(t *testing.T) {
		labels := []string{
			"AI-Reviewed",
			"reviewed",
			"AI_REVIEWED_v2",
			"label-with-dashes",
		}
		for _, label := range labels {
			u := validUpsource()
			u.ReviewedLabel = label
			require.NoError(t, u.Validate())
		}
	})
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
