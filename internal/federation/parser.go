package federation

import (
	"fmt"
	"strconv"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

var sqlParser = sqlparser.NewTestParser()

// Parse turns federated SQL into a ParsedQuery or returns a clear error.
func Parse(sql string) (*ParsedQuery, error) {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return nil, errEmptySQL
	}

	stmt, _, err := sqlParser.Parse2(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid SQL: %w", err)
	}

	if _, ok := stmt.(*sqlparser.Union); ok {
		return nil, errUnion
	}

	sel, ok := stmt.(*sqlparser.Select)
	if !ok {
		return nil, errNotSelect
	}

	if err := validateSelectShape(sel); err != nil {
		return nil, err
	}

	tables, join, err := parseFrom(sel.From)
	if err != nil {
		return nil, err
	}

	aliasMap := aliasByName(tables)
	selectCols, err := parseSelectList(sel.SelectExprs, aliasMap)
	if err != nil {
		return nil, err
	}

	where, err := parseWhere(sel.Where, aliasMap)
	if err != nil {
		return nil, err
	}

	limit, err := parseLimit(sel.Limit)
	if err != nil {
		return nil, err
	}

	return &ParsedQuery{
		Tables:     tables,
		Join:       join,
		SelectCols: selectCols,
		Where:      where,
		Limit:      limit,
	}, nil
}

func validateSelectShape(sel *sqlparser.Select) error {
	if sel.With != nil {
		return errWithClause
	}
	if sel.Distinct {
		return errDistinct
	}
	if sel.GroupBy != nil {
		return errGroupBy
	}
	if len(sel.OrderBy) > 0 {
		return errOrderBy
	}
	if sel.Having != nil {
		return errGroupBy
	}
	return nil
}

func parseFrom(from sqlparser.TableExprs) ([]QualifiedTable, *JoinSpec, error) {
	if len(from) == 0 {
		return nil, nil, fmt.Errorf("missing FROM clause; %s", hintFederatedSelect)
	}
	if len(from) > 1 {
		return nil, nil, errMultiTableFrom
	}

	switch expr := from[0].(type) {
	case *sqlparser.AliasedTableExpr:
		table, err := parseQualifiedTable(expr)
		if err != nil {
			return nil, nil, err
		}
		return []QualifiedTable{table}, nil, nil
	case *sqlparser.JoinTableExpr:
		return parseJoinFrom(expr)
	default:
		return nil, nil, errSubqueryInFrom
	}
}

func parseJoinFrom(join *sqlparser.JoinTableExpr) ([]QualifiedTable, *JoinSpec, error) {
	if join.Join != sqlparser.NormalJoinType {
		return nil, nil, errUnsupportedJoin
	}
	if hasNestedJoin(join.LeftExpr) || hasNestedJoin(join.RightExpr) {
		return nil, nil, errTooManyTables
	}

	left, err := parseTableOperand(join.LeftExpr)
	if err != nil {
		return nil, nil, err
	}
	right, err := parseTableOperand(join.RightExpr)
	if err != nil {
		return nil, nil, err
	}

	joinSpec, err := parseJoinCondition(join.Condition, left.Alias, right.Alias, aliasByName([]QualifiedTable{left, right}))
	if err != nil {
		return nil, nil, err
	}

	return []QualifiedTable{left, right}, joinSpec, nil
}

func hasNestedJoin(expr sqlparser.TableExpr) bool {
	switch e := expr.(type) {
	case *sqlparser.JoinTableExpr:
		return true
	case *sqlparser.AliasedTableExpr:
		if _, ok := e.Expr.(*sqlparser.DerivedTable); ok {
			return true
		}
	}
	return false
}

func parseTableOperand(expr sqlparser.TableExpr) (QualifiedTable, error) {
	aliased, ok := expr.(*sqlparser.AliasedTableExpr)
	if !ok {
		return QualifiedTable{}, errSubqueryInFrom
	}
	return parseQualifiedTable(aliased)
}

