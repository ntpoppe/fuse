package federation

import (
	"github.com/ntpoppe/fuse/internal/driver"
	"github.com/ntpoppe/fuse/internal/fuseerr"
)

// ConnectionLookup resolves registered connections by id.
type ConnectionLookup interface {
	Fetch(id string) (driver.Target, bool)
}

// ResolveConnections verifies every connection id referenced in q exists in lookup.
func ResolveConnections(q *ParsedQuery, lookup ConnectionLookup) error {
	for _, id := range connectionIDs(q) {
		if _, ok := lookup.Fetch(id); !ok {
			return fuseerr.NotFoundError{ID: id}
		}
	}
	return nil
}

func connectionIDs(q *ParsedQuery) []string {
	if q == nil {
		return nil
	}

	seen := make(map[string]struct{}, len(q.Tables))
	ids := make([]string, 0, len(q.Tables))
	for _, table := range q.Tables {
		if table.ConnectionID == "" {
			continue
		}
		if _, ok := seen[table.ConnectionID]; ok {
			continue
		}
		seen[table.ConnectionID] = struct{}{}
		ids = append(ids, table.ConnectionID)
	}
	return ids
}
