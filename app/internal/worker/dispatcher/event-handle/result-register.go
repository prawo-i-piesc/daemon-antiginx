package event_handle

type EventRegister struct {
	results []ScanResult
	Done    chan struct{}
}

func (r *EventRegister) StartListening(resultChan chan ResultEvent) {
	for result := range resultChan {
		r.results = append(r.results, parseToScanResult(result))
	}

	close(r.Done)
}

func (r *EventRegister) GetResults() []ScanResult {
	return r.results
}
