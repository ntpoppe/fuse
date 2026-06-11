package driver

// SelectLeg is the driver-layer input for RenderSelect.
type SelectLeg struct {
	Table   string
	Columns []string
	Where   []SelectPredicate
}

// SelectPredicate is a parameterized WHERE condition on one column.
type SelectPredicate struct {
	Column string
	Op     string
	Value  any
}
