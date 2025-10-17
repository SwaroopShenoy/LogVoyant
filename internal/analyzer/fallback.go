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
	// Count errors and collect keywords
	errorCount := 0
	allMessages := make([]string, 0)
	
	for _, log := range logs {
		if log.Level == "ERROR" || log.Level == "FATAL" {
			errorCount++
		}
		allMessages = append(allMessages, strings.ToLower(log.Message))
	}

	// Find matching pattern
	var matchedPattern *ErrorPattern
	maxMatches := 0

	for i := range f.patterns {
		pattern := &f.patterns[i]
		matches := 0
		for _, keyword := range pattern.Keywords {
			for _, msg := range allMessages {
				if strings.Contains(msg, strings.ToLower(keyword)) {
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
		Severity: "P3", // Default
	}

	if matchedPattern != nil {
		analysis.Summary = fmt.Sprintf("Detected %s issue (%d errors in logs)", 
			matchedPattern.Keywords[0], errorCount)
		analysis.RootCause = matchedPattern.RootCause
		analysis.Fixes = matchedPattern.Fixes
		analysis.Severity = matchedPattern.Severity
	} else {
		// Generic analysis
		analysis.Summary = fmt.Sprintf("Generic errors detected (%d errors)", errorCount)
		analysis.RootCause = "Unable to identify specific pattern. Manual investigation recommended."
		analysis.Fixes = []string{
			"Review full error logs for detailed stack traces",
			"Check recent deployments or configuration changes",
			"Consult service documentation for common issues",
		}
	}

	// Check if related to previous issues
	if len(ctx.Analyses) > 0 {
		latest := ctx.Analyses[len(ctx.Analyses)-1]
		if !latest.Resolved {
			analysis.Context = fmt.Sprintf("May be related to previous unresolved issue: %s", latest.Summary)
		}
	}

	return analysis
}