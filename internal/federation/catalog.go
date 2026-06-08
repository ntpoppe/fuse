package federation

import "github.com/ntpoppe/fuse/internal/fuseerr"

// ConnectionLookup checks whether registered connections exist by id.
type ConnectionLookup interface {
	HasConnection(id string) bool
}

// ResolveConnections verifies every connection id referenced in q exists in lookup.
func ResolveConnections(q *ParsedQuery, lookup ConnectionLookup) error {
	for _, id := range connectionIDs(q) {
		if !lookup.HasConnection(id) {
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
