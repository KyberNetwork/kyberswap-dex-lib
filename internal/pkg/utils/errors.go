package utils

type WrappedError struct {
	detail error
	code   int
}

func NewWrappedError(err error, code int) WrappedError {
	return WrappedError{detail: err, code: code}
}

func (err WrappedError) Error() string {
	return err.detail.Error()
}

func (err WrappedError) Code() int {
	return err.code
}
