package federation_test

import (
	"errors"
	"testing"

	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/federation"
	"github.com/ntpoppe/fuse/internal/fuseerr"
	"github.com/ntpoppe/fuse/internal/registry"
	"github.com/ntpoppe/fuse/internal/testutil"
)

type stubLookup map[string]driver.Target

func (s stubLookup) Fetch(id string) (driver.Target, bool) {
	target, ok := s[id]
	return target, ok
}

func TestResolveConnectionsAllPresent(t *testing.T) {
	q := &federation.ParsedQuery{
		Tables: []federation.QualifiedTable{
			{ConnectionID: "billing", Table: "users", Alias: "u"},
			{ConnectionID: "analytics", Table: "orders", Alias: "o"},
		},
	}

	lookup := stubLookup{
		"billing":   testutil.NewStubTarget("billing"),
		"analytics": testutil.NewStubTarget("analytics"),
	}

	if err := federation.ResolveConnections(q, lookup); err != nil {
		t.Fatalf("ResolveConnections() error = %v", err)
	}
}

func TestResolveConnectionsSameConnectionTwice(t *testing.T) {
	q := &federation.ParsedQuery{
		Tables: []federation.QualifiedTable{
			{ConnectionID: "billing", Table: "users", Alias: "u"},
			{ConnectionID: "billing", Table: "orders", Alias: "o"},
		},
	}

	lookup := stubLookup{
		"billing": testutil.NewStubTarget("billing"),
	}

	if err := federation.ResolveConnections(q, lookup); err != nil {
		t.Fatalf("ResolveConnections() error = %v", err)
	}
}

func TestResolveConnectionsMissing(t *testing.T) {
	q := &federation.ParsedQuery{
		Tables: []federation.QualifiedTable{
			{ConnectionID: "billing", Table: "users", Alias: "u"},
			{ConnectionID: "analytics", Table: "orders", Alias: "o"},
		},
	}

	lookup := stubLookup{
		"billing": testutil.NewStubTarget("billing"),
	}

	err := federation.ResolveConnections(q, lookup)
	var notFound fuseerr.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v, want NotFoundError", err)
	}
	if notFound.ID != "analytics" {
		t.Fatalf("NotFoundError.ID = %q, want analytics", notFound.ID)
	}
	if !errors.Is(err, fuseerr.ErrNotFound) {
		t.Fatalf("errors.Is(err, ErrNotFound) = false")
	}
}

func TestResolveConnectionsWithRegistry(t *testing.T) {
	reg := registry.NewRegistry()
	reg.Save("billing", testutil.NewStubTarget("billing"))

	q := &federation.ParsedQuery{
		Tables: []federation.QualifiedTable{
			{ConnectionID: "billing", Table: "users", Alias: "u"},
		},
	}

	if err := federation.ResolveConnections(q, reg); err != nil {
		t.Fatalf("ResolveConnections() error = %v", err)
	}
}

func TestRegistryImplementsConnectionLookup(t *testing.T) {
	var _ federation.ConnectionLookup = (*registry.Registry)(nil)
}
