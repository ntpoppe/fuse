package testutil

import (
	"context"
	"testing"
	"time"
)

const defaultTestTimeout = 2 * time.Second

func Context(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	t.Cleanup(cancel)
	return ctx
}