func parseQualifiedTable(aliased *sqlparser.AliasedTableExpr) (QualifiedTable, error) {
	tableName, ok := aliased.Expr.(sqlparser.TableName)
	if !ok {
		return QualifiedTable{}, errSubqueryInFrom
	}
	if tableName.Qualifier.IsEmpty() {
		return QualifiedTable{}, errUnqualifiedTable
	}

	alias := aliased.As.String()
	if alias == "" {
		alias = tableName.Name.String()
	}

	return QualifiedTable{
		ConnectionID: tableName.Qualifier.String(),
		Table:        tableName.Name.String(),
		Alias:        alias,
	}, nil
}

func parseJoinCondition(cond *sqlparser.JoinCondition, leftAlias, rightAlias string, aliases map[string]QualifiedTable) (*JoinSpec, error) {
	if cond == nil {
		return nil, errNonEquiJoinOn
	}
	if len(cond.Using) > 0 {
		return nil, errJoinUsing
	}

	cmp, ok := cond.On.(*sqlparser.ComparisonExpr)
	if !ok {
		return nil, errCompoundJoinOn
	}
	if cmp.Operator != sqlparser.EqualOp {
		return nil, errNonEquiJoinOn
	}

	leftKey, err := parseJoinColumn(cmp.Left, aliases)
	if err != nil {
		return nil, err
	}
	rightKey, err := parseJoinColumn(cmp.Right, aliases)
	if err != nil {
		return nil, err
	}

	return &JoinSpec{
		Kind:       InnerJoin,
		LeftAlias:  leftAlias,
		RightAlias: rightAlias,
		LeftKey:    leftKey,
		RightKey:   rightKey,
	}, nil
}

func parseJoinColumn(expr sqlparser.Expr, aliases map[string]QualifiedTable) (ColumnRef, error) {
	col, ok := expr.(*sqlparser.ColName)
	if !ok {
		return ColumnRef{}, errJoinColumnRef
	}
	return columnRefFromColName(col, aliases)
}

func parseSelectList(selectExprs *sqlparser.SelectExprs, aliases map[string]QualifiedTable) ([]ColumnRef, error) {
	if selectExprs == nil || len(selectExprs.Exprs) == 0 {
		return nil, fmt.Errorf("SELECT must list at least one column; %s", hintFederatedSelect)
	}

	cols := make([]ColumnRef, 0, len(selectExprs.Exprs))
	for _, expr := range selectExprs.Exprs {
		switch e := expr.(type) {
		case *sqlparser.StarExpr:
			return nil, errSelectStar
		case *sqlparser.AliasedExpr:
			if !e.As.IsEmpty() {
				return nil, errSelectExpression
			}
			col, ok := e.Expr.(*sqlparser.ColName)
			if !ok {
				return nil, errSelectExpression
			}
			ref, err := columnRefFromColName(col, aliases)
			if err != nil {
				return nil, err
			}
			cols = append(cols, ref)
		default:
			return nil, errSelectExpression
		}
	}
	return cols, nil
}

