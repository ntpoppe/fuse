package federation

import (
	"reflect"
	"testing"
)

func sampleJoinSpec() JoinSpec {
	return JoinSpec{
		Kind:       InnerJoin,
		LeftAlias:  "u",
		RightAlias: "o",
		LeftKey:    ColumnRef{Table: "u", Column: "id"},
		RightKey:   ColumnRef{Table: "o", Column: "user_id"},
	}
}

func sampleSelectCols() []ColumnRef {
	return []ColumnRef{
		{Table: "u", Column: "id"},
		{Table: "u", Column: "name"},
		{Table: "o", Column: "total"},
	}
}

func TestHashJoinMatchesRows(t *testing.T) {
	left := []map[string]any{
		{"id": 1, "name": "alice", "active": 1},
		{"id": 2, "name": "bob", "active": 1},
	}
	right := []map[string]any{
		{"user_id": 1, "total": 10.5},
		{"user_id": 2, "total": 20.0},
	}

	got, err := HashJoin(left, right, sampleJoinSpec(), sampleSelectCols(), nil)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}

	want := []map[string]any{
		{"id": 1, "name": "alice", "total": 10.5},
		{"id": 2, "name": "bob", "total": 20.0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HashJoin() = %+v, want %+v", got, want)
	}
}

func TestHashJoinNoMatch(t *testing.T) {
	left := []map[string]any{{"id": 1, "name": "alice"}}
	right := []map[string]any{{"user_id": 99, "total": 10.5}}

	got, err := HashJoin(left, right, sampleJoinSpec(), sampleSelectCols(), nil)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("HashJoin() = %+v, want empty", got)
	}
}

func TestHashJoinOneToMany(t *testing.T) {
	left := []map[string]any{{"id": 1, "name": "alice"}}
	right := []map[string]any{
		{"user_id": 1, "total": 10.0},
		{"user_id": 1, "total": 15.0},
	}

	got, err := HashJoin(left, right, sampleJoinSpec(), sampleSelectCols(), nil)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("HashJoin() len = %d, want 2", len(got))
	}
}

func TestHashJoinEmptyInput(t *testing.T) {
	spec := sampleJoinSpec()
	cols := sampleSelectCols()

	got, err := HashJoin(nil, nil, spec, cols, nil)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("HashJoin() = %+v, want empty", got)
	}

	got, err = HashJoin([]map[string]any{{"id": 1}}, nil, spec, cols, nil)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("HashJoin() = %+v, want empty", got)
	}
}

func TestHashJoinLimit(t *testing.T) {
	left := []map[string]any{
		{"id": 1, "name": "alice"},
		{"id": 2, "name": "bob"},
	}
	right := []map[string]any{
		{"user_id": 1, "total": 10.0},
		{"user_id": 2, "total": 20.0},
	}

	limit := 1
	got, err := HashJoin(left, right, sampleJoinSpec(), sampleSelectCols(), &limit)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("HashJoin() len = %d, want 1", len(got))
	}
}

func TestHashJoinBuildsFromSmallerLeg(t *testing.T) {
	left := []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
		{"id": 3, "name": "c"},
	}
	right := []map[string]any{
		{"user_id": 2, "total": 20.0},
	}

	got, err := HashJoin(left, right, sampleJoinSpec(), sampleSelectCols(), nil)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}
	if len(got) != 1 || got[0]["name"] != "b" {
		t.Fatalf("HashJoin() = %+v, want one bob row", got)
	}
}

func TestHashJoinDuplicateSelectColumnNames(t *testing.T) {
	left := []map[string]any{{"id": 1, "name": "alice"}}
	right := []map[string]any{{"user_id": 1, "id": 99, "total": 10.0}}

	join := sampleJoinSpec()
	selectCols := []ColumnRef{
		{Table: "u", Column: "id"},
		{Table: "o", Column: "id"},
	}

	got, err := HashJoin(left, right, join, selectCols, nil)
	if err != nil {
		t.Fatalf("HashJoin() error = %v", err)
	}

	want := []map[string]any{{"u.id": 1, "o.id": 99}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HashJoin() = %+v, want %+v", got, want)
	}
}
