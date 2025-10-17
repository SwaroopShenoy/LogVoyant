package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"logvoyant/internal/storage"
)

func (s *Server) handleListStreams(w http.ResponseWriter, r *http.Request) {
	streams, err := s.config.Storage.ListStreams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, streams)
}

func (s *Server) handleGetStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")

	stream, err := s.config.Storage.GetStream(streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respondJSON(w, stream)
}

func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")

	// Parse query params
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}

	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err == nil {
			since = time.Now().Add(-duration)
		}
	}

	opts := storage.GetLogsOptions{
		Limit: limit,
		Since: since,
	}

	logs, err := s.config.Storage.GetLogs(streamID, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, logs)
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")

	// Get recent logs
	logs, err := s.config.Storage.GetLogs(streamID, storage.GetLogsOptions{Limit: 100})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(logs) == 0 {
		http.Error(w, "no logs to analyze", http.StatusBadRequest)
		return
	}

	// Run analysis
	analysis, err := s.analyzer.Analyze(streamID, logs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store analysis
	if err := s.config.Storage.StoreAnalysis(analysis); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update context with new analysis summary
	ctx, _ := s.config.Storage.GetContext(streamID)
	ctx.Analyses = append(ctx.Analyses, storage.AnalysisSummary{
		Timestamp: analysis.Timestamp,
		Summary:   analysis.Summary,
		RootCause: analysis.RootCause,
		Severity:  analysis.Severity,
		Resolved:  false,
	})
	s.config.Storage.UpdateContext(streamID, ctx)

	respondJSON(w, analysis)
}

func (s *Server) handleGetContext(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")

	ctx, err := s.config.Storage.GetContext(streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, ctx)
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "id")

	var req struct {
		AnalysisIndex int    `json:"analysis_index"`
		Note          string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, err := s.config.Storage.GetContext(streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.AnalysisIndex >= len(ctx.Analyses) {
		http.Error(w, "invalid analysis index", http.StatusBadRequest)
		return
	}

	ctx.Analyses[req.AnalysisIndex].Resolved = true
	ctx.Analyses[req.AnalysisIndex].ResolutionNote = req.Note

	if err := s.config.Storage.UpdateContext(streamID, ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]bool{"success": true})
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}