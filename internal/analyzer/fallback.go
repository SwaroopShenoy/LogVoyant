package analyzer

import (
	"fmt"
	"strings"

	"logvoyant/internal/storage"
)

// FallbackAnalyzer provides offline pattern-based analysis
type FallbackAnalyzer struct {
	patterns []ErrorPattern
}

type ErrorPattern struct {
	Keywords   []string
	RootCause  string
	Fixes      []string
	Severity   string
}

func NewFallbackAnalyzer() *FallbackAnalyzer {
	return &FallbackAnalyzer{
		patterns: []ErrorPattern{
			{
				Keywords:  []string{"connection", "timeout", "refused"},
				RootCause: "Network connectivity issue - service unreachable or connection timing out",
				Fixes: []string{
					"Check network connectivity and firewall rules",
					"Verify target service is running and accessible",
					"Review connection timeout settings",
				},
				Severity: "P1",
			},
			{
				Keywords:  []string{"out of memory", "oom", "memory limit"},
				RootCause: "Application exceeding memory limits",
				Fixes: []string{
					"Increase memory allocation for the container/pod",
					"Review memory leaks in application code",
					"Enable memory profiling to identify hot spots",
				},
				Severity: "P0",
			},
			{
				Keywords:  []string{"database", "db", "sql", "query"},
				RootCause: "Database-related error - connection, query, or schema issue",
				Fixes: []string{
					"Verify database connection parameters",
					"Check database server status and load",
					"Review query performance and indexes",
				},
				Severity: "P1",
			},
			{
				Keywords:  []string{"authentication", "auth", "unauthorized", "403", "401"},
				RootCause: "Authentication or authorization failure",
				Fixes: []string{
					"Verify API keys and credentials are valid",
					"Check token expiration and refresh mechanisms",
					"Review RBAC policies and permissions",
				},
				Severity: "P2",
			},
			{
				Keywords:  []string{"disk", "storage", "volume", "no space"},
				RootCause: "Storage capacity issue - disk full or volume mount problem",
				Fixes: []string{
					"Check available disk space on nodes",
					"Review log rotation and cleanup policies",
					"Increase persistent volume size if needed",
				},
				Severity: "P0",
			},
			{
				Keywords:  []string{"crash", "panic", "fatal", "segfault"},
				RootCause: "Critical application crash or fatal error",
				Fixes: []string{
					"Review application logs for stack traces",
					"Check recent code deployments for regressions",
					"Enable core dumps for debugging",
				},
				Severity: "P0",
			},
			{
				Keywords:  []string{"ssl", "tls", "certificate", "x509"},
				RootCause: "SSL/TLS certificate validation failure",
				Fixes: []string{
					"Verify certificate expiration dates",
					"Check certificate chain and CA trust",
					"Review certificate SANs and hostname matching",
				},
				Severity: "P1",
			},
			{
				Keywords:  []string{"rate limit", "throttle", "429"},
				RootCause: "API rate limiting or throttling in effect",
				Fixes: []string{
					"Implement exponential backoff in client code",
					"Review and increase rate limit quotas",
					"Optimize request patterns to reduce frequency",
				},
				Severity: "P2",
			},
			{
				Keywords:  []string{"dns", "resolve", "hostname"},
				RootCause: "DNS resolution failure",
				Fixes: []string{
					"Check DNS server configuration",
					"Verify service names and namespace in K8s",
					"Review /etc/resolv.conf settings",
				},
				Severity: "P1",
			},
			{
				Keywords:  []string{"port", "bind", "address already in use"},
				RootCause: "Port already in use - potential duplicate service",
				Fixes: []string{
					"Check for duplicate deployments on same port",
					"Review port allocation and conflicts",
					"Verify service discovery configuration",
				},
				Severity: "P2",
			},
		},
	}
}

