package span

const AttributeReqBody = "request.body"

const (
	AttributeRespHttpCode = "http.status_code"
	AttributeRespErrMsg   = "error.message"
)

const (
	AttributeDBStatement    = "db.statement"
	AttributeDBTable        = "db.table"
	AttributeDbRowsAffected = "db.rows_affected"
	AttributeCommandName    = "command.name"
	AttributeCommandParams  = "command.params"
	AttributeReply          = "reply"
	AttributeFailure        = "failure"
)

const (
	AttributeOpenSearchReqBody  = "opensearch.request.body"
	AttributeOpenSearchRespBody = "opensearch.response.body"
	AttributeOpenSearchDuration = "opensearch.duration"
)
