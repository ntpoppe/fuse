package executor_test

import (
	"errors"
	"testing"

	"github.com/ntpoppe/fuse/internal/executor"
	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/testutil"
)

const validFederatedSQL = `
SELECT u.id, u.name, o.total
FROM billing.users u
JOIN analytics.orders o ON u.id = o.user_id
WHERE u.active = 1
LIMIT 100
`

func TestFederatedExecutorValidSQLNotImplemented(t *testing.T) {
	reg := registry.NewRegistry()
	reg.Save("billing", testutil.NewStubTarget("billing"))
	reg.Save("analytics", testutil.NewStubTarget("analytics"))

	fed := executor.NewFederatedExecutor(reg)
	_, err := fed.ExecuteFederatedQuery(testutil.Context(t), validFederatedSQL)
	if !errors.Is(err, executor.ErrNotImplemented) {
		t.Fatalf("error = %v, want ErrNotImplemented", err)
	}
}

func TestFederatedExecutorInvalidSQL(t *testing.T) {
	fed := executor.NewFederatedExecutor(registry.NewRegistry())
	_, err := fed.ExecuteFederatedQuery(testutil.Context(t), `SELECT u.id FROM users u`)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, executor.ErrNotImplemented) {
		t.Fatalf("error = %v, want parse error", err)
	}
}

func TestFederatedExecutorUnknownConnection(t *testing.T) {
	reg := registry.NewRegistry()
	reg.Save("billing", testutil.NewStubTarget("billing"))

	fed := executor.NewFederatedExecutor(reg)
	_, err := fed.ExecuteFederatedQuery(testutil.Context(t), validFederatedSQL)
	var notFound fuseerr.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v, want NotFoundError", err)
	}
	if notFound.ID != "analytics" {
		t.Fatalf("NotFoundError.ID = %q, want analytics", notFound.ID)
	}
}