func (f *FallbackAnalyzer) Analyze(logs []storage.LogLine, ctx *storage.StreamContext) *storage.Analysis {
	// Count errors by level
	errorCount := 0
	warnCount := 0
	fatalCount := 0
	allMessages := make([]string, 0)
	errorMessages := make([]string, 0)
	
	for _, log := range logs {
		msgLower := strings.ToLower(log.Message)
		allMessages = append(allMessages, msgLower)
		
		switch log.Level {
		case "ERROR":
			errorCount++
			errorMessages = append(errorMessages, msgLower)
		case "FATAL":
			fatalCount++
			errorMessages = append(errorMessages, msgLower)
		case "WARN":
			warnCount++
		}
	}

	// Find matching patterns
	var matchedPattern *ErrorPattern
	maxMatches := 0

	for i := range f.patterns {
		pattern := &f.patterns[i]
		matches := 0
		for _, keyword := range pattern.Keywords {
			keywordLower := strings.ToLower(keyword)
			for _, msg := range errorMessages {
				if strings.Contains(msg, keywordLower) {
					matches++
					break
				}
			}
		}
		if matches > maxMatches {
			maxMatches = matches
			matchedPattern = pattern
		}
	}

	// Build analysis
	analysis := &storage.Analysis{
		Severity: "P3",
	}

	// Determine severity based on error types
	if fatalCount > 0 {
		analysis.Severity = "P0"
	} else if errorCount > 50 {
		analysis.Severity = "P0"
	} else if errorCount > 10 {
		analysis.Severity = "P1"
	} else if errorCount > 0 {
		analysis.Severity = "P2"
	} else if warnCount > 20 {
		analysis.Severity = "P2"
	}

	if matchedPattern != nil {
		// Use matched pattern
		analysis.Summary = fmt.Sprintf("Detected %s errors (%d errors, %d warnings)", 
			matchedPattern.Keywords[0], errorCount, warnCount)
		analysis.RootCause = matchedPattern.RootCause
		analysis.Fixes = matchedPattern.Fixes
		
		// Override severity if pattern has a higher priority
		if matchedPattern.Severity == "P0" && (analysis.Severity == "P1" || analysis.Severity == "P2" || analysis.Severity == "P3") {
			analysis.Severity = "P0"
		} else if matchedPattern.Severity == "P1" && (analysis.Severity == "P2" || analysis.Severity == "P3") {
			analysis.Severity = "P1"
		}
	} else {
		// Generic analysis - try to extract common error phrases
		errorPhrases := f.extractCommonPhrases(errorMessages)
		
		if len(errorPhrases) > 0 {
			analysis.Summary = fmt.Sprintf("Multiple errors detected: %s (%d errors, %d warnings)", 
				strings.Join(errorPhrases[:min(2, len(errorPhrases))], ", "), errorCount, warnCount)
			analysis.RootCause = fmt.Sprintf("Recurring issues detected: %s. Manual investigation recommended to identify root cause.", 
				strings.Join(errorPhrases, ", "))
		} else {
			analysis.Summary = fmt.Sprintf("Generic errors detected (%d errors, %d warnings)", errorCount, warnCount)
			analysis.RootCause = "Unable to identify specific pattern from error messages. Review full error logs for stack traces and context."
		}
		
		analysis.Fixes = []string{
			"Review full error logs with stack traces for detailed context",
			"Check recent deployments or configuration changes",
			"Verify all required services and dependencies are running",
			"Check application health endpoints and metrics",
		}
	}

	// Check if related to previous issues
	if len(ctx.Analyses) > 0 {
		latest := ctx.Analyses[len(ctx.Analyses)-1]
		if !latest.Resolved {
			analysis.Context = fmt.Sprintf("May be related to previous unresolved issue: %s (%s)", latest.Summary, latest.Severity)
		}
	}

	return analysis
}

// extractCommonPhrases finds commonly repeated phrases in error messages
func (f *FallbackAnalyzer) extractCommonPhrases(messages []string) []string {
	if len(messages) == 0 {
		return []string{}
	}
	
	// Count word frequencies
	wordCount := make(map[string]int)
	for _, msg := range messages {
		words := strings.Fields(msg)
		for _, word := range words {
			// Skip common words
			if len(word) > 3 && !isCommonWord(word) {
				wordCount[word]++
			}
		}
	}
	
	// Find top repeated words
	type wordFreq struct {
		word  string
		count int
	}
	var frequencies []wordFreq
	for word, count := range wordCount {
		if count > 1 { // Only words that appear multiple times
			frequencies = append(frequencies, wordFreq{word, count})
		}
	}
	
	// Sort by frequency
	if len(frequencies) == 0 {
		return []string{}
	}
	
	// Simple bubble sort for top 3
	for i := 0; i < min(3, len(frequencies)); i++ {
		for j := i + 1; j < len(frequencies); j++ {
			if frequencies[j].count > frequencies[i].count {
				frequencies[i], frequencies[j] = frequencies[j], frequencies[i]
			}
		}
	}
	
	// Return top phrases
	result := []string{}
	for i := 0; i < min(3, len(frequencies)); i++ {
		result = append(result, frequencies[i].word)
	}
	return result
}

func isCommonWord(word string) bool {
	common := []string{"error", "failed", "warning", "info", "the", "and", "for", "from", "with", "that", "this"}
	wordLower := strings.ToLower(word)
	for _, c := range common {
		if wordLower == c {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}