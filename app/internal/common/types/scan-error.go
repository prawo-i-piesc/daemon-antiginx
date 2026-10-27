package types

type ScanError struct {
	Message   string
	Code      int
	Err       error
	TaskId    string
	WorkerId  int
	Component string
}

func (e *ScanError) Error() string {
	return e.Message
}
