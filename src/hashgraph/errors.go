package hashgraph

// SelfParentError is used to differentiate errors that are normal when the
// hashgraph is being used corrently by multiple go-routines, from errors that
// should not be occuring event in a concurrent context.
type SelfParentError struct {
	msg    string
	normal bool
}

// NewSelfParentError creates a new SelfParentError
func NewSelfParentError(msg string, normal bool) SelfParentError {
	return SelfParentError{
		msg:    msg,
		normal: normal,
	}
}

// Error implements the Error interface
func (e SelfParentError) Error() string {
	return e.msg
}

// IsNormalSelfParentError checks that an error is of type SelfParentError and
// that it is normal. Because of the asynchronous nature of Babble, some
// routines will simultaneously try to append the same event, which is obviously
// not allowed and will throw a SelfParent error. But such errors are not real
// errors that we want to report in the logs.
func IsNormalSelfParentError(err error) bool {
	spErr, ok := err.(SelfParentError)
	return ok && spErr.normal
}
