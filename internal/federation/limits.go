package federation

import "errors"

// v1 capability hints included in parse errors so callers know what is supported.
const (
	hintFederatedSelect = "federated queries support a single SELECT with 1-2 qualified tables (connection_id.table), INNER JOIN, equi-join ON (=), AND-only WHERE with column op literal, explicit columns, and LIMIT"
)

var (
	errEmptySQL           = errors.New("empty SQL statement")
	errNotSelect          = errors.New("only SELECT is supported; " + hintFederatedSelect)
	errWithClause         = errors.New("WITH (CTE) is not supported yet; " + hintFederatedSelect)
	errUnion              = errors.New("UNION is not supported yet; " + hintFederatedSelect)
	errDistinct           = errors.New("SELECT DISTINCT is not supported yet; " + hintFederatedSelect)
	errGroupBy            = errors.New("GROUP BY is not supported yet; " + hintFederatedSelect)
	errHaving             = errors.New("HAVING is not supported yet; " + hintFederatedSelect)
	errOrderBy            = errors.New("ORDER BY is not supported yet; " + hintFederatedSelect)
	errSelectStar         = errors.New("SELECT * is not supported; list columns explicitly, e.g. u.id, u.name")
	errSelectExpression   = errors.New("SELECT expressions are not supported yet; use qualified column names only")
	errUnqualifiedTable   = errors.New("tables must be qualified as connection_id.table, e.g. billing.users")
	errSubqueryInFrom     = errors.New("subqueries in FROM are not supported yet; " + hintFederatedSelect)
	errMultiTableFrom     = errors.New("comma-separated FROM tables are not supported; use INNER JOIN")
	errTooManyTables      = errors.New("more than two tables are not supported yet; " + hintFederatedSelect)
	errUnsupportedJoin    = errors.New("only INNER JOIN is supported in v1; LEFT/RIGHT/FULL joins are not supported yet")
	errJoinUsing          = errors.New("JOIN USING is not supported; use ON with a single = between two columns")
	errCompoundJoinOn     = errors.New("JOIN ON must be a single = between two columns; AND in ON is not supported yet")
	errNonEquiJoinOn      = errors.New("JOIN ON must use = between two columns; " + hintFederatedSelect)
	errJoinColumnRef      = errors.New("JOIN ON must compare two column references, e.g. u.id = o.user_id")
	errWhereOr            = errors.New("OR in WHERE is not supported; use AND only")
	errWhereNot           = errors.New("NOT in WHERE is not supported yet")
	errWhereExpression    = errors.New("WHERE must be column op literal conditions joined by AND")
	errUnqualifiedColumn  = errors.New("columns must be qualified with a table alias, e.g. u.id")
	errUnsupportedWhereOp = errors.New("WHERE supports =, <>, <, <=, >, >= with a literal value only")
	errLimitOffset        = errors.New("LIMIT OFFSET is not supported yet; use LIMIT n only")
	errLimitLiteral       = errors.New("LIMIT must be a non-negative integer literal")
	errNegativeLimit      = errors.New("LIMIT must be non-negative")
)
