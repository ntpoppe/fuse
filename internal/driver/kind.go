package driver

type Kind string

const (
	KindSQLite  Kind = DriverSQLite
	KindMySQL   Kind = DriverMySQL
	KindUnknown Kind = "unknown"
)

func KindFromDriver(driverName string) Kind {
	switch driverName {
	case DriverSQLite:
		return KindSQLite
	case DriverMySQL:
		return KindMySQL
	default:
		return KindUnknown
	}
}
