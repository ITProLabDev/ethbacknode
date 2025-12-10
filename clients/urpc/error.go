package urpc

type WarpedError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err *WarpedError) Error() string {
	return err.Message
}
