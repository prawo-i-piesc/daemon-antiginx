package event_handle

func parseToScanResult(result ResultEvent) ScanResult {
	return ScanResult{
		TestResult: result.Result,
		ScanInfo:   result.ProcessInfo,
		ResultType: result.ResultType,
	}
}
