package ingest

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"logvoyant/internal/storage"
)

// FileTailer tails log files and parses them
type FileTailer struct {
	path     string
	streamID string
	storage  storage.Storage
	hub      LogBroadcaster
}

type LogBroadcaster interface {
	BroadcastLog(streamID string, log storage.LogLine)
}

func NewFileTailer(path, streamID string, store storage.Storage, hub LogBroadcaster) *FileTailer {
	return &FileTailer{
		path:     path,
		streamID: streamID,
		storage:  store,
		hub:      hub,
	}
}

// Start begins tailing the file
func (f *FileTailer) Start() error {
	file, err := os.Open(f.path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read existing logs first (last 100 lines), then tail new ones
	info, _ := file.Stat()
	log.Printf("File %s size: %d bytes", f.path, info.Size())
	
	if info.Size() > 0 {
		// Seek to beginning to read existing logs
		file.Seek(0, os.SEEK_SET)
		scanner := bufio.NewScanner(file)
		
		// Read up to 100 lines on startup
		lines := []string{}
		lineCount := 0
		for scanner.Scan() {
			lineCount++
			lines = append(lines, scanner.Text())
			if len(lines) > 10000 {
				lines = lines[1:] // Keep sliding window
			}
		}
		
		log.Printf("Read %d lines from %s", lineCount, f.path)
		
		// Process last 100 lines
		start := len(lines) - 100
		if start < 0 {
			start = 0
		}
		
		logsToStore := []storage.LogLine{}
		for i := start; i < len(lines); i++ {
			if lines[i] == "" {
				continue
			}
			logLine := f.parseLine(lines[i])
			logsToStore = append(logsToStore, logLine)
		}
		
		if len(logsToStore) > 0 {
			log.Printf("Storing %d logs for %s", len(logsToStore), f.streamID)
			if err := f.storage.StoreLogs(f.streamID, logsToStore); err != nil {
				log.Printf("Failed to store logs: %v", err)
			}
			
			// Broadcast initial logs
			if f.hub != nil {
				for _, logLine := range logsToStore {
					f.hub.BroadcastLog(f.streamID, logLine)
				}
			}
		}
	}

	// Now seek to end and tail new logs
	file.Seek(0, os.SEEK_END)
	
	log.Printf("Started tailing %s (stream: %s)", f.path, f.streamID)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		logLine := f.parseLine(line)
		
		// Store in database
		if err := f.storage.StoreLogs(f.streamID, []storage.LogLine{logLine}); err != nil {
			log.Printf("Failed to store log: %v", err)
		}

		// Broadcast to WebSocket clients
		if f.hub != nil {
			f.hub.BroadcastLog(f.streamID, logLine)
		}
	}

	return scanner.Err()
}

// parseLine attempts to extract structured data from log line
func (f *FileTailer) parseLine(line string) storage.LogLine {
	logLine := storage.LogLine{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   line,
		Raw:       line,
		StreamID:  f.streamID,
		Labels:    make(map[string]string),
	}

	// Try to extract log level
	levelPattern := regexp.MustCompile(`\[(ERROR|WARN|INFO|DEBUG|FATAL)\]|ERROR|WARN|INFO|DEBUG|FATAL`)
	if match := levelPattern.FindString(line); match != "" {
		logLine.Level = strings.Trim(strings.ToUpper(match), "[]")
	}

	// Try to extract timestamp (ISO8601 or common formats)
	timestampPattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2})`)
	if match := timestampPattern.FindString(line); match != "" {
		if ts, err := time.Parse("2006-01-02T15:04:05", match); err == nil {
			logLine.Timestamp = ts
		} else if ts, err := time.Parse("2006-01-02 15:04:05", match); err == nil {
			logLine.Timestamp = ts
		}
	}

	// Extract message (remove timestamp and level)
	msg := line
	msg = levelPattern.ReplaceAllString(msg, "")
	msg = timestampPattern.ReplaceAllString(msg, "")
	msg = strings.TrimSpace(msg)
	if msg != "" {
		logLine.Message = msg
	}

	return logLine
}

// TailMultipleFiles starts multiple tailers
func TailMultipleFiles(paths []string, store storage.Storage, hub LogBroadcaster) error {
	for _, path := range paths {
		streamID := fmt.Sprintf("file:%s", path)
		tailer := NewFileTailer(path, streamID, store, hub)
		
		go func(t *FileTailer) {
			if err := t.Start(); err != nil {
				log.Printf("Tailer error for %s: %v", t.path, err)
			}
		}(tailer)
	}
	
	return nil
}