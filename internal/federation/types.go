package federation

// JoinKind identifies how two table legs are combined.
type JoinKind int

const (
	// InnerJoin matches rows where join keys are equal on both sides.
	InnerJoin JoinKind = iota + 1
)

// QualifiedTable is a table reference in federated SQL: connection_id.table AS alias.
type QualifiedTable struct {
	ConnectionID string // Registered connection id, e.g. "billing".
	Table        string // Table name on that connection, e.g. "users".
	Alias        string // Query alias, e.g. "u".
}

// ColumnRef identifies a column, usually via table alias and column name.
type ColumnRef struct {
	ConnectionID string // Connection id when known from qualification.
	Table        string // Table name or alias used in the query.
	Column       string // Column name.
}

// JoinSpec describes how two aliased tables are joined.
type JoinSpec struct {
	Kind       JoinKind  // Join type.
	LeftAlias  string    // Alias of the left table in the join.
	RightAlias string    // Alias of the right table in the join.
	LeftKey    ColumnRef // Join column on the left side.
	RightKey   ColumnRef // Join column on the right side.
}

// Predicate is a simple WHERE condition: column op literal value.
type Predicate struct {
	Column ColumnRef
	Op     string // Comparison operator: =, <>, <, <=, >, >=.
	Value  any    // Literal value from the query.
}

// ParsedQuery is the structured form of a federated SELECT after parsing.
type ParsedQuery struct {
	Tables     []QualifiedTable // Tables referenced in FROM/JOIN.
	Join       *JoinSpec        // Nil for a single-table query.
	SelectCols []ColumnRef      // Columns in the SELECT list.
	Where      []Predicate      // AND-ed WHERE conditions.
	Limit      *int             // LIMIT value, if present.
}

// QueryLeg is one sub-query sent to a single connection.
type QueryLeg struct {
	ConnectionID string         // Target connection for this leg.
	Table        QualifiedTable // Primary table for this leg.
	Columns      []string       // Column names to SELECT on this leg.
	Where        []Predicate    // WHERE conditions pushed down to this leg.
}

// FederatedPlan is the execution plan: legs to run plus how to join them.
type FederatedPlan struct {
	Legs       []QueryLeg  // One leg per connection involved.
	Join       *JoinSpec   // Nil for a single-table query.
	SelectCols []ColumnRef // Final output columns after the join.
	Limit      *int        // Row cap applied after joining.
}
