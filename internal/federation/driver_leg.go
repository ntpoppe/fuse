package federation

import "github.com/ntpoppe/fuse/internal/driver"

// SelectLegForDriver converts a federation QueryLeg into driver.SelectLeg for RenderSelect.
func SelectLegForDriver(leg QueryLeg) driver.SelectLeg {
	where := make([]driver.SelectPredicate, len(leg.Where))
	for i, pred := range leg.Where {
		where[i] = driver.SelectPredicate{
			Column: pred.Column.Column,
			Op:     pred.Op,
			Value:  pred.Value,
		}
	}

	return driver.SelectLeg{
		Table:   leg.Table.Table,
		Columns: leg.Columns,
		Where:   where,
	}
}
