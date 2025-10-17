package storage

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	logsBucketPrefix = []byte("logs:")
	contextBucket    = []byte("context")
	analysisBucket   = []byte("analysis")
	streamsBucket    = []byte("streams")
)

type BoltStorage struct {
	db *bolt.DB
}

func NewBoltStorage(path string) (*BoltStorage, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %w", err)
	}

	// Initialize buckets
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{contextBucket, analysisBucket, streamsBucket}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &BoltStorage{db: db}, nil
}

func (s *BoltStorage) Close() error {
	return s.db.Close()
}

// StoreLogs saves logs to stream-specific bucket with ring buffer (keep last 10k)
func (s *BoltStorage) StoreLogs(streamID string, logs []LogLine) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucketName := append(logsBucketPrefix, []byte(streamID)...)
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		errorCount := 0
		for _, log := range logs {
			key := []byte(log.Timestamp.Format(time.RFC3339Nano))
			data, err := json.Marshal(log)
			if err != nil {
				return err
			}
			if err := bucket.Put(key, data); err != nil {
				return err
			}
			
			if log.Level == "ERROR" || log.Level == "FATAL" {
				errorCount++
			}
		}

		// Ring buffer: delete old entries if > 10k
		stats := bucket.Stats()
		if stats.KeyN > 10000 {
			c := bucket.Cursor()
			toDelete := stats.KeyN - 10000
			count := 0
			for k, _ := c.First(); k != nil && count < toDelete; k, _ = c.Next() {
				bucket.Delete(k)
				count++
			}
		}

		// Update stream metadata
		streamsBucket := tx.Bucket(streamsBucket)
		if streamsBucket != nil {
			streamData := streamsBucket.Get([]byte(streamID))
			var stream Stream
			if streamData != nil {
				json.Unmarshal(streamData, &stream)
			} else {
				stream = Stream{
					ID:     streamID,
					Active: true,
				}
			}
			
			// Update stats
			stream.LastSeen = time.Now()
			totalLogs := int64(stats.KeyN)
			
			// Get context for error count
			ctxBucket := tx.Bucket(contextBucket)
			if ctxBucket != nil {
				ctxData := ctxBucket.Get([]byte(streamID))
				var ctx StreamContext
				if ctxData != nil {
					json.Unmarshal(ctxData, &ctx)
					ctx.TotalLogs = totalLogs
					ctx.ErrorCount += int64(errorCount)
					ctx.LastSeen = time.Now()
					
					if totalLogs > 0 {
						ctx.Patterns.ErrorRate = float64(ctx.ErrorCount) / float64(totalLogs)
					}
					
					// Update context
					updatedCtx, _ := json.Marshal(ctx)
					ctxBucket.Put([]byte(streamID), updatedCtx)
				}
			}
			
			// Save updated stream
			updatedStream, _ := json.Marshal(stream)
			streamsBucket.Put([]byte(streamID), updatedStream)
		}

		return nil
	})
}

func (s *BoltStorage) GetLogs(streamID string, opts GetLogsOptions) ([]LogLine, error) {
	var logs []LogLine

	err := s.db.View(func(tx *bolt.Tx) error {
		bucketName := append(logsBucketPrefix, []byte(streamID)...)
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return nil // No logs yet
		}

		c := bucket.Cursor()
		count := 0

		// Start from most recent
		for k, v := c.Last(); k != nil && (opts.Limit == 0 || count < opts.Limit); k, v = c.Prev() {
			var log LogLine
			if err := json.Unmarshal(v, &log); err != nil {
				continue
			}

			// Apply filters
			if !opts.Since.IsZero() && log.Timestamp.Before(opts.Since) {
				break
			}
			if len(opts.Levels) > 0 && !contains(opts.Levels, log.Level) {
				continue
			}

			logs = append([]LogLine{log}, logs...) // Prepend to maintain order
			count++
		}

		return nil
	})

	return logs, err
}

func (s *BoltStorage) ListStreams() ([]Stream, error) {
	var streams []Stream

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(streamsBucket)
		if bucket == nil {
			return nil
		}

		ctxBucket := tx.Bucket(contextBucket)

		return bucket.ForEach(func(k, v []byte) error {
			var stream Stream
			if err := json.Unmarshal(v, &stream); err != nil {
				return err
			}
			
			// Get log count from logs bucket
			logsBucketName := append(logsBucketPrefix, k...)
			logsBucket := tx.Bucket(logsBucketName)
			if logsBucket != nil {
				stats := logsBucket.Stats()
				stream.LogsPerMin = stats.KeyN // Total logs for now
			}
			
			// Enrich with context data
			if ctxBucket != nil {
				ctxData := ctxBucket.Get(k)
				if ctxData != nil {
					var ctx StreamContext
					if err := json.Unmarshal(ctxData, &ctx); err == nil {
						stream.ErrorRate = ctx.Patterns.ErrorRate
						if len(ctx.Analyses) > 0 {
							latest := ctx.Analyses[len(ctx.Analyses)-1]
							stream.ContextSummary = fmt.Sprintf("Last: %s (%s)", latest.Summary, latest.Severity)
						}
					}
				}
			}
			
			streams = append(streams, stream)
			return nil
		})
	})

	return streams, err
}

func (s *BoltStorage) GetStream(streamID string) (*Stream, error) {
	var stream Stream

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(streamsBucket)
		if bucket == nil {
			return fmt.Errorf("stream not found")
		}

		data := bucket.Get([]byte(streamID))
		if data == nil {
			return fmt.Errorf("stream not found")
		}

		return json.Unmarshal(data, &stream)
	})

	if err != nil {
		return nil, err
	}
	return &stream, nil
}

func (s *BoltStorage) UpdateStream(stream *Stream) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(streamsBucket)
		data, err := json.Marshal(stream)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(stream.ID), data)
	})
}

func (s *BoltStorage) GetContext(streamID string) (*StreamContext, error) {
	var ctx StreamContext

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(contextBucket)
		data := bucket.Get([]byte(streamID))
		if data == nil {
			// Return empty context if not found
			ctx = StreamContext{
				StreamID:  streamID,
				FirstSeen: time.Now(),
				Analyses:  []AnalysisSummary{},
				Patterns:  StreamPatterns{CommonErrors: []string{}},
			}
			return nil
		}
		return json.Unmarshal(data, &ctx)
	})

	return &ctx, err
}

func (s *BoltStorage) UpdateContext(streamID string, ctx *StreamContext) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(contextBucket)
		data, err := json.Marshal(ctx)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(streamID), data)
	})
}

func (s *BoltStorage) StoreAnalysis(analysis *Analysis) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(analysisBucket)
		key := fmt.Sprintf("%s:%s", analysis.StreamID, analysis.Timestamp.Format(time.RFC3339))
		data, err := json.Marshal(analysis)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), data)
	})
}

func (s *BoltStorage) GetAnalysisHistory(streamID string, limit int) ([]Analysis, error) {
	var analyses []Analysis

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(analysisBucket)
		c := bucket.Cursor()
		prefix := []byte(streamID + ":")
		count := 0

		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			if limit > 0 && count >= limit {
				break
			}

			var analysis Analysis
			if err := json.Unmarshal(v, &analysis); err != nil {
				continue
			}
			analyses = append(analyses, analysis)
			count++
		}

		return nil
	})

	return analyses, err
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}