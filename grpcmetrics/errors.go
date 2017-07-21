package grpcmetrics

// Ignorable returns true if the error should be ignored for the purposes of
// error metrics.
func Ignorable(err error) bool {
	igerr, ok := err.(ignorable)
	return ok && igerr.Ignore()
}

type ignorable struct {
	error
}

func (i ignorable) Ignore() bool  { return true }
func (i ignorable) Cause() error  { return i.error }
func (i ignorable) Error() string { return i.error.Error() }

// Ignore wraps errors in a marker interface indicating that the error should be
// ignored.
func Ignore(err error) error {
	return ignorable{err}
}
