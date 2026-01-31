package errorx

type CustomError struct {
	HttpCode int
	Code     int
	Message  string
	Cause    error
}

func New(httpCode, code int, message string) error {
	return &CustomError{HttpCode: httpCode, Code: code, Message: message}
}

func (e *CustomError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e *CustomError) Unwrap() error {
	return e.Cause
}

func (e *CustomError) Is(target error) bool {
	return e == target
}
