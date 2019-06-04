package binding

// Error validate error
type Error struct {
	ErrType, FailField, Msg string
}

// Error implements error interface.
func (e *Error) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return e.ErrType + ": " + e.FailField
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
