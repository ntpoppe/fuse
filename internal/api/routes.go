package api

const (
	PathHealth         = "/health"
	PathConnections    = "/api/connections"
	PathConnectionByID = "/api/connections/{id}"
	PathQuery          = "/api/query"

	ContentTypeJSON = "application/json"

	fieldStatus    = "status"
	healthStatusOK = "ok"

	errInvalidJSON         = "invalid JSON payload"
	errMissingQueryFields  = "missing required fields: id and sql"
	errMissingConnFields   = "missing required fields: id, driver, and host"
	errMissingConnectionID = "missing connection id"
)
