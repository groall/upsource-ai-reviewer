package llm

import (
	"context"
	"time"
)

func withRequestTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(parent, timeout)
	}

	return context.WithCancel(parent)
}
