package storage

import "time"

// LogLine represents a single log entry
type LogLine struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`     // ERROR, WARN, INFO, DEBUG
	Message   string            `json:"message"`
	Raw       string            `json:"raw"`       // Original log line
	Labels    map[string]string `json:"labels"`    // pod, namespace, etc.
	StreamID  string            `json:"stream_id"`
}

// StreamContext holds historical knowledge about a stream
type StreamContext struct {
	StreamID   string            `json:"stream_id"`
	FirstSeen  time.Time         `json:"first_seen"`
	LastSeen   time.Time         `json:"last_seen"`
	Analyses   []AnalysisSummary `json:"analyses"`
	Patterns   StreamPatterns    `json:"patterns"`
	TotalLogs  int64             `json:"total_logs"`
	ErrorCount int64             `json:"error_count"`
}

// AnalysisSummary is a condensed analysis stored as context
type AnalysisSummary struct {
	Timestamp      time.Time `json:"timestamp"`
	Summary        string    `json:"summary"`
	RootCause      string    `json:"root_cause"`
	Severity       string    `json:"severity"`        // P0, P1, P2, P3
	Resolved       bool      `json:"resolved"`
	ResolutionNote string    `json:"resolution_note,omitempty"`
}

// StreamPatterns tracks recurring issues
type StreamPatterns struct {
	CommonErrors []string `json:"common_errors"`
	ErrorRate    float64  `json:"error_rate"`
}

// Analysis is the full AI-generated analysis
type Analysis struct {
	Timestamp time.Time `json:"timestamp"`
	StreamID  string    `json:"stream_id"`
	Summary   string    `json:"summary"`
	RootCause string    `json:"root_cause"`
	Severity  string    `json:"severity"`
	Fixes     []string  `json:"fixes,omitempty"`
	Context   string    `json:"context,omitempty"` // Historical context used
}

// Stream represents an active log stream
type Stream struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Source      string    `json:"source"`      // kubectl, docker, file
	Active      bool      `json:"active"`
	LogsPerMin  int       `json:"logs_per_min"`
	ErrorRate   float64   `json:"error_rate"`
	LastSeen    time.Time `json:"last_seen"`
	ContextSummary string `json:"context_summary"`
}