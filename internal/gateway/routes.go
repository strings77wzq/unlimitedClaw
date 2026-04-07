package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/strings77wzq/golem/core/session"
)

type healthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type versionResponse struct {
	Version string `json:"version"`
}

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type chatResponse struct {
	SessionID string `json:"session_id"`
	Response  string `json:"response"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /health/providers", s.handleProvidersHealth)
	s.mux.HandleFunc("GET /api/version", s.handleVersion)
	s.mux.HandleFunc("POST /api/chat", s.handleChat)
	s.mux.HandleFunc("POST /api/chat/stream", s.handleChatStream)
	s.mux.HandleFunc("GET /api/sessions/{id}/export", s.handleSessionExport)
	s.mux.HandleFunc("POST /api/sessions/import", s.handleSessionImport)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProvidersHealth(w http.ResponseWriter, r *http.Request) {
	if s.healthChecker == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "not_configured",
			"message": "health checker not configured",
		})
		return
	}

	statuses := s.healthChecker.GetAllStatuses()
	writeJSON(w, http.StatusOK, statuses)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	resp := versionResponse{
		Version: "dev",
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed to read request body", slog.Any("error", err))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	defer r.Body.Close()

	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.logger.Error("failed to parse JSON", slog.Any("error", err))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "message is required"})
		return
	}

	response, err := s.agent.HandleMessage(r.Context(), req.SessionID, req.Message)
	if err != nil {
		s.logger.Error("agent error", slog.Any("error", err), slog.String("session_id", req.SessionID))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
		return
	}

	resp := chatResponse{
		SessionID: req.SessionID,
		Response:  response,
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode JSON response", http.StatusInternalServerError)
	}
}

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "streaming not supported"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed to read request body", slog.Any("error", err))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	defer r.Body.Close()

	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.logger.Error("failed to parse JSON", slog.Any("error", err))
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "message is required"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	streamer, isStreaming := s.agent.(StreamingAgentHandler)
	if isStreaming {
		tokens := make(chan string, 32)
		errCh := make(chan error, 1)

		go func() {
			errCh <- streamer.HandleMessageStream(r.Context(), req.SessionID, req.Message, tokens)
		}()

		for token := range tokens {
			fmt.Fprintf(w, "data: %s\n\n", token)
			flusher.Flush()
		}

		if err := <-errCh; err != nil {
			s.logger.Error("stream error", slog.Any("error", err), slog.String("session_id", req.SessionID))
			fmt.Fprintf(w, "event: error\ndata: internal server error\n\n")
			flusher.Flush()
			return
		}
	} else {
		response, err := s.agent.HandleMessage(r.Context(), req.SessionID, req.Message)
		if err != nil {
			s.logger.Error("agent error", slog.Any("error", err), slog.String("session_id", req.SessionID))
			fmt.Fprintf(w, "event: error\ndata: internal server error\n\n")
			flusher.Flush()
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", response)
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
	flusher.Flush()
}

func (s *Server) handleSessionExport(w http.ResponseWriter, r *http.Request) {
	if s.sessionStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "session store not configured"})
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "session ID required"})
		return
	}

	sess, ok := s.sessionStore.Get(sessionID)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "session not found"})
		return
	}

	exportData := sess.Export()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", sessionID))
	writeJSON(w, http.StatusOK, exportData)
}

func (s *Server) handleSessionImport(w http.ResponseWriter, r *http.Request) {
	if s.sessionStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "session store not configured"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	defer r.Body.Close()

	sess, err := session.ImportFromJSON(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("import failed: %v", err)})
		return
	}

	if err := s.sessionStore.Save(sess); err != nil {
		s.logger.Error("failed to save imported session", slog.Any("error", err))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to save session"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"session_id": sess.ID,
		"status":     "imported",
	})
}
