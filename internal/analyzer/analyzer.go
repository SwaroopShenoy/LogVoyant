package analyzer

import (
	"fmt"
	"time"

	"logvoyant/internal/storage"
)

type Config struct {
	Storage    storage.Storage
	GroqAPIKey string
}

type Analyzer struct {
	config   *Config
	llm      *GroqClient
	fallback *FallbackAnalyzer
}

func New(cfg *Config) *Analyzer {
	var llm *GroqClient
	if cfg.GroqAPIKey != "" {
		llm = NewGroqClient(cfg.GroqAPIKey)
	}

	return &Analyzer{
		config:   cfg,
		llm:      llm,
		fallback: NewFallbackAnalyzer(),
	}
}

// Analyze runs context-aware analysis on logs
func (a *Analyzer) Analyze(streamID string, logs []storage.LogLine) (*storage.Analysis, error) {
	if len(logs) == 0 {
		return nil, fmt.Errorf("no logs to analyze")
	}

	// 1. Load historical context
	ctx, err := a.config.Storage.GetContext(streamID)
	if err != nil {
		return nil, err
	}

	// 2. Build enriched prompt with history
	prompt := a.buildPrompt(streamID, logs, ctx)

	// 3. Get analysis (LLM or fallback)
	var analysis *storage.Analysis
	if a.llm != nil {
		analysis, err = a.llm.Analyze(prompt)
		if err != nil {
			// Fallback to pattern matching if LLM fails
			analysis = a.fallback.Analyze(logs, ctx)
		}
	} else {
		// No LLM configured, use fallback
		analysis = a.fallback.Analyze(logs, ctx)
	}

	analysis.StreamID = streamID
	analysis.Timestamp = time.Now()

	return analysis, nil
}

func (a *Analyzer) buildPrompt(streamID string, logs []storage.LogLine, ctx *storage.StreamContext) string {
	prompt := fmt.Sprintf("# Log Analysis for Stream: %s\n\n", streamID)

	// Add historical context
	if len(ctx.Analyses) > 0 {
		prompt += "## Historical Context\n"
		for i, analysis := range ctx.Analyses {
			if i >= 3 {
				break // Only include last 3 analyses
			}
			resolvedStr := "UNRESOLVED"
			if analysis.Resolved {
				resolvedStr = "RESOLVED"
			}
			prompt += fmt.Sprintf("- %s: %s (%s, %s)\n",
				analysis.Timestamp.Format("15:04"),
				analysis.Summary,
				analysis.Severity,
				resolvedStr,
			)
		}
		prompt += "\n"
	}

	// Add common patterns
	if len(ctx.Patterns.CommonErrors) > 0 {
		prompt += "## Common Error Patterns\n"
		for _, pattern := range ctx.Patterns.CommonErrors {
			prompt += fmt.Sprintf("- %s\n", pattern)
		}
		prompt += fmt.Sprintf("- Current error rate: %.1f%%\n\n", ctx.Patterns.ErrorRate*100)
	}

	// Add recent logs
	prompt += "## Recent Logs (Last 100 Lines)\n"
	for _, log := range logs {
		prompt += fmt.Sprintf("[%s] [%s] %s\n",
			log.Timestamp.Format("15:04:05"),
			log.Level,
			log.Message,
		)
	}

	// Add analysis instructions
	prompt += `

## Analysis Tasks
1. Is this related to any previous issues in the historical context?
2. Identify the root cause
3. Assign severity: P0 (critical), P1 (high), P2 (medium), P3 (low)
4. Suggest 2-3 actionable fixes

Respond in JSON format:
{
  "summary": "Brief one-line summary",
  "root_cause": "Detailed root cause analysis",
  "severity": "P0|P1|P2|P3",
  "fixes": ["Fix 1", "Fix 2", "Fix 3"],
  "context": "How this relates to previous issues"
}
`

	return prompt
}