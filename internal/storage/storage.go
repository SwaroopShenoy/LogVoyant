package storage

import "time"

// Storage interface for log and context management
type Storage interface {
	// Logs
	StoreLogs(streamID string, logs []LogLine) error
	GetLogs(streamID string, opts GetLogsOptions) ([]LogLine, error)
	
	// Streams
	ListStreams() ([]Stream, error)
	GetStream(streamID string) (*Stream, error)
	UpdateStream(stream *Stream) error
	
	// Context
	GetContext(streamID string) (*StreamContext, error)
	UpdateContext(streamID string, ctx *StreamContext) error
	
	// Analysis
	StoreAnalysis(analysis *Analysis) error
	GetAnalysisHistory(streamID string, limit int) ([]Analysis, error)
	
	// Lifecycle
	Close() error
}

// GetLogsOptions for filtering logs
type GetLogsOptions struct {
	Limit  int
	Since  time.Time
	Levels []string // ERROR, WARN, INFO, DEBUG
}