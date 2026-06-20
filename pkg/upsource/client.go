package upsource

import (
	"fmt"

	"github.com/groall/upsource-go-client/client"
)

func NewClient(baseURL, username, password string) (*client.Client, error) {
	upsourceClient, err := client.New(client.Options{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Upsource client: %w", err)
	}

	return upsourceClient, nil
}
