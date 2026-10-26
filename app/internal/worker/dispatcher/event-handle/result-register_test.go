package event_handle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventRegister_Flow(t *testing.T) {
	register := &EventRegister{
		results: []ScanResult{},
		Done:    make(chan struct{}),
	}

	resultChan := make(chan ResultEvent, 2)
	resultChan <- ResultEvent{ResultType: Success}
	resultChan <- ResultEvent{ResultType: Message}
	close(resultChan)
	
	register.StartListening(resultChan)

	<-register.Done

	results := register.GetResults()
	assert.Len(t, results, 2)
	assert.Equal(t, Success, results[0].ResultType)
	assert.Equal(t, Message, results[1].ResultType)
}
