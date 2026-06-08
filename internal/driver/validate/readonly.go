package validate

import (
	"errors"
	"fmt"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

var parser = sqlparser.NewTestParser()

const (
	errEmptySQL     = "empty SQL statement"
	errSelectInto   = "SELECT INTO is not allowed"
	errReadOnlyCTE  = "read-only CTE required"
	errReadOnlyOnly = "only read queries are allowed, got %s"
)

func ReadOnlySQL(sql string, opts Options) error {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return errors.New(errEmptySQL)
	}

	stmt, _, err := parser.Parse2(trimmed)
	if err != nil {
		return fmt.Errorf("invalid SQL: %w", err)
	}

	return validateStatement(stmt, opts)
}

func validateStatement(stmt sqlparser.Statement, opts Options) error {
	switch stmtType := sqlparser.ASTToStatementType(stmt); stmtType {
	case sqlparser.StmtSelect:
		return validateSelectStatement(stmt, opts)
	case sqlparser.StmtShow:
		if !opts.AllowShow {
			return errors.New("SHOW statements are not allowed")
		}
		return nil
	case sqlparser.StmtExplain:
		if !opts.AllowExplain {
			return errors.New("EXPLAIN statements are not allowed")
		}
		return validateExplainStatement(stmt)
	default:
		return fmt.Errorf(errReadOnlyOnly, stmtType.String())
	}
}

func validateExplainStatement(stmt sqlparser.Statement) error {
	switch explain := stmt.(type) {
	case *sqlparser.ExplainStmt:
		return validateStatement(explain.Statement, Options{})
	case *sqlparser.VExplainStmt:
		return validateStatement(explain.Statement, Options{})
	case *sqlparser.ExplainTab:
		return nil
	default:
		return fmt.Errorf("unsupported EXPLAIN statement type %T", stmt)
	}
}

func validateSelectStatement(stmt sqlparser.Statement, opts Options) error {
	switch s := stmt.(type) {
	case *sqlparser.Select:
		return validateSelect(s, opts)
	case *sqlparser.Union:
		return validateUnion(s, opts)
	default:
		return fmt.Errorf("unsupported SELECT statement type %T", stmt)
	}
}

func validateSelect(sel *sqlparser.Select, opts Options) error {
	if sel.Into != nil {
		return errors.New(errSelectInto)
	}
	return validateWithClause(sel.With, opts)
}

func validateUnion(union *sqlparser.Union, opts Options) error {
	if union.Into != nil {
		return errors.New(errSelectInto)
	}
	if err := validateWithClause(union.With, opts); err != nil {
		return err
	}
	if err := validateTableStatement(union.Left, opts); err != nil {
		return err
	}
	return validateTableStatement(union.Right, opts)
}

func validateWithClause(with *sqlparser.With, opts Options) error {
	if with == nil {
		return nil
	}
	for _, cte := range with.CTEs {
		if err := validateTableStatement(cte.Subquery, opts); err != nil {
			return fmt.Errorf("%s: %w", errReadOnlyCTE, err)
		}
	}
	return nil
}

func validateTableStatement(stmt sqlparser.TableStatement, opts Options) error {
	switch s := stmt.(type) {
	case *sqlparser.Select:
		return validateSelect(s, opts)
	case *sqlparser.Union:
		return validateUnion(s, opts)
	default:
		stmtType := sqlparser.ASTToStatementType(s)
		if stmtType == sqlparser.StmtSelect {
			return validateSelectStatement(s, opts)
		}
		return fmt.Errorf(errReadOnlyOnly, stmtType.String())
	}
}
