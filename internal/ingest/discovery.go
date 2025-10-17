package ingest

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logvoyant/internal/storage"
)

// DiscoverAndStart finds log files and starts tailing them
func DiscoverAndStart(store storage.Storage, hub LogBroadcaster) error {
	var logPaths []string

	// Common log locations to check
	searchPaths := []string{
		"/host/var/log",
		"/var/log",
		"/logs",
	}

	// Common log patterns
	patterns := []string{
		"*.log",
		"syslog*",
		"messages*",
		"auth.log*",
	}

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		for _, pattern := range patterns {
			matches, err := filepath.Glob(filepath.Join(searchPath, pattern))
			if err != nil {
				continue
			}

			for _, match := range matches {
				// Skip rotated/compressed logs
				if strings.HasSuffix(match, ".gz") || strings.HasSuffix(match, ".1") {
					continue
				}

				// Check if file is readable
				if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
					logPaths = append(logPaths, match)
				}
			}
		}
	}

	if len(logPaths) == 0 {
		log.Println("⚠️  No log files discovered. Mount logs with -v /var/log:/host/var/log:ro")
		return nil
	}

	log.Printf("✓ Discovered %d log files", len(logPaths))
	
	// Start tailing each file
	for _, path := range logPaths {
		streamID := "file:" + path
		
		// Create stream entry
		stream := &storage.Stream{
			ID:     streamID,
			Name:   filepath.Base(path),
			Source: "file",
			Active: true,
		}
		store.UpdateStream(stream)
		
		// Initialize context
		ctx, _ := store.GetContext(streamID)
		if ctx.StreamID == "" {
			ctx.StreamID = streamID
			ctx.FirstSeen = time.Now()
			ctx.Analyses = []storage.AnalysisSummary{}
			ctx.Patterns = storage.StreamPatterns{CommonErrors: []string{}}
			store.UpdateContext(streamID, ctx)
		}

		// Start tailer in background
		tailer := NewFileTailer(path, streamID, store, hub)
		go func(t *FileTailer, p string) {
			log.Printf("📂 Tailing: %s", p)
			if err := t.Start(); err != nil {
				log.Printf("❌ Tailer error for %s: %v", p, err)
			}
		}(tailer, path)
	}

	return nil
}