package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
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
	QueryInsightDetails(context.Context, biquerydomain.InsightDetailFilter) ([]biquerydomain.InsightDetailRow, error)
	QueryCampaignDiagnostics(context.Context, biquerydomain.CampaignDiagnosticFilter) ([]biquerydomain.CampaignDiagnosticRow, error)
	QuerySearchTerms(context.Context, biquerydomain.SearchTermFilter) ([]biquerydomain.SearchTermRow, error)
	Ping(context.Context) error
}

type ControlReader interface {
	ListWorkerLeases(context.Context) ([]biquerydomain.WorkerLeaseView, error)
	ListShardAssignments(context.Context) ([]biquerydomain.ShardAssignmentView, error)
	CountRawRecords(context.Context) (int64, error)
	CountOutboxByStatus(context.Context, string) (int64, error)
}

type DeadLetterReader interface {
	DeadLetterCount(context.Context) (uint64, error)
	ListDeadLetters(context.Context, int) ([]biquerydomain.DeadLetterView, error)
	ReplayDeadLetters(context.Context, biquerydomain.ReplayRequest) (*biquerydomain.ActionResult, error)
}

type ControlActionHandler interface {
	DispatchBackfill(context.Context, biquerydomain.BackfillRequest) (*biquerydomain.ActionResult, error)
}

type Server struct {
	logger       *log.Logger
	snapshotRepo SnapshotReader
	insightRepo  InsightReader
	controlRepo  ControlReader
	dlqReader    DeadLetterReader
	actions      ControlActionHandler
}

