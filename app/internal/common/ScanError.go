package common

type ScanError struct {
	Message string
	Code    int
}

func (e *ScanError) Error() string {
	return e.Message
}