func parseWhere(where *sqlparser.Where, aliases map[string]QualifiedTable) ([]Predicate, error) {
	if where == nil {
		return nil, nil
	}

	preds, err := flattenAndExprs(where.Expr)
	if err != nil {
		return nil, err
	}

	out := make([]Predicate, 0, len(preds))
	for _, pred := range preds {
		p, err := parsePredicate(pred, aliases)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func flattenAndExprs(expr sqlparser.Expr) ([]sqlparser.Expr, error) {
	switch e := expr.(type) {
	case *sqlparser.AndExpr:
		left, err := flattenAndExprs(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := flattenAndExprs(e.Right)
		if err != nil {
			return nil, err
		}
		return append(left, right...), nil
	case *sqlparser.OrExpr:
		return nil, errWhereOr
	case *sqlparser.NotExpr:
		return nil, errWhereNot
	default:
		return []sqlparser.Expr{expr}, nil
	}
}

func parsePredicate(expr sqlparser.Expr, aliases map[string]QualifiedTable) (Predicate, error) {
	cmp, ok := expr.(*sqlparser.ComparisonExpr)
	if !ok {
		return Predicate{}, errWhereExpression
	}

	op, err := comparisonOperator(cmp.Operator)
	if err != nil {
		return Predicate{}, err
	}

	col, ok := cmp.Left.(*sqlparser.ColName)
	if !ok {
		return Predicate{}, errWhereExpression
	}

	ref, err := columnRefFromColName(col, aliases)
	if err != nil {
		return Predicate{}, err
	}

	value, err := literalValue(cmp.Right)
	if err != nil {
		return Predicate{}, err
	}

	return Predicate{
		Column: ref,
		Op:     op,
		Value:  value,
	}, nil
}

func comparisonOperator(op sqlparser.ComparisonExprOperator) (string, error) {
	switch op {
	case sqlparser.EqualOp:
		return "=", nil
	case sqlparser.NotEqualOp:
		return "<>", nil
	case sqlparser.LessThanOp:
		return "<", nil
	case sqlparser.LessEqualOp:
		return "<=", nil
	case sqlparser.GreaterThanOp:
		return ">", nil
	case sqlparser.GreaterEqualOp:
		return ">=", nil
	default:
		return "", errUnsupportedWhereOp
	}
}

func literalValue(expr sqlparser.Expr) (any, error) {
	switch v := expr.(type) {
	case *sqlparser.Literal:
		switch v.Type {
		case sqlparser.IntVal:
			n, err := strconv.Atoi(v.Val)
			if err != nil {
				return nil, errWhereExpression
			}
			return n, nil
		case sqlparser.FloatVal, sqlparser.DecimalVal:
			f, err := strconv.ParseFloat(v.Val, 64)
			if err != nil {
				return nil, errWhereExpression
			}
			return f, nil
		case sqlparser.StrVal:
			return v.Val, nil
		default:
			return nil, errWhereExpression
		}
	case sqlparser.BoolVal:
		return bool(v), nil
	case *sqlparser.NullVal:
		return nil, nil
	default:
		return nil, errWhereExpression
	}
}

func parseLimit(limit *sqlparser.Limit) (*int, error) {
	if limit == nil {
		return nil, nil
	}
	if limit.Offset != nil {
		return nil, errLimitOffset
	}
	if limit.Rowcount == nil {
		return nil, errLimitLiteral
	}

	lit, ok := limit.Rowcount.(*sqlparser.Literal)
	if !ok || lit.Type != sqlparser.IntVal {
		return nil, errLimitLiteral
	}

	n, err := strconv.Atoi(lit.Val)
	if err != nil || n < 0 {
		return nil, errNegativeLimit
	}
	return &n, nil
}

func columnRefFromColName(col *sqlparser.ColName, aliases map[string]QualifiedTable) (ColumnRef, error) {
	tableKey := tableKeyFromColName(col)
	if tableKey == "" {
		return ColumnRef{}, errUnqualifiedColumn
	}

	ref := ColumnRef{
		Table:  tableKey,
		Column: col.Name.String(),
	}

	if qt, ok := aliases[tableKey]; ok {
		ref.ConnectionID = qt.ConnectionID
		ref.Table = qt.Alias
	}

	return ref, nil
}

func tableKeyFromColName(col *sqlparser.ColName) string {
	if !col.Qualifier.Name.IsEmpty() {
		return col.Qualifier.Name.String()
	}
	if !col.Qualifier.Qualifier.IsEmpty() {
		return col.Qualifier.Qualifier.String() + "." + col.Qualifier.Name.String()
	}
	return ""
}

func aliasByName(tables []QualifiedTable) map[string]QualifiedTable {
	m := make(map[string]QualifiedTable, len(tables))
	for _, t := range tables {
		m[t.Alias] = t
	}
	return m
}