func NewServer(snapshotRepo SnapshotReader, insightRepo InsightReader, controlRepo ControlReader, dlqReader DeadLetterReader, actions ControlActionHandler, logger *log.Logger) *Server {
	return &Server{
		logger:       logger,
		snapshotRepo: snapshotRepo,
		insightRepo:  insightRepo,
		controlRepo:  controlRepo,
		dlqReader:    dlqReader,
		actions:      actions,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/bi", s.handleBIPanel)
	mux.HandleFunc("/api/bi/snapshots", s.handleSnapshots)
	mux.HandleFunc("/api/bi/campaigns", s.handleCampaigns)
	mux.HandleFunc("/api/bi/insights/summary", s.handleInsightSummary)
	mux.HandleFunc("/api/bi/insights/detail", s.handleInsightDetail)
	mux.HandleFunc("/api/bi/campaign-diagnostics", s.handleCampaignDiagnostics)
	mux.HandleFunc("/api/bi/search-terms", s.handleSearchTerms)
	mux.HandleFunc("/api/control/overview", s.handleControlOverview)
	mux.HandleFunc("/api/control/leases", s.handleControlLeases)
	mux.HandleFunc("/api/control/shards", s.handleControlShards)
	mux.HandleFunc("/api/control/dlq", s.handleControlDLQ)
	mux.HandleFunc("/api/control/dlq/replay", s.handleReplayDLQ)
	mux.HandleFunc("/api/control/backfill", s.handleBackfill)
	mux.HandleFunc("/", s.handleDashboard)
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

func (s *Server) handleInsightDetail(w http.ResponseWriter, r *http.Request) {
	filter := biquerydomain.InsightDetailFilter{
		Platform:    strings.TrimSpace(r.URL.Query().Get("platform")),
		AccountID:   strings.TrimSpace(r.URL.Query().Get("platform_account_id")),
		EntityLevel: strings.TrimSpace(r.URL.Query().Get("entity_level")),
		Device:      strings.TrimSpace(r.URL.Query().Get("device")),
		Network:     strings.TrimSpace(r.URL.Query().Get("network")),
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		filter.Limit = limit
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

	items, err := s.insightRepo.QueryInsightDetails(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCampaignDiagnostics(w http.ResponseWriter, r *http.Request) {
	filter := biquerydomain.CampaignDiagnosticFilter{
		Platform:  strings.TrimSpace(r.URL.Query().Get("platform")),
		AccountID: strings.TrimSpace(r.URL.Query().Get("platform_account_id")),
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		filter.Limit = limit
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
	items, err := s.insightRepo.QueryCampaignDiagnostics(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleSearchTerms(w http.ResponseWriter, r *http.Request) {
	filter := biquerydomain.SearchTermFilter{
		Platform:        strings.TrimSpace(r.URL.Query().Get("platform")),
		AccountID:       strings.TrimSpace(r.URL.Query().Get("platform_account_id")),
		MatchType:       strings.TrimSpace(r.URL.Query().Get("match_type")),
		SearchTermQuery: strings.TrimSpace(r.URL.Query().Get("search_term")),
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		filter.Limit = limit
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
	items, err := s.insightRepo.QuerySearchTerms(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleControlOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := s.loadOverview(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleControlLeases(w http.ResponseWriter, r *http.Request) {
	items, err := s.controlRepo.ListWorkerLeases(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleControlShards(w http.ResponseWriter, r *http.Request) {
	items, err := s.controlRepo.ListShardAssignments(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleControlDLQ(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := s.dlqReader.ListDeadLetters(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleReplayDLQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req biquerydomain.ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	result, err := s.dlqReader.ReplayDeadLetters(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleBackfill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req biquerydomain.BackfillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	result, err := s.actions.DispatchBackfill(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	overview, err := s.loadOverview(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = dashboardTemplate.Execute(w, overview)
}

func (s *Server) handleBIPanel(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/bi" {
		http.NotFound(w, r)
		return
	}

	filter := biquerydomain.CampaignFilter{
		Platform:  strings.TrimSpace(r.URL.Query().Get("platform")),
		AccountID: strings.TrimSpace(r.URL.Query().Get("account_id")),
	}
	insightFilter := biquerydomain.InsightSummaryFilter{
		Platform:  filter.Platform,
		AccountID: filter.AccountID,
	}
	detailFilter := biquerydomain.InsightDetailFilter{
		Platform:  filter.Platform,
		AccountID: filter.AccountID,
		Limit:     200,
	}
	diagnosticFilter := biquerydomain.CampaignDiagnosticFilter{
		Platform:  filter.Platform,
		AccountID: filter.AccountID,
		Limit:     200,
	}
	searchTermFilter := biquerydomain.SearchTermFilter{
		Platform:  filter.Platform,
		AccountID: filter.AccountID,
		Limit:     100,
	}
	if dateFrom := strings.TrimSpace(r.URL.Query().Get("date_from")); dateFrom != "" {
		parsed, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			http.Error(w, "invalid date_from", http.StatusBadRequest)
			return
		}
		insightFilter.DateFrom = parsed
		detailFilter.DateFrom = parsed
		diagnosticFilter.DateFrom = parsed
		searchTermFilter.DateFrom = parsed
	}
	if dateTo := strings.TrimSpace(r.URL.Query().Get("date_to")); dateTo != "" {
		parsed, err := time.Parse("2006-01-02", dateTo)
		if err != nil {
			http.Error(w, "invalid date_to", http.StatusBadRequest)
			return
		}
		insightFilter.DateTo = parsed
		detailFilter.DateTo = parsed
		diagnosticFilter.DateTo = parsed
		searchTermFilter.DateTo = parsed
	}
	detailFilter.EntityLevel = strings.TrimSpace(r.URL.Query().Get("entity_level"))
	detailFilter.Device = strings.TrimSpace(r.URL.Query().Get("device"))
	detailFilter.Network = strings.TrimSpace(r.URL.Query().Get("network"))
	searchTermFilter.MatchType = strings.TrimSpace(r.URL.Query().Get("match_type"))
	searchTermFilter.SearchTermQuery = strings.TrimSpace(r.URL.Query().Get("search_term"))
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("detail_limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			http.Error(w, "invalid detail_limit", http.StatusBadRequest)
			return
		}
		detailFilter.Limit = limit
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("search_term_limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			http.Error(w, "invalid search_term_limit", http.StatusBadRequest)
			return
		}
		searchTermFilter.Limit = limit
	}

	snapshots, err := s.snapshotRepo.ListAccountSnapshots(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	campaigns, err := s.snapshotRepo.ListCampaigns(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	insights, err := s.insightRepo.QueryInsightSummary(r.Context(), insightFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	insightDetails, err := s.insightRepo.QueryInsightDetails(r.Context(), detailFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	campaignDiagnostics, err := s.insightRepo.QueryCampaignDiagnostics(r.Context(), diagnosticFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	searchTerms, err := s.insightRepo.QuerySearchTerms(r.Context(), searchTermFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := buildBIPageView(filter, insightFilter, detailFilter, searchTermFilter, snapshots, campaigns, insights, insightDetails, campaignDiagnostics, searchTerms)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = biPanelTemplate.Execute(w, page)
}

func (s *Server) loadOverview(ctx context.Context) (*biquerydomain.ControlPanelOverview, error) {
	leases, err := s.controlRepo.ListWorkerLeases(ctx)
	if err != nil {
		return nil, err
	}
	assignments, err := s.controlRepo.ListShardAssignments(ctx)
	if err != nil {
		return nil, err
	}
	rawCount, err := s.controlRepo.CountRawRecords(ctx)
	if err != nil {
		return nil, err
	}
	outboxPending, err := s.controlRepo.CountOutboxByStatus(ctx, "pending")
	if err != nil {
		return nil, err
	}
	outboxPublished, err := s.controlRepo.CountOutboxByStatus(ctx, "published")
	if err != nil {
		return nil, err
	}
	snapshots, err := s.snapshotRepo.ListAccountSnapshots(ctx)
	if err != nil {
		return nil, err
	}
	var dlqCount uint64
	if s.dlqReader != nil {
		dlqCount, _ = s.dlqReader.DeadLetterCount(ctx)
	}
	return &biquerydomain.ControlPanelOverview{
		GeneratedAt:      time.Now().UTC(),
		WorkerLeases:     leases,
		ShardAssignments: assignments,
		RawRecordCount:   rawCount,
		OutboxPending:    outboxPending,
		OutboxPublished:  outboxPublished,
		DeadLetterCount:  dlqCount,
		Snapshots:        snapshots,
	}, nil
}

type biPanelView struct {
	GeneratedAt                         time.Time
	Platform                            string
	AccountID                           string
	DateFrom                            string
	DateTo                              string
	EntityLevel                         string
	Device                              string
	Network                             string
	DetailLimit                         int
	MatchType                           string
	SearchTerm                          string
	SearchTermLimit                     int
	SnapshotCount                       int
	CampaignCount                       int
	InsightRowCount                     int
	InsightDetailRowCount               int
	CampaignDiagnosticRowCount          int
	SearchTermRowCount                  int
	TotalImpressions                    int64
	TotalClicks                         int64
	TotalSpend                          string
	TotalConversions                    string
	TotalAllConversions                 string
	TotalConversionsValue               string
	AvgCostPerConversion                string
	AvgCostPerAllConversions            string
	AvgSearchImpressionShare            string
	AvgSearchTopImpressionShare         string
	AvgSearchAbsoluteTopImpressionShare string
	SearchTermClicks                    int64
	SearchTermSpend                     string
	SearchTermConversions               string
	SearchTermConversionsValue          string
	ImpressionsSVG                      template.HTML
	ClicksSVG                           template.HTML
	SpendSVG                            template.HTML
	PlatformSummary                     []biPlatformSummary
	AccountSummary                      []biAccountSummary
	Snapshots                           []biquerydomain.AccountSnapshot
	Campaigns                           []biquerydomain.CampaignView
	Insights                            []biquerydomain.InsightSummaryRow
	InsightDetails                      []biquerydomain.InsightDetailRow
	CampaignDiagnostics                 []biquerydomain.CampaignDiagnosticRow
	SearchTerms                         []biquerydomain.SearchTermRow
}

type biTrendPoint struct {
	Label       string
	Impressions int64
	Clicks      int64
	Spend       float64
}

type biPlatformSummary struct {
	Platform         string
	AccountCount     int
	CampaignCount    int
	AdGroupCount     int
	AdCount          int
	InsightCount     int
	TotalImpressions int64
	TotalClicks      int64
	TotalSpend       string
}

type biAccountSummary struct {
	Platform         string
	AccountID        string
	AccountName      string
	SourceMode       string
	CampaignCount    int
	AdGroupCount     int
	AdCount          int
	InsightCount     int
	TotalImpressions int64
	TotalClicks      int64
	TotalSpend       string
}

func buildBIPageView(
	filter biquerydomain.CampaignFilter,
	insightFilter biquerydomain.InsightSummaryFilter,
	detailFilter biquerydomain.InsightDetailFilter,
	searchTermFilter biquerydomain.SearchTermFilter,
	snapshots []biquerydomain.AccountSnapshot,
	campaigns []biquerydomain.CampaignView,
	insights []biquerydomain.InsightSummaryRow,
	insightDetails []biquerydomain.InsightDetailRow,
	campaignDiagnostics []biquerydomain.CampaignDiagnosticRow,
	searchTerms []biquerydomain.SearchTermRow,
) biPanelView {
	var totalImpressions int64
	var totalClicks int64
	var totalSpend float64
	var totalConversions float64
	var totalAllConversions float64
	var totalConversionsValue float64
	var totalSearchImpressionShare float64
	var totalSearchTopImpressionShare float64
	var totalSearchAbsoluteTopImpressionShare float64
	var searchTermClicks int64
	var searchTermSpend float64
	var searchTermConversions float64
	var searchTermConversionsValue float64
	for _, item := range insights {
		totalImpressions += item.Impressions
		totalClicks += item.Clicks
		if spend, err := strconv.ParseFloat(item.Spend, 64); err == nil {
			totalSpend += spend
		}
		if conversions, err := strconv.ParseFloat(item.Conversions, 64); err == nil {
			totalConversions += conversions
		}
		if allConversions, err := strconv.ParseFloat(item.AllConversions, 64); err == nil {
			totalAllConversions += allConversions
		}
		if conversionsValue, err := strconv.ParseFloat(item.ConversionsValue, 64); err == nil {
			totalConversionsValue += conversionsValue
		}
	}
	for _, item := range campaignDiagnostics {
		if value, err := strconv.ParseFloat(item.SearchImpressionShare, 64); err == nil {
			totalSearchImpressionShare += value
		}
		if value, err := strconv.ParseFloat(item.SearchTopImpressionShare, 64); err == nil {
			totalSearchTopImpressionShare += value
		}
		if value, err := strconv.ParseFloat(item.SearchAbsoluteTopImpressionShare, 64); err == nil {
			totalSearchAbsoluteTopImpressionShare += value
		}
	}
	for _, item := range searchTerms {
		searchTermClicks += item.Clicks
		if value, err := strconv.ParseFloat(item.Spend, 64); err == nil {
			searchTermSpend += value
		}
		if value, err := strconv.ParseFloat(item.Conversions, 64); err == nil {
			searchTermConversions += value
		}
		if value, err := strconv.ParseFloat(item.ConversionsValue, 64); err == nil {
			searchTermConversionsValue += value
		}
	}

	avgCostPerConversion := "0.00"
	if totalConversions > 0 {
		avgCostPerConversion = strconv.FormatFloat(totalSpend/totalConversions, 'f', 2, 64)
	}
	avgCostPerAllConversions := "0.00"
	if totalAllConversions > 0 {
		avgCostPerAllConversions = strconv.FormatFloat(totalSpend/totalAllConversions, 'f', 2, 64)
	}
	avgSearchImpressionShare := "0.0000"
	avgSearchTopImpressionShare := "0.0000"
	avgSearchAbsoluteTopImpressionShare := "0.0000"
	if len(campaignDiagnostics) > 0 {
		divisor := float64(len(campaignDiagnostics))
		avgSearchImpressionShare = strconv.FormatFloat(totalSearchImpressionShare/divisor, 'f', 4, 64)
		avgSearchTopImpressionShare = strconv.FormatFloat(totalSearchTopImpressionShare/divisor, 'f', 4, 64)
		avgSearchAbsoluteTopImpressionShare = strconv.FormatFloat(totalSearchAbsoluteTopImpressionShare/divisor, 'f', 4, 64)
	}

	page := biPanelView{
		GeneratedAt:                         time.Now().UTC(),
		Platform:                            filter.Platform,
		AccountID:                           filter.AccountID,
		EntityLevel:                         detailFilter.EntityLevel,
		Device:                              detailFilter.Device,
		Network:                             detailFilter.Network,
		DetailLimit:                         detailFilter.Limit,
		MatchType:                           searchTermFilter.MatchType,
		SearchTerm:                          searchTermFilter.SearchTermQuery,
		SearchTermLimit:                     searchTermFilter.Limit,
		SnapshotCount:                       len(snapshots),
		CampaignCount:                       len(campaigns),
		InsightRowCount:                     len(insights),
		InsightDetailRowCount:               len(insightDetails),
		CampaignDiagnosticRowCount:          len(campaignDiagnostics),
		SearchTermRowCount:                  len(searchTerms),
		TotalImpressions:                    totalImpressions,
		TotalClicks:                         totalClicks,
		TotalSpend:                          strconv.FormatFloat(totalSpend, 'f', 2, 64),
		TotalConversions:                    strconv.FormatFloat(totalConversions, 'f', 2, 64),
		TotalAllConversions:                 strconv.FormatFloat(totalAllConversions, 'f', 2, 64),
		TotalConversionsValue:               strconv.FormatFloat(totalConversionsValue, 'f', 2, 64),
		AvgCostPerConversion:                avgCostPerConversion,
		AvgCostPerAllConversions:            avgCostPerAllConversions,
		AvgSearchImpressionShare:            avgSearchImpressionShare,
		AvgSearchTopImpressionShare:         avgSearchTopImpressionShare,
		AvgSearchAbsoluteTopImpressionShare: avgSearchAbsoluteTopImpressionShare,
		SearchTermClicks:                    searchTermClicks,
		SearchTermSpend:                     strconv.FormatFloat(searchTermSpend, 'f', 2, 64),
		SearchTermConversions:               strconv.FormatFloat(searchTermConversions, 'f', 2, 64),
		SearchTermConversionsValue:          strconv.FormatFloat(searchTermConversionsValue, 'f', 2, 64),
		ImpressionsSVG:                      buildTrendBars(insights, "impressions"),
		ClicksSVG:                           buildTrendBars(insights, "clicks"),
		SpendSVG:                            buildTrendBars(insights, "spend"),
		PlatformSummary:                     buildPlatformSummary(snapshots, insights),
		AccountSummary:                      buildAccountSummary(snapshots, insights),
		Snapshots:                           snapshots,
		Campaigns:                           campaigns,
		Insights:                            insights,
		InsightDetails:                      insightDetails,
		CampaignDiagnostics:                 campaignDiagnostics,
		SearchTerms:                         searchTerms,
	}
	if !insightFilter.DateFrom.IsZero() {
		page.DateFrom = insightFilter.DateFrom.Format("2006-01-02")
	}
	if !insightFilter.DateTo.IsZero() {
		page.DateTo = insightFilter.DateTo.Format("2006-01-02")
	}
	return page
}

func buildPlatformSummary(snapshots []biquerydomain.AccountSnapshot, insights []biquerydomain.InsightSummaryRow) []biPlatformSummary {
	type row struct {
		accounts         map[string]struct{}
		campaignCount    int
		adGroupCount     int
		adCount          int
		insightCount     int
		totalImpressions int64
		totalClicks      int64
		totalSpend       float64
	}

	rows := make(map[string]*row, len(snapshots))
	for _, snapshot := range snapshots {
		key := string(snapshot.Platform)
		if _, ok := rows[key]; !ok {
			rows[key] = &row{accounts: map[string]struct{}{}}
		}
		current := rows[key]
		current.accounts[snapshot.AccountID] = struct{}{}
		current.campaignCount += snapshot.CampaignCount
		current.adGroupCount += snapshot.AdGroupCount
		current.adCount += snapshot.AdCount
		current.insightCount += snapshot.InsightCount
	}

	for _, item := range insights {
		key := string(item.Platform)
		if _, ok := rows[key]; !ok {
			rows[key] = &row{accounts: map[string]struct{}{}}
		}
		current := rows[key]
		current.totalImpressions += item.Impressions
		current.totalClicks += item.Clicks
		if spend, err := strconv.ParseFloat(item.Spend, 64); err == nil {
			current.totalSpend += spend
		}
	}

	result := make([]biPlatformSummary, 0, len(rows))
	for platform, item := range rows {
		result = append(result, biPlatformSummary{
			Platform:         platform,
			AccountCount:     len(item.accounts),
			CampaignCount:    item.campaignCount,
			AdGroupCount:     item.adGroupCount,
			AdCount:          item.adCount,
			InsightCount:     item.insightCount,
			TotalImpressions: item.totalImpressions,
			TotalClicks:      item.totalClicks,
			TotalSpend:       strconv.FormatFloat(item.totalSpend, 'f', 2, 64),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Platform < result[j].Platform })
	return result
}

func buildAccountSummary(snapshots []biquerydomain.AccountSnapshot, insights []biquerydomain.InsightSummaryRow) []biAccountSummary {
	type row struct {
		platform         string
		accountID        string
		accountName      string
		sourceMode       string
		campaignCount    int
		adGroupCount     int
		adCount          int
		insightCount     int
		totalImpressions int64
		totalClicks      int64
		totalSpend       float64
	}

	rows := make(map[string]*row, len(snapshots))
	accountByPlatformAccountID := make(map[string]string, len(snapshots))

	for _, snapshot := range snapshots {
		key := fmt.Sprintf("%s/%s", snapshot.Platform, snapshot.AccountID)
		if _, ok := rows[key]; !ok {
			rows[key] = &row{
				platform:    string(snapshot.Platform),
				accountID:   snapshot.AccountID,
				accountName: snapshot.AccountName,
				sourceMode:  snapshot.LastSourceMode,
			}
		}
		accountByPlatformAccountID[snapshot.PlatformAccountID] = key
		current := rows[key]
		current.campaignCount += snapshot.CampaignCount
		current.adGroupCount += snapshot.AdGroupCount
		current.adCount += snapshot.AdCount
		current.insightCount += snapshot.InsightCount
	}

	for _, item := range insights {
		key, ok := accountByPlatformAccountID[item.PlatformAccountID]
		if !ok {
			key = fmt.Sprintf("%s/%s", item.Platform, item.PlatformAccountID)
			if _, exists := rows[key]; !exists {
				rows[key] = &row{
					platform:   string(item.Platform),
					accountID:  item.PlatformAccountID,
					sourceMode: "unknown",
				}
			}
		}
		current := rows[key]
		current.totalImpressions += item.Impressions
		current.totalClicks += item.Clicks
		if spend, err := strconv.ParseFloat(item.Spend, 64); err == nil {
			current.totalSpend += spend
		}
	}

	result := make([]biAccountSummary, 0, len(rows))
	for _, item := range rows {
		result = append(result, biAccountSummary{
			Platform:         item.platform,
			AccountID:        item.accountID,
			AccountName:      item.accountName,
			SourceMode:       item.sourceMode,
			CampaignCount:    item.campaignCount,
			AdGroupCount:     item.adGroupCount,
			AdCount:          item.adCount,
			InsightCount:     item.insightCount,
			TotalImpressions: item.totalImpressions,
			TotalClicks:      item.totalClicks,
			TotalSpend:       strconv.FormatFloat(item.totalSpend, 'f', 2, 64),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Platform == result[j].Platform {
			return result[i].AccountID < result[j].AccountID
		}
		return result[i].Platform < result[j].Platform
	})
	return result
}

func buildTrendBars(insights []biquerydomain.InsightSummaryRow, metric string) template.HTML {
	points := aggregateTrendPoints(insights)
	if len(points) == 0 {
		return template.HTML(`<div class="empty-chart">No data</div>`)
	}

	maxValue := 0.0
	values := make([]float64, 0, len(points))
	for _, point := range points {
		var value float64
		switch metric {
		case "clicks":
			value = float64(point.Clicks)
		case "spend":
			value = point.Spend
		default:
			value = float64(point.Impressions)
		}
		values = append(values, value)
		if value > maxValue {
			maxValue = value
		}
	}
	if maxValue <= 0 {
		maxValue = 1
	}

	var builder strings.Builder
	builder.WriteString(`<svg viewBox="0 0 320 96" preserveAspectRatio="none" class="trend-svg">`)
	barWidth := 320.0 / float64(len(values))
	for idx, value := range values {
		height := (value / maxValue) * 72
		x := float64(idx)*barWidth + 4
		y := 84 - height
		width := barWidth - 8
		if width < 6 {
			width = 6
		}
		builder.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="6" ry="6"></rect>`, x, y, width, height))
	}
	builder.WriteString(`</svg>`)
	return template.HTML(builder.String())
}

func aggregateTrendPoints(insights []biquerydomain.InsightSummaryRow) []biTrendPoint {
	type row struct {
		impressions int64
		clicks      int64
		spend       float64
	}
	grouped := make(map[string]*row, len(insights))
	for _, item := range insights {
		label := item.StatDate.Format("2006-01-02")
		if _, ok := grouped[label]; !ok {
			grouped[label] = &row{}
		}
		current := grouped[label]
		current.impressions += item.Impressions
		current.clicks += item.Clicks
		if spend, err := strconv.ParseFloat(item.Spend, 64); err == nil {
			current.spend += spend
		}
	}

	labels := make([]string, 0, len(grouped))
	for label := range grouped {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	points := make([]biTrendPoint, 0, len(labels))
	for _, label := range labels {
		item := grouped[label]
		points = append(points, biTrendPoint{
			Label:       label,
			Impressions: item.impressions,
			Clicks:      item.clicks,
			Spend:       item.spend,
		})
	}
	return points
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>be_ads Control Panel</title>
  <style>
    :root { --bg:#f3f0e8; --ink:#1f2b24; --muted:#627067; --card:#fffdf8; --line:#d9d1c3; --accent:#0d7c66; --warn:#b85c38; }
    * { box-sizing:border-box; }
    body { margin:0; font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "PingFang SC", "Helvetica Neue", sans-serif; background:linear-gradient(180deg,#efe7d6 0%,#f7f4ec 100%); color:var(--ink); }
    .wrap { max-width:1200px; margin:0 auto; padding:32px 20px 48px; }
    .hero { display:flex; justify-content:space-between; align-items:flex-end; gap:16px; margin-bottom:24px; }
    .hero h1 { margin:0; font-size:36px; letter-spacing:-0.03em; }
    .hero p { margin:8px 0 0; color:var(--muted); }
    .grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:16px; margin-bottom:24px; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:18px; padding:18px; box-shadow:0 10px 30px rgba(45,51,39,0.06); }
    .metric { font-size:32px; font-weight:700; margin:6px 0 0; }
    .label { color:var(--muted); font-size:13px; text-transform:uppercase; letter-spacing:0.08em; }
    .panel { margin-top:18px; }
    .panel h2 { margin:0 0 12px; font-size:18px; }
    table { width:100%; border-collapse:collapse; font-size:14px; }
    th, td { text-align:left; padding:10px 8px; border-bottom:1px solid #ece5d9; }
    th { color:var(--muted); font-weight:600; font-size:12px; text-transform:uppercase; letter-spacing:0.06em; }
    .two { display:grid; grid-template-columns:1.1fr 1fr; gap:16px; }
    .actions { display:grid; grid-template-columns:1fr 1fr; gap:16px; margin-bottom:16px; }
    .field { display:flex; flex-direction:column; gap:6px; margin-bottom:10px; }
    input, select, button { font:inherit; border-radius:12px; border:1px solid var(--line); padding:10px 12px; background:white; }
    button { background:var(--accent); color:white; border:none; cursor:pointer; font-weight:600; }
    button.secondary { background:#d9ece6; color:#145447; }
    .status-dot { width:10px; height:10px; border-radius:50%; display:inline-block; margin-right:8px; background:#0d7c66; }
    .status-dot.warn { background:#b85c38; }
    .status-dot.dim { background:#b8b1a3; }
    .tiny { font-size:12px; color:var(--muted); }
    .badge { display:inline-block; padding:4px 8px; border-radius:999px; background:#e4f1ed; color:var(--accent); font-size:12px; font-weight:600; }
    .warn { background:#f8e8e0; color:var(--warn); }
    @media (max-width: 960px) { .grid,.two { grid-template-columns:1fr 1fr; } }
    @media (max-width: 640px) { .grid,.two { grid-template-columns:1fr; } .hero { flex-direction:column; align-items:flex-start; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div>
        <h1>be_ads Control Panel</h1>
        <p>Generated at {{ .GeneratedAt }}</p>
      </div>
      <div><span class="badge">collector / transformer / shard ownership</span></div>
    </div>

    <div class="grid">
      <div class="card"><div class="label">Worker Leases</div><div class="metric">{{ len .WorkerLeases }}</div></div>
      <div class="card"><div class="label">Shard Assignments</div><div class="metric">{{ len .ShardAssignments }}</div></div>
      <div class="card"><div class="label">Raw Records</div><div class="metric">{{ .RawRecordCount }}</div></div>
      <div class="card"><div class="label">DLQ Messages</div><div class="metric">{{ .DeadLetterCount }}</div></div>
    </div>

    <div class="grid">
      <div class="card"><div class="label">Outbox Pending</div><div class="metric">{{ .OutboxPending }}</div></div>
      <div class="card"><div class="label">Outbox Published</div><div class="metric">{{ .OutboxPublished }}</div></div>
      <div class="card"><div class="label">Account Snapshots</div><div class="metric">{{ len .Snapshots }}</div></div>
      <div class="card"><div class="label">Pipelines</div><div class="metric">2</div></div>
    </div>

    <div class="actions">
      <div class="card">
        <h2>Replay DLQ</h2>
        <div class="field"><label>Kind</label><select id="replay-kind"><option value="raw_event">raw_event</option><option value="collect_job">collect_job</option></select></div>
        <div class="field"><label>Platform</label><select id="replay-platform"><option value="">all</option><option value="facebook">facebook</option><option value="google_ads">google_ads</option><option value="tiktok_ads">tiktok_ads</option></select></div>
        <div class="field"><label>Limit</label><input id="replay-limit" type="number" value="10"></div>
        <button onclick="replayDlq()">Replay</button>
        <div id="replay-result" class="tiny"></div>
      </div>
      <div class="card">
        <h2>Dispatch Backfill</h2>
        <div class="field"><label>Platform</label><select id="backfill-platform"><option value="">all</option><option value="facebook">facebook</option><option value="google_ads">google_ads</option><option value="tiktok_ads">tiktok_ads</option></select></div>
        <div class="field"><label>Account ID</label><input id="backfill-account" type="text" placeholder="248-390-1805"></div>
        <div class="field"><label>Object Type</label><select id="backfill-object"><option value="">all</option><option value="campaign">campaign</option><option value="ad_group">ad_group</option><option value="ad">ad</option><option value="insight">insight</option></select></div>
        <button class="secondary" onclick="dispatchBackfill()">Backfill</button>
        <div id="backfill-result" class="tiny"></div>
      </div>
    </div>

    <div class="two">
      <div class="card panel">
        <h2>Worker Leases</h2>
        <table>
          <thead><tr><th>Role</th><th>Worker</th><th>Platforms</th><th>Capacity</th><th>Expires</th></tr></thead>
          <tbody>
            {{ range .WorkerLeases }}
            <tr>
              <td><span class="badge">{{ .WorkerRole }}</span></td>
              <td>{{ .WorkerID }}</td>
              <td>{{ if .PlatformScope }}{{ .PlatformScope }}{{ else }}all{{ end }}</td>
              <td>{{ .Capacity }}</td>
              <td>{{ .ExpiresAt }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="5">No active worker leases</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card panel">
        <h2>Shard Assignments</h2>
        <table>
          <thead><tr><th>Role</th><th>Platform</th><th>Shard</th><th>Worker</th></tr></thead>
          <tbody>
            {{ range .ShardAssignments }}
            <tr>
              <td>{{ .WorkerRole }}</td>
              <td>{{ .Platform }}</td>
              <td>{{ .ShardID }}</td>
              <td>{{ .WorkerID }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="4">No shard assignments</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>
    </div>

    <div class="card panel">
      <h2>Account Snapshots</h2>
      <table>
        <thead><tr><th>Platform</th><th>Account</th><th>Source</th><th>Object</th><th>Campaigns</th><th>Ad Groups</th><th>Ads</th><th>Insights</th></tr></thead>
        <tbody>
          {{ range .Snapshots }}
          <tr>
            <td>{{ .Platform }}</td>
            <td>{{ .AccountID }}</td>
            <td>{{ .LastSourceMode }}</td>
            <td>{{ .LastObjectType }}</td>
            <td>{{ .CampaignCount }}</td>
            <td>{{ .AdGroupCount }}</td>
            <td>{{ .AdCount }}</td>
            <td>{{ .InsightCount }}</td>
          </tr>
          {{ else }}
          <tr><td colspan="8">No snapshots</td></tr>
          {{ end }}
        </tbody>
      </table>
    </div>
  </div>
  <script>
    async function fetchJSON(url, options) {
      const res = await fetch(url, options);
      if (!res.ok) throw new Error(await res.text());
      return await res.json();
    }
    async function replayDlq() {
      const payload = {
        kind: document.getElementById('replay-kind').value,
        platform: document.getElementById('replay-platform').value,
        limit: Number(document.getElementById('replay-limit').value || '10')
      };
      const result = await fetchJSON('/api/control/dlq/replay', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
      document.getElementById('replay-result').textContent = 'replayed ' + result.Accepted + ' item(s)';
      setTimeout(() => location.reload(), 800);
    }
    async function dispatchBackfill() {
      const payload = {
        platform: document.getElementById('backfill-platform').value,
        account_id: document.getElementById('backfill-account').value,
        object_type: document.getElementById('backfill-object').value
      };
      const result = await fetchJSON('/api/control/backfill', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
      document.getElementById('backfill-result').textContent = 'dispatched ' + result.Accepted + ' job(s)';
      setTimeout(() => location.reload(), 800);
    }
    setInterval(() => location.reload(), 10000);
  </script>
</body>
</html>`))

var biPanelTemplate = template.Must(template.New("bi-panel").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>be_ads BI Panel</title>
  <style>
    :root { --bg:#f6f1e8; --card:#fffdfa; --line:#ddd4c6; --ink:#223127; --muted:#68756d; --accent:#146c5a; --accent-soft:#e0efe9; --warn:#c46d37; --warn-soft:#f8e7dc; }
    * { box-sizing:border-box; }
    body { margin:0; background:linear-gradient(180deg,#efe5d2 0%,#f7f4ed 100%); color:var(--ink); font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "PingFang SC", "Helvetica Neue", sans-serif; }
    .wrap { max-width:1280px; margin:0 auto; padding:28px 18px 48px; }
    .hero { display:flex; justify-content:space-between; align-items:flex-end; gap:16px; margin-bottom:18px; }
    .hero h1 { margin:0; font-size:34px; letter-spacing:-0.03em; }
    .hero p { margin:8px 0 0; color:var(--muted); }
    .nav { display:flex; gap:10px; flex-wrap:wrap; }
    .nav a { color:var(--accent); text-decoration:none; background:var(--accent-soft); padding:10px 14px; border-radius:999px; font-weight:600; }
    .filters { display:grid; grid-template-columns:repeat(10,minmax(0,1fr)); gap:12px; background:var(--card); border:1px solid var(--line); border-radius:20px; padding:16px; box-shadow:0 12px 30px rgba(34,49,39,0.06); margin-bottom:18px; }
    .field { display:flex; flex-direction:column; gap:6px; }
    .field label { font-size:12px; color:var(--muted); text-transform:uppercase; letter-spacing:0.06em; }
    input, select, button { font:inherit; border-radius:12px; border:1px solid var(--line); padding:10px 12px; background:white; }
    button { background:var(--accent); color:white; border:none; cursor:pointer; font-weight:600; }
    .metrics { display:grid; grid-template-columns:repeat(10,minmax(0,1fr)); gap:14px; margin-bottom:18px; }
    .charts { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:14px; margin-bottom:18px; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:20px; padding:18px; box-shadow:0 12px 30px rgba(34,49,39,0.06); }
    .label { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:0.08em; }
    .metric { margin-top:8px; font-size:32px; font-weight:700; }
    .metric.good { color:var(--accent); }
    .metric.warn { color:var(--warn); }
    .panels { display:grid; grid-template-columns:1fr; gap:16px; }
    .panel-head { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:10px; }
    .panel-head h2 { margin:0; font-size:18px; }
    .panel-head span { font-size:12px; color:var(--muted); }
    table { width:100%; border-collapse:collapse; font-size:14px; }
    th, td { text-align:left; padding:10px 8px; border-bottom:1px solid #ece4d8; vertical-align:top; }
    th { color:var(--muted); font-weight:600; font-size:12px; text-transform:uppercase; letter-spacing:0.06em; }
    .badge { display:inline-block; padding:4px 8px; border-radius:999px; background:var(--accent-soft); color:var(--accent); font-size:12px; font-weight:700; }
    .badge.warn { background:var(--warn-soft); color:var(--warn); }
    .muted { color:var(--muted); }
    .trend-svg { width:100%; height:96px; display:block; margin-top:14px; }
    .trend-svg rect { fill:#2e8973; opacity:0.92; }
    .chart-head { display:flex; justify-content:space-between; align-items:center; gap:12px; }
    .chart-head h2 { margin:0; font-size:18px; }
    .empty-chart { margin-top:14px; min-height:96px; display:flex; align-items:center; justify-content:center; border-radius:16px; background:#f5f1e7; color:var(--muted); }
    @media (max-width: 1080px) { .filters, .metrics, .charts { grid-template-columns:repeat(2,minmax(0,1fr)); } }
    @media (max-width: 640px) { .hero { flex-direction:column; align-items:flex-start; } .filters, .metrics, .charts { grid-template-columns:1fr; } .wrap { padding:20px 14px 40px; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div>
        <h1>be_ads BI Panel</h1>
        <p>简单业务面板，直接看账户快照、Campaign 和 Insight 汇总。生成时间 {{ .GeneratedAt }}</p>
      </div>
      <div class="nav">
        <a href="/bi">BI Panel</a>
        <a href="/">Control Panel</a>
      </div>
    </div>

    <form class="filters" method="get" action="/bi">
      <div class="field">
        <label>Platform</label>
        <select name="platform">
          <option value="" {{ if eq .Platform "" }}selected{{ end }}>all</option>
          <option value="facebook" {{ if eq .Platform "facebook" }}selected{{ end }}>facebook</option>
          <option value="google_ads" {{ if eq .Platform "google_ads" }}selected{{ end }}>google_ads</option>
          <option value="tiktok_ads" {{ if eq .Platform "tiktok_ads" }}selected{{ end }}>tiktok_ads</option>
        </select>
      </div>
      <div class="field">
        <label>Account ID</label>
        <input type="text" name="account_id" value="{{ .AccountID }}" placeholder="248-390-1805">
      </div>
      <div class="field">
        <label>Date From</label>
        <input type="date" name="date_from" value="{{ .DateFrom }}">
      </div>
      <div class="field">
        <label>Date To</label>
        <input type="date" name="date_to" value="{{ .DateTo }}">
      </div>
      <div class="field">
        <label>Entity Level</label>
        <select name="entity_level">
          <option value="" {{ if eq .EntityLevel "" }}selected{{ end }}>all</option>
          <option value="campaign" {{ if eq .EntityLevel "campaign" }}selected{{ end }}>campaign</option>
          <option value="ad_group" {{ if eq .EntityLevel "ad_group" }}selected{{ end }}>ad_group</option>
          <option value="ad" {{ if eq .EntityLevel "ad" }}selected{{ end }}>ad</option>
        </select>
      </div>
      <div class="field">
        <label>Device</label>
        <input type="text" name="device" value="{{ .Device }}" placeholder="MOBILE">
      </div>
      <div class="field">
        <label>Network</label>
        <input type="text" name="network" value="{{ .Network }}" placeholder="SEARCH">
      </div>
      <div class="field">
        <label>Detail Limit</label>
        <input type="number" name="detail_limit" value="{{ .DetailLimit }}">
      </div>
      <div class="field">
        <label>Match Type</label>
        <input type="text" name="match_type" value="{{ .MatchType }}" placeholder="EXACT">
      </div>
      <div class="field">
        <label>Search Term</label>
        <input type="text" name="search_term" value="{{ .SearchTerm }}" placeholder="brand">
      </div>
      <div class="field">
        <label>Search Limit</label>
        <input type="number" name="search_term_limit" value="{{ .SearchTermLimit }}">
      </div>
      <div class="field">
        <label>Action</label>
        <button type="submit">刷新面板</button>
      </div>
    </form>

    <div class="metrics">
      <div class="card"><div class="label">Snapshots</div><div class="metric good">{{ .SnapshotCount }}</div></div>
      <div class="card"><div class="label">Campaigns</div><div class="metric good">{{ .CampaignCount }}</div></div>
      <div class="card"><div class="label">Insight Rows</div><div class="metric">{{ .InsightRowCount }}</div></div>
      <div class="card"><div class="label">Detail Rows</div><div class="metric">{{ .InsightDetailRowCount }}</div></div>
      <div class="card"><div class="label">Diag Rows</div><div class="metric">{{ .CampaignDiagnosticRowCount }}</div></div>
      <div class="card"><div class="label">Impressions</div><div class="metric">{{ .TotalImpressions }}</div></div>
      <div class="card"><div class="label">Clicks / Spend</div><div class="metric {{ if eq .InsightRowCount 0 }}warn{{ else }}good{{ end }}">{{ .TotalClicks }} / {{ .TotalSpend }}</div></div>
      <div class="card"><div class="label">Conversions</div><div class="metric good">{{ .TotalConversions }}</div></div>
      <div class="card"><div class="label">Conv Value</div><div class="metric good">{{ .TotalConversionsValue }}</div></div>
      <div class="card"><div class="label">CPA / All CPA</div><div class="metric {{ if eq .InsightRowCount 0 }}warn{{ else }}good{{ end }}">{{ .AvgCostPerConversion }} / {{ .AvgCostPerAllConversions }}</div></div>
      <div class="card"><div class="label">Search IS</div><div class="metric good">{{ .AvgSearchImpressionShare }}</div></div>
      <div class="card"><div class="label">Top IS</div><div class="metric good">{{ .AvgSearchTopImpressionShare }}</div></div>
      <div class="card"><div class="label">Abs Top IS</div><div class="metric good">{{ .AvgSearchAbsoluteTopImpressionShare }}</div></div>
      <div class="card"><div class="label">Search Terms</div><div class="metric">{{ .SearchTermRowCount }}</div></div>
      <div class="card"><div class="label">Search Clicks</div><div class="metric good">{{ .SearchTermClicks }}</div></div>
      <div class="card"><div class="label">Search Spend</div><div class="metric">{{ .SearchTermSpend }}</div></div>
      <div class="card"><div class="label">Search Conv</div><div class="metric good">{{ .SearchTermConversions }}</div></div>
      <div class="card"><div class="label">Search Value</div><div class="metric good">{{ .SearchTermConversionsValue }}</div></div>
    </div>

    <div class="charts">
      <div class="card">
        <div class="chart-head">
          <h2>Impressions Trend</h2>
          <span class="muted">按日期聚合</span>
        </div>
        {{ .ImpressionsSVG }}
      </div>
      <div class="card">
        <div class="chart-head">
          <h2>Clicks Trend</h2>
          <span class="muted">按日期聚合</span>
        </div>
        {{ .ClicksSVG }}
      </div>
      <div class="card">
        <div class="chart-head">
          <h2>Spend Trend</h2>
          <span class="muted">按日期聚合</span>
        </div>
        {{ .SpendSVG }}
      </div>
    </div>

    <div class="panels">
      <div class="card">
        <div class="panel-head">
          <h2>Platform Summary</h2>
          <span>{{ len .PlatformSummary }} platform group(s)</span>
        </div>
        <table>
          <thead><tr><th>Platform</th><th>Accounts</th><th>Campaigns</th><th>Ad Groups</th><th>Ads</th><th>Insights</th><th>Impressions</th><th>Clicks</th><th>Spend</th></tr></thead>
          <tbody>
            {{ range .PlatformSummary }}
            <tr>
              <td><span class="badge">{{ .Platform }}</span></td>
              <td>{{ .AccountCount }}</td>
              <td>{{ .CampaignCount }}</td>
              <td>{{ .AdGroupCount }}</td>
              <td>{{ .AdCount }}</td>
              <td>{{ .InsightCount }}</td>
              <td>{{ .TotalImpressions }}</td>
              <td>{{ .TotalClicks }}</td>
              <td>{{ .TotalSpend }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="9" class="muted">No grouped platform data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div class="panel-head">
          <h2>Account Summary</h2>
          <span>{{ len .AccountSummary }} account group(s)</span>
        </div>
        <table>
          <thead><tr><th>Platform</th><th>Account</th><th>Name</th><th>Source</th><th>Campaigns</th><th>Ad Groups</th><th>Ads</th><th>Insights</th><th>Impressions</th><th>Clicks</th><th>Spend</th></tr></thead>
          <tbody>
            {{ range .AccountSummary }}
            <tr>
              <td>{{ .Platform }}</td>
              <td>{{ .AccountID }}</td>
              <td>{{ if .AccountName }}{{ .AccountName }}{{ else }}-{{ end }}</td>
              <td>{{ .SourceMode }}</td>
              <td>{{ .CampaignCount }}</td>
              <td>{{ .AdGroupCount }}</td>
              <td>{{ .AdCount }}</td>
              <td>{{ .InsightCount }}</td>
              <td>{{ .TotalImpressions }}</td>
              <td>{{ .TotalClicks }}</td>
              <td>{{ .TotalSpend }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="11" class="muted">No grouped account data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div class="panel-head">
          <h2>Account Snapshots</h2>
          <span>{{ .SnapshotCount }} account(s)</span>
        </div>
        <table>
          <thead><tr><th>Platform</th><th>Account</th><th>Name</th><th>Source</th><th>Last Object</th><th>Campaigns</th><th>Ad Groups</th><th>Ads</th><th>Insights</th></tr></thead>
          <tbody>
            {{ range .Snapshots }}
            <tr>
              <td><span class="badge">{{ .Platform }}</span></td>
              <td>{{ .AccountID }}</td>
              <td>{{ .AccountName }}</td>
              <td>{{ .LastSourceMode }}</td>
              <td>{{ .LastObjectType }}</td>
              <td>{{ .CampaignCount }}</td>
              <td>{{ .AdGroupCount }}</td>
              <td>{{ .AdCount }}</td>
              <td>{{ .InsightCount }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="9" class="muted">No snapshot data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div class="panel-head">
          <h2>Campaigns</h2>
          <span>{{ .CampaignCount }} row(s)</span>
        </div>
        <table>
          <thead><tr><th>Platform</th><th>Account</th><th>Campaign</th><th>Status</th><th>Objective</th><th>Bidding</th><th>Budget</th><th>Start</th><th>End</th><th>Updated</th><th>Ingested</th></tr></thead>
          <tbody>
            {{ range .Campaigns }}
            <tr>
              <td>{{ .Platform }}</td>
              <td>{{ .AccountID }}</td>
              <td>{{ .CampaignName }}</td>
              <td><span class="badge {{ if ne .Status "ACTIVE" }}warn{{ end }}">{{ .Status }}</span></td>
              <td>{{ .Objective }}</td>
              <td>{{ if .BiddingStrategy }}{{ .BiddingStrategy }}{{ else if .BuyingType }}{{ .BuyingType }}{{ else }}-{{ end }}</td>
              <td>{{ if .DailyBudget }}{{ .DailyBudget }}{{ else }}{{ .LifetimeBudget }}{{ end }} {{ .Currency }}</td>
              <td>{{ if .StartTime.IsZero }}-{{ else }}{{ .StartTime.Format "2006-01-02" }}{{ end }}</td>
              <td>{{ if .EndTime.IsZero }}-{{ else }}{{ .EndTime.Format "2006-01-02" }}{{ end }}</td>
              <td>{{ if .SourceUpdatedAt.IsZero }}-{{ else }}{{ .SourceUpdatedAt.Format "2006-01-02 15:04" }}{{ end }}</td>
              <td>{{ .IngestedAt }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="11" class="muted">No campaign data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div class="panel-head">
          <h2>Campaign Diagnostics</h2>
          <span>{{ .CampaignDiagnosticRowCount }} row(s)</span>
        </div>
        <table>
          <thead><tr><th>Date</th><th>Platform</th><th>Account</th><th>Campaign ID</th><th>Search IS</th><th>Top IS</th><th>Abs Top IS</th></tr></thead>
          <tbody>
            {{ range .CampaignDiagnostics }}
            <tr>
              <td>{{ .StatDate.Format "2006-01-02" }}</td>
              <td>{{ .Platform }}</td>
              <td>{{ .PlatformAccountID }}</td>
              <td>{{ .PlatformCampaignID }}</td>
              <td>{{ .SearchImpressionShare }}</td>
              <td>{{ .SearchTopImpressionShare }}</td>
              <td>{{ .SearchAbsoluteTopImpressionShare }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="7" class="muted">No campaign diagnostic data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div class="panel-head">
          <h2>Search Term Diagnostics</h2>
          <span>{{ .SearchTermRowCount }} row(s)</span>
        </div>
        <table>
          <thead><tr><th>Date</th><th>Platform</th><th>Account</th><th>Campaign</th><th>Ad Group</th><th>Search Term</th><th>Match</th><th>Impr</th><th>Clicks</th><th>Spend</th><th>Conv</th><th>Value</th></tr></thead>
          <tbody>
            {{ range .SearchTerms }}
            <tr>
              <td>{{ .StatDate.Format "2006-01-02" }}</td>
              <td>{{ .Platform }}</td>
              <td>{{ .PlatformAccountID }}</td>
              <td>{{ .PlatformCampaignID }}</td>
              <td>{{ if .PlatformAdGroupID }}{{ .PlatformAdGroupID }}{{ else }}-{{ end }}</td>
              <td>{{ .SearchTerm }}</td>
              <td>{{ if .SearchTermMatchType }}{{ .SearchTermMatchType }}{{ else }}-{{ end }}</td>
              <td>{{ .Impressions }}</td>
              <td>{{ .Clicks }}</td>
              <td>{{ .Spend }}</td>
              <td>{{ .Conversions }}</td>
              <td>{{ .ConversionsValue }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="12" class="muted">No search term data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div class="panel-head">
          <h2>Insight Summary</h2>
          <span>{{ .InsightRowCount }} row(s)</span>
        </div>
        <table>
          <thead><tr><th>Platform</th><th>Platform Account</th><th>Date</th><th>Impressions</th><th>Clicks</th><th>Spend</th><th>Conversions</th><th>All Conv</th><th>Conv Value</th><th>CPA</th><th>All CPA</th><th>Reach</th></tr></thead>
          <tbody>
            {{ range .Insights }}
            <tr>
              <td>{{ .Platform }}</td>
              <td>{{ .PlatformAccountID }}</td>
              <td>{{ .StatDate.Format "2006-01-02" }}</td>
              <td>{{ .Impressions }}</td>
              <td>{{ .Clicks }}</td>
              <td>{{ .Spend }}</td>
              <td>{{ .Conversions }}</td>
              <td>{{ .AllConversions }}</td>
              <td>{{ .ConversionsValue }}</td>
              <td>{{ .CostPerConversion }}</td>
              <td>{{ .CostPerAllConversions }}</td>
              <td>{{ .Reach }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="12" class="muted">No insight data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div class="panel-head">
          <h2>Insight Detail Drilldown</h2>
          <span>{{ .InsightDetailRowCount }} row(s)</span>
        </div>
        <table>
          <thead><tr><th>Date</th><th>Platform</th><th>Account</th><th>Level</th><th>Entity ID</th><th>Ad Group</th><th>Ad</th><th>Device</th><th>Network</th><th>Impr</th><th>Clicks</th><th>Spend</th><th>Conv</th><th>All Conv</th><th>Value</th><th>CPA</th></tr></thead>
          <tbody>
            {{ range .InsightDetails }}
            <tr>
              <td>{{ .StatDate.Format "2006-01-02" }}</td>
              <td>{{ .Platform }}</td>
              <td>{{ .PlatformAccountID }}</td>
              <td><span class="badge">{{ .EntityLevel }}</span></td>
              <td>{{ .EntityID }}</td>
              <td>{{ if .PlatformAdGroupID }}{{ .PlatformAdGroupID }}{{ else }}-{{ end }}</td>
              <td>{{ if .PlatformAdID }}{{ .PlatformAdID }}{{ else }}-{{ end }}</td>
              <td>{{ if .Device }}{{ .Device }}{{ else }}-{{ end }}</td>
              <td>{{ if .Network }}{{ .Network }}{{ else }}-{{ end }}</td>
              <td>{{ .Impressions }}</td>
              <td>{{ .Clicks }}</td>
              <td>{{ .Spend }}</td>
              <td>{{ .Conversions }}</td>
              <td>{{ .AllConversions }}</td>
              <td>{{ .ConversionsValue }}</td>
              <td>{{ .CostPerConversion }}</td>
            </tr>
            {{ else }}
            <tr><td colspan="16" class="muted">No insight detail data</td></tr>
            {{ end }}
          </tbody>
        </table>
      </div>
    </div>
  </div>
  <script>
    setTimeout(() => location.reload(), 30000);
  </script>
</body>
</html>`))

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
