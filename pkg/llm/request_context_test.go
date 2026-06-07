package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithRequestTimeoutZeroHasNoDeadline(t *testing.T) {
	ctx, cancel := withRequestTimeout(context.Background(), 0)
	defer cancel()

	_, hasDeadline := ctx.Deadline()
	require.False(t, hasDeadline)
	require.NoError(t, ctx.Err())
}

func TestWithRequestTimeoutPositiveSetsDeadline(t *testing.T) {
	timeout := 50 * time.Millisecond
	ctx, cancel := withRequestTimeout(context.Background(), timeout)
	defer cancel()

	deadline, hasDeadline := ctx.Deadline()
	require.True(t, hasDeadline)
	require.WithinDuration(t, time.Now().Add(timeout), deadline, 150*time.Millisecond)
}

func TestWithRequestTimeoutHonorsParentCancel(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	ctx, cancel := withRequestTimeout(parent, 0)
	defer cancel()

	parentCancel()

	select {
	case <-ctx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("context did not propagate parent cancellation")
	}
}
