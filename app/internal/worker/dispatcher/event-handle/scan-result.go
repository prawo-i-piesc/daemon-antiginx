package event_handle

type ScanContext struct {
	Target  string     `json:"target"`
	TestId  string     `json:"testId"`
	EndFlag bool       `json:"endFlag"`
	Result  ScanResult `json:"result"`
}

type ScanResult struct {
	TestResult TestResult  `json:"testResult"`
	ScanInfo   RequestInfo `json:"scanInfo"`
	ResultType ResultType  `json:"resultType"`
}
