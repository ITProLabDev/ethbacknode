package urpc

// JSON-RPC version constants.
const (
	JSON_RPC_VERSION_1_0 = "1.0" // JSON-RPC 1.0 (legacy)
	JSON_RPC_VERSION_2_0 = "2.0" // JSON-RPC 2.0 (standard)
)

// Standard JSON-RPC 2.0 error codes and messages.
const (
	// Parse error: Invalid JSON was received.
	ERROR_CODE_PARSE_ERROR    = -32700
	ERROR_MESSAGE_PARSE_ERROR = "Parse error"

	// Invalid Request: The JSON sent is not a valid Request object.
	ERROR_CODE_INVALID_REQUEST    = -32600
	ERROR_MESSAGE_INVALID_REQUEST = "invalid request"

	// Method not found: The method does not exist.
	ERROR_CODE_METHOD_NOT_FOUND    = -32601
	ERROR_MESSAGE_METHOD_NOT_FOUND = "method not found"

	// Server error: Reserved for implementation-defined server errors.
	ERROR_CODE_SERVER_ERROR    = -32000
	ERROR_MESSAGE_SERVER_ERROR = "server error"
)
