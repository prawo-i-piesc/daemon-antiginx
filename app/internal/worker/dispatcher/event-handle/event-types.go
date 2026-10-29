package event_handle

import (
	"encoding/json"
	"log/slog"
)

type EngineEventType int

const (
	Result EngineEventType = iota
	Log
	Metrics
	Error
)

type EngineEventWrapper struct {
	EventType EngineEventType
	Event     json.RawMessage
}

type ErrorEvent struct {
	Message string `json:"Message"`
	Error   error  `json:"Error"`
	Code    int    `json:"Code"`
}

type ResultEvent struct {
	Target      string      `json:"target"`
	TestId      string      `json:"testId"`
	Result      TestResult  `json:"result"`
	EndFlag     bool        `json:"endFlag"`
	ResultType  ResultType  `json:"resultType"`
	ProcessInfo RequestInfo `json:"message"`
}

type LogEvent struct {
	Message string     `json:"Message"`
	TestId  string     `json:"TestId"`
	Level   slog.Level `json:"Level"`
	Error   string     `json:"Error"`
	Event   string     `json:"Event"`
}

// TODO research o metrykach prometeusza
type MetricsEvent struct {
	Name   string
	Type   MetricType
	Labels []string
	Value  float64
}

type ResultType int

const (
	Message ResultType = iota
	Success
)

type RequestInfo struct {
	Message     string   `json:"Message"`
	Code        int      `json:"Code"`
	Protections []string `json:"Protections,omitempty"`
}
type TestResult struct {
	Name        string      `json:"Name"`        // Test name for identification
	Certainty   int         `json:"Certainty"`   // Confidence percentage (0-100)
	ThreatLevel ThreatLevel `json:"ThreatLevel"` // Security threat classification
	Metadata    any         `json:"Metadata"`    // Test-specific detailed data
	Description string      `json:"Description"` // Human-readable findings explanation
}

type ThreatLevel string

const (
	None     ThreatLevel = "None"     // 0 - No security issues
	Info     ThreatLevel = "Info"     // 1 - Informational findings
	Low      ThreatLevel = "Low"      // 2 - Low severity issues
	Medium   ThreatLevel = "Medium"   // 3 - Medium severity issues
	High     ThreatLevel = "High"     // 4 - High severity vulnerabilities
	Critical ThreatLevel = "Critical" // 5 - Critical vulnerabilities
)
