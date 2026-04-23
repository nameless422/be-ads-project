package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	biquerydomain "be_ads_project/internal/modules/reporting/domain"
)

type SnapshotReader interface {
	ListAccountSnapshots(context.Context) ([]biquerydomain.AccountSnapshot, error)
	ListCampaigns(context.Context, biquerydomain.CampaignFilter) ([]biquerydomain.CampaignView, error)
	Ping(context.Context) error
}

type InsightReader interface {
	QueryInsightSummary(context.Context, biquerydomain.InsightSummaryFilter) ([]biquerydomain.InsightSummaryRow, error)
	Ping(context.Context) error
}

type Server struct {
	logger       *log.Logger
	snapshotRepo SnapshotReader
	insightRepo  InsightReader
}

func NewServer(snapshotRepo SnapshotReader, insightRepo InsightReader, logger *log.Logger) *Server {
	return &Server{
		logger:       logger,
		snapshotRepo: snapshotRepo,
		insightRepo:  insightRepo,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/bi/snapshots", s.handleSnapshots)
	mux.HandleFunc("/api/bi/campaigns", s.handleCampaigns)
	mux.HandleFunc("/api/bi/insights/summary", s.handleInsightSummary)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.snapshotRepo.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "mysql": err.Error()})
		return
	}
	if err := s.insightRepo.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "clickhouse": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	items, err := s.snapshotRepo.ListAccountSnapshots(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCampaigns(w http.ResponseWriter, r *http.Request) {
	items, err := s.snapshotRepo.ListCampaigns(r.Context(), biquerydomain.CampaignFilter{
		Platform:  r.URL.Query().Get("platform"),
		AccountID: r.URL.Query().Get("account_id"),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleInsightSummary(w http.ResponseWriter, r *http.Request) {
	filter := biquerydomain.InsightSummaryFilter{
		Platform:  r.URL.Query().Get("platform"),
		AccountID: r.URL.Query().Get("platform_account_id"),
	}
	if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
		parsed, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date_from"})
			return
		}
		filter.DateFrom = parsed
	}
	if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
		parsed, err := time.Parse("2006-01-02", dateTo)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date_to"})
			return
		}
		filter.DateTo = parsed
	}

	items, err := s.insightRepo.QueryInsightSummary(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
