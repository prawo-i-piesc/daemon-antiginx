package event_handle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseToScanResult(t *testing.T) {
	input := ResultEvent{
		Result:      TestResult{Name: "test-name", ThreatLevel: High},
		ProcessInfo: RequestInfo{Code: 200, Message: "ok"},
		ResultType:  Success,
	}

	output := parseToScanResult(input)

	assert.Equal(t, "test-name", output.TestResult.Name)
	assert.Equal(t, High, output.TestResult.ThreatLevel)
	assert.Equal(t, 200, output.ScanInfo.Code)
	assert.Equal(t, "ok", output.ScanInfo.Message)
	assert.Equal(t, Success, output.ResultType)
}
