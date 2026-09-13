package types

type ScanError struct {
	Message string
	Code    int
	Err     error
}

func (e *ScanError) Error() string {
	return e.Message
}
