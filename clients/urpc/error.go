package urpc

// WarpedError wraps a JSON-RPC error as a Go error.
// Implements the error interface.
type WarpedError struct {
	Code    int    `json:"code"`    // JSON-RPC error code
	Message string `json:"message"` // Error message
}

// Error returns the error message, implementing the error interface.
func (err *WarpedError) Error() string {
	return err.Message
}
