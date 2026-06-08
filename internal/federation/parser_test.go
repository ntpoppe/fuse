package federation

import (
	"errors"
	"testing"
)

const validJoinSQL = `
SELECT u.id, u.name, o.total
FROM billing.users AS u
JOIN analytics.orders AS o ON u.id = o.user_id
WHERE u.active = 1
LIMIT 100
`

func TestParseValidTwoTableJoin(t *testing.T) {
	q, err := Parse(validJoinSQL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(q.Tables) != 2 {
		t.Fatalf("Tables len = %d, want 2", len(q.Tables))
	}
	if q.Tables[0].ConnectionID != "billing" || q.Tables[0].Table != "users" || q.Tables[0].Alias != "u" {
		t.Fatalf("left table = %+v", q.Tables[0])
	}
	if q.Tables[1].ConnectionID != "analytics" || q.Tables[1].Table != "orders" || q.Tables[1].Alias != "o" {
		t.Fatalf("right table = %+v", q.Tables[1])
	}

	if q.Join == nil {
		t.Fatal("Join is nil")
	}
	if q.Join.Kind != InnerJoin {
		t.Fatalf("Join.Kind = %v, want InnerJoin", q.Join.Kind)
	}
	if q.Join.LeftAlias != "u" || q.Join.RightAlias != "o" {
		t.Fatalf("join aliases = %q, %q", q.Join.LeftAlias, q.Join.RightAlias)
	}
	if q.Join.LeftKey.Column != "id" || q.Join.RightKey.Column != "user_id" {
		t.Fatalf("join keys = %+v, %+v", q.Join.LeftKey, q.Join.RightKey)
	}

	if len(q.SelectCols) != 3 {
		t.Fatalf("SelectCols len = %d, want 3", len(q.SelectCols))
	}
	if len(q.Where) != 1 {
		t.Fatalf("Where len = %d, want 1", len(q.Where))
	}
	if q.Where[0].Op != "=" || q.Where[0].Value != 1 {
		t.Fatalf("Where[0] = %+v", q.Where[0])
	}
	if q.Limit == nil || *q.Limit != 100 {
		t.Fatalf("Limit = %v, want 100", q.Limit)
	}
}

func TestParseValidSingleTable(t *testing.T) {
	q, err := Parse(`SELECT u.id FROM billing.users u WHERE u.active = 0`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if q.Join != nil {
		t.Fatal("expected no join")
	}
	if len(q.Tables) != 1 {
		t.Fatalf("Tables len = %d, want 1", len(q.Tables))
	}
	if q.Tables[0].Alias != "u" {
		t.Fatalf("alias = %q, want u", q.Tables[0].Alias)
	}
}

func TestParseUnqualifiedTable(t *testing.T) {
	_, err := Parse(`SELECT u.id FROM users AS u`)
	if !errors.Is(err, errUnqualifiedTable) {
		t.Fatalf("error = %v, want errUnqualifiedTable", err)
	}
}

func TestParseLeftJoinRejected(t *testing.T) {
	sql := `
SELECT u.id, o.total
FROM billing.users u
LEFT JOIN analytics.orders o ON u.id = o.user_id`
	_, err := Parse(sql)
	if !errors.Is(err, errUnsupportedJoin) {
		t.Fatalf("error = %v, want errUnsupportedJoin", err)
	}
}

func TestParseGroupByRejected(t *testing.T) {
	sql := `
SELECT u.id, COUNT(*)
FROM billing.users u
GROUP BY u.id`
	_, err := Parse(sql)
	if !errors.Is(err, errGroupBy) {
		t.Fatalf("error = %v, want errGroupBy", err)
	}
}

func TestParseSelectStarRejected(t *testing.T) {
	sql := `
SELECT *
FROM billing.users u
JOIN analytics.orders o ON u.id = o.user_id`
	_, err := Parse(sql)
	if !errors.Is(err, errSelectStar) {
		t.Fatalf("error = %v, want errSelectStar", err)
	}
}

func TestParseCompoundJoinOnRejected(t *testing.T) {
	sql := `
SELECT u.id, o.total
FROM billing.users u
JOIN analytics.orders o ON u.id = o.user_id AND o.status = 1`
	_, err := Parse(sql)
	if !errors.Is(err, errCompoundJoinOn) {
		t.Fatalf("error = %v, want errCompoundJoinOn", err)
	}
}

func TestParseUnionRejected(t *testing.T) {
	_, err := Parse(`SELECT billing.users.id FROM billing.users u UNION SELECT analytics.orders.user_id FROM analytics.orders o`)
	if !errors.Is(err, errUnion) {
		t.Fatalf("error = %v, want errUnion", err)
	}
}

func TestParseEmptySQL(t *testing.T) {
	_, err := Parse("   ")
	if !errors.Is(err, errEmptySQL) {
		t.Fatalf("error = %v, want errEmptySQL", err)
	}
}
