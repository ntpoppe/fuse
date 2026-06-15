package api

const (
	PathHealth         = "/health"
	PathConnections    = "/api/connections"
	PathConnectionByID = "/api/connections/{id}"
	PathQuery = "/api/query"

	ContentTypeJSON = "application/json"

	fieldStatus    = "status"
	healthStatusOK = "ok"

	errInvalidJSON           = "invalid JSON payload"
	errMissingSQL            = "missing required field: sql"
	errMissingConnFields   = "missing required fields: id, driver, and host"
	errMissingConnectionID        = "missing connection id"
	errConnectionChangesDisabled  = "connection changes are not allowed"
)
