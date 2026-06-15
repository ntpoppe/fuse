package api

const (
	PathHealth         = "/health"
	PathConnections    = "/api/connections"
	PathConnectionByID = "/api/connections/{id}"
	PathQuery           = "/api/query"
	PathFederatedQuery  = "/api/federated-query"

	ContentTypeJSON = "application/json"

	fieldStatus    = "status"
	healthStatusOK = "ok"

	errInvalidJSON         = "invalid JSON payload"
	errMissingQueryFields     = "missing required fields: id and sql"
	errMissingFederatedSQL    = "missing required field: sql"
	errMissingConnFields   = "missing required fields: id, driver, and host"
	errMissingConnectionID   = "missing connection id"
	errDemoModeConnections   = "demo mode: connections are read-only"
)
