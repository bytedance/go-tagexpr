package binding

// Error validate error
type Error struct {
	ErrType, FailField, Msg string
}

// Error implements error interface.
func (e *Error) Error() string {
	if e.Msg != "" {
		return e.ErrType + ": expr_path=" + e.FailField + ", cause=" + e.Msg
	}
	return e.ErrType + ": expr_path=" + e.FailField + ", cause=invalid"
}

func newDefaultErrorFactory(errType string) func(string, string) error {
	return func(failField, msg string) error {
		return &Error{
			ErrType:   errType,
			FailField: failField,
			Msg:       msg,
		}
	}
}
