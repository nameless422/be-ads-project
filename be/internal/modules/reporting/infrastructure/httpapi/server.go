package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	biquerydomain "be_ads_project/internal/modules/reporting/domain"
)

type SnapshotReader interface {
	ListAccountSnapshots(context.Context) ([]biquerydomain.AccountSnapshot, error)
	ListCampaigns(context.Context, biquerydomain.CampaignFilter) ([]biquerydomain.CampaignView, error)
	ListGameKPIs(context.Context, biquerydomain.GameKPIQueryFilter) ([]biquerydomain.GameKPIRecord, error)
	UpsertGameKPIs(context.Context, []biquerydomain.GameKPIRecord) error
	Ping(context.Context) error
}

type InsightReader interface {
	QueryInsightSummary(context.Context, biquerydomain.InsightSummaryFilter) ([]biquerydomain.InsightSummaryRow, error)
	QueryInsightDetails(context.Context, biquerydomain.InsightDetailFilter) ([]biquerydomain.InsightDetailRow, error)
	QueryCampaignDiagnostics(context.Context, biquerydomain.CampaignDiagnosticFilter) ([]biquerydomain.CampaignDiagnosticRow, error)
	QuerySearchTerms(context.Context, biquerydomain.SearchTermFilter) ([]biquerydomain.SearchTermRow, error)
	QueryUAReportRows(context.Context, biquerydomain.UAReportFilter) ([]biquerydomain.UAAdReportRow, error)
	Ping(context.Context) error
}

type UAReportReader interface {
	QueryReport(context.Context, biquerydomain.UAReportFilter) ([]biquerydomain.UAReportRow, error)
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

type LocalStackOperator interface {
	Status(context.Context) (*biquerydomain.LocalStackStatus, error)
	Start(context.Context) (*biquerydomain.LocalCommandResult, error)
	Stop(context.Context) (*biquerydomain.LocalCommandResult, error)
	Restart(context.Context) (*biquerydomain.LocalCommandResult, error)
	Verify(context.Context) (*biquerydomain.LocalCommandResult, error)
	StartInfra(context.Context) (*biquerydomain.LocalCommandResult, error)
	StopInfra(context.Context) (*biquerydomain.LocalCommandResult, error)
	StartWorkers(context.Context) (*biquerydomain.LocalCommandResult, error)
	StopWorkers(context.Context) (*biquerydomain.LocalCommandResult, error)
	RestartCollector(context.Context) (*biquerydomain.LocalCommandResult, error)
	AddWorker(context.Context, string) (*biquerydomain.LocalCommandResult, error)
	RemoveWorker(context.Context, string) (*biquerydomain.LocalCommandResult, error)
}

type Server struct {
	logger       *log.Logger
	snapshotRepo SnapshotReader
	insightRepo  InsightReader
	uaReport     UAReportReader
	controlRepo  ControlReader
	dlqReader    DeadLetterReader
	actions      ControlActionHandler
	localOps     LocalStackOperator
}

func NewServer(snapshotRepo SnapshotReader, insightRepo InsightReader, uaReport UAReportReader, controlRepo ControlReader, dlqReader DeadLetterReader, actions ControlActionHandler, localOps LocalStackOperator, logger *log.Logger) *Server {
	return &Server{
		logger:       logger,
		snapshotRepo: snapshotRepo,
		insightRepo:  insightRepo,
		uaReport:     uaReport,
		controlRepo:  controlRepo,
		dlqReader:    dlqReader,
		actions:      actions,
		localOps:     localOps,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/bi", s.handleBIApp)
	mux.HandleFunc("/bi/", s.handleBIApp)
	mux.HandleFunc("/api/bi/snapshots", s.handleSnapshots)
	mux.HandleFunc("/api/bi/campaigns", s.handleCampaigns)
	mux.HandleFunc("/api/bi/insights/summary", s.handleInsightSummary)
	mux.HandleFunc("/api/bi/insights/detail", s.handleInsightDetail)
	mux.HandleFunc("/api/bi/ua-report", s.handleUAReport)
	mux.HandleFunc("/api/bi/ua-fields", s.handleUAFields)
	mux.HandleFunc("/api/bi/game-kpis", s.handleGameKPIs)
	mux.HandleFunc("/api/bi/campaign-diagnostics", s.handleCampaignDiagnostics)
	mux.HandleFunc("/api/bi/search-terms", s.handleSearchTerms)
	mux.HandleFunc("/api/control/overview", s.handleControlOverview)
	mux.HandleFunc("/api/control/leases", s.handleControlLeases)
	mux.HandleFunc("/api/control/shards", s.handleControlShards)
	mux.HandleFunc("/api/control/dlq", s.handleControlDLQ)
	mux.HandleFunc("/api/control/dlq/replay", s.handleReplayDLQ)
	mux.HandleFunc("/api/control/backfill", s.handleBackfill)
	mux.HandleFunc("/api/control/local-stack", s.handleLocalStackStatus)
	mux.HandleFunc("/api/control/local-stack/start", s.handleLocalStackStart)
	mux.HandleFunc("/api/control/local-stack/stop", s.handleLocalStackStop)
	mux.HandleFunc("/api/control/local-stack/restart", s.handleLocalStackRestart)
	mux.HandleFunc("/api/control/local-stack/verify", s.handleLocalStackVerify)
	mux.HandleFunc("/api/control/local-stack/start-infra", s.handleLocalStackStartInfra)
	mux.HandleFunc("/api/control/local-stack/stop-infra", s.handleLocalStackStopInfra)
	mux.HandleFunc("/api/control/local-stack/start-workers", s.handleLocalStackStartWorkers)
	mux.HandleFunc("/api/control/local-stack/stop-workers", s.handleLocalStackStopWorkers)
	mux.HandleFunc("/api/control/local-stack/restart-collector", s.handleLocalStackRestartCollector)
	mux.HandleFunc("/api/control/local-stack/add-worker", s.handleLocalStackAddWorker)
	mux.HandleFunc("/api/control/local-stack/remove-worker", s.handleLocalStackRemoveWorker)
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

func (s *Server) handleUAReport(w http.ResponseWriter, r *http.Request) {
	filter, err := parseUAReportFilter(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	items, err := s.uaReport.QueryReport(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleUAFields(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": uaFieldCatalog()})
}

func (s *Server) handleGameKPIs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter, err := parseGameKPIQueryFilter(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		items, err := s.snapshotRepo.ListGameKPIs(r.Context(), filter)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		var req biquerydomain.GameKPIUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if len(req.Items) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items is required"})
			return
		}
		if err := s.snapshotRepo.UpsertGameKPIs(r.Context(), req.Items); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"accepted": len(req.Items)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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

func (s *Server) handleLocalStackStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !isLocalRequest(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "local requests only"})
		return
	}
	if s.localOps == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "local stack controls disabled"})
		return
	}
	status, err := s.localOps.Status(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleLocalStackStart(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "start")
}

func (s *Server) handleLocalStackStop(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "stop")
}

func (s *Server) handleLocalStackRestart(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "restart")
}

func (s *Server) handleLocalStackVerify(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "verify")
}

func (s *Server) handleLocalStackStartInfra(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "start-infra")
}

func (s *Server) handleLocalStackStopInfra(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "stop-infra")
}

func (s *Server) handleLocalStackStartWorkers(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "start-workers")
}

func (s *Server) handleLocalStackStopWorkers(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "stop-workers")
}

func (s *Server) handleLocalStackRestartCollector(w http.ResponseWriter, r *http.Request) {
	s.handleLocalStackAction(w, r, "restart-collector")
}

func (s *Server) handleLocalStackAddWorker(w http.ResponseWriter, r *http.Request) {
	s.handleLocalWorkerScale(w, r, "add")
}

func (s *Server) handleLocalStackRemoveWorker(w http.ResponseWriter, r *http.Request) {
	s.handleLocalWorkerScale(w, r, "remove")
}

func (s *Server) handleLocalStackAction(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !isLocalRequest(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "local requests only"})
		return
	}
	if s.localOps == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "local stack controls disabled"})
		return
	}
	var (
		result *biquerydomain.LocalCommandResult
		err    error
	)
	switch action {
	case "start":
		result, err = s.localOps.Start(r.Context())
	case "stop":
		result, err = s.localOps.Stop(r.Context())
	case "restart":
		result, err = s.localOps.Restart(r.Context())
	case "verify":
		result, err = s.localOps.Verify(r.Context())
	case "start-infra":
		result, err = s.localOps.StartInfra(r.Context())
	case "stop-infra":
		result, err = s.localOps.StopInfra(r.Context())
	case "start-workers":
		result, err = s.localOps.StartWorkers(r.Context())
	case "stop-workers":
		result, err = s.localOps.StopWorkers(r.Context())
	case "restart-collector":
		result, err = s.localOps.RestartCollector(r.Context())
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action"})
		return
	}
	if err != nil {
		status := http.StatusInternalServerError
		if result != nil {
			writeJSON(w, status, result)
			return
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleLocalWorkerScale(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !isLocalRequest(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "local requests only"})
		return
	}
	if s.localOps == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "local stack controls disabled"})
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.Role = strings.TrimSpace(strings.ToLower(req.Role))
	if req.Role != "collector" && req.Role != "transformer" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role must be collector or transformer"})
		return
	}
	var (
		result *biquerydomain.LocalCommandResult
		err    error
	)
	if action == "add" {
		result, err = s.localOps.AddWorker(r.Context(), req.Role)
	} else {
		result, err = s.localOps.RemoveWorker(r.Context(), req.Role)
	}
	if err != nil {
		if result != nil {
			writeJSON(w, http.StatusInternalServerError, result)
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

const biFrontendDistDir = "../fe/dist"

func (s *Server) handleBIApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/bi/assets/") {
		rel := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/bi/"))
		if rel == "." || strings.HasPrefix(rel, "..") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(biFrontendDistDir, rel))
		return
	}

	indexPath := filepath.Join(biFrontendDistDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		http.Error(w, "BI React frontend is not built; run `npm --prefix ../fe install && npm --prefix ../fe run build` from be/", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, indexPath)
}

func (s *Server) handleBIRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/bi" {
		http.NotFound(w, r)
		return
	}
	target := "/bi/overview"
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) handleBIPage(w http.ResponseWriter, r *http.Request) {
	pageKey, ok := biPageKeyFromPath(r.URL.Path)
	if !ok {
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
	uaFilter := biquerydomain.UAReportFilter{
		Platform:  filter.Platform,
		AccountID: filter.AccountID,
		Limit:     200,
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
		uaFilter.DateFrom = parsed
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
		uaFilter.DateTo = parsed
	}
	detailFilter.EntityLevel = strings.TrimSpace(r.URL.Query().Get("entity_level"))
	detailFilter.Device = strings.TrimSpace(r.URL.Query().Get("device"))
	detailFilter.Network = strings.TrimSpace(r.URL.Query().Get("network"))
	uaFilter.EntityLevel = detailFilter.EntityLevel
	uaFilter.Device = detailFilter.Device
	uaFilter.Network = detailFilter.Network
	uaFilter.Country = strings.TrimSpace(r.URL.Query().Get("country"))
	uaFilter.OS = strings.TrimSpace(r.URL.Query().Get("os"))
	uaFilter.PlatformCampaignID = strings.TrimSpace(r.URL.Query().Get("platform_campaign_id"))
	uaFilter.PlatformAdGroupID = strings.TrimSpace(r.URL.Query().Get("platform_ad_group_id"))
	uaFilter.PlatformAdID = strings.TrimSpace(r.URL.Query().Get("platform_ad_id"))
	searchTermFilter.MatchType = strings.TrimSpace(r.URL.Query().Get("match_type"))
	searchTermFilter.SearchTermQuery = strings.TrimSpace(r.URL.Query().Get("search_term"))
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("detail_limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			http.Error(w, "invalid detail_limit", http.StatusBadRequest)
			return
		}
		detailFilter.Limit = limit
		uaFilter.Limit = limit
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
	uaReports, err := s.uaReport.QueryReport(r.Context(), uaFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	gameKPIs, err := s.snapshotRepo.ListGameKPIs(r.Context(), biquerydomain.GameKPIQueryFilter{
		Platform:           uaFilter.Platform,
		AccountID:          uaFilter.AccountID,
		DateFrom:           uaFilter.DateFrom,
		DateTo:             uaFilter.DateTo,
		Country:            uaFilter.Country,
		OS:                 uaFilter.OS,
		PlatformCampaignID: uaFilter.PlatformCampaignID,
		PlatformAdGroupID:  uaFilter.PlatformAdGroupID,
		PlatformAdID:       uaFilter.PlatformAdID,
		Limit:              uaFilter.Limit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page := buildBIPageView(filter, insightFilter, detailFilter, searchTermFilter, uaFilter, snapshots, campaigns, insights, insightDetails, campaignDiagnostics, searchTerms, uaReports, gameKPIs)
	title, description, _ := biPageMeta(pageKey)
	page.PageKey = pageKey
	page.PageTitle = title
	page.PageDescription = description
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = biPanelTemplate.Execute(w, page)
}

func biPageKeyFromPath(path string) (string, bool) {
	switch path {
	case "/bi/overview":
		return "overview", true
	case "/bi/breakdown":
		return "breakdown", true
	case "/bi/creatives":
		return "creatives", true
	case "/bi/quality":
		return "quality", true
	default:
		return "", false
	}
}

func biPageMeta(pageKey string) (title, description string, ok bool) {
	switch pageKey {
	case "overview":
		return "Overview", "一眼看今天的大盘、漏斗和回本表现。", true
	case "breakdown":
		return "Breakdown", "按平台、账户、campaign、ad group、ad 与搜索词快速定位问题维度。", true
	case "creatives":
		return "Creatives", "围绕素材、版位和创意相关字段做优化判断。", true
	case "quality":
		return "Quality", "重点看留存、付费、深度行为和可疑流量质量信号。", true
	default:
		return "", "", false
	}
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
	var localStack *biquerydomain.LocalStackStatus
	if s.localOps != nil {
		localStack, _ = s.localOps.Status(ctx)
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
		LocalStack:       localStack,
	}, nil
}

func isLocalRequest(r *http.Request) bool {
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return false
	}
	if strings.HasPrefix(host, "127.0.0.1:") || strings.HasPrefix(host, "[::1]:") || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return false
	}
	return false
}

func localStateClass(state string) string {
	normalized := strings.ToLower(strings.TrimSpace(state))
	switch normalized {
	case "running", "listening", "ok", "healthy":
		return ""
	case "stopped", "closed", "missing":
		return "dim"
	default:
		if strings.Contains(normalized, "warn") || strings.Contains(normalized, "stale") || strings.Contains(normalized, "error") || strings.Contains(normalized, "fail") {
			return "warn"
		}
		return "warn"
	}
}

type biPanelView struct {
	PageKey                             string
	PageTitle                           string
	PageDescription                     string
	GeneratedAt                         time.Time
	Platform                            string
	AccountID                           string
	DateFrom                            string
	DateTo                              string
	Country                             string
	OS                                  string
	PlatformCampaignID                  string
	PlatformAdGroupID                   string
	PlatformAdID                        string
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
	UAReportRowCount                    int
	GameKPIRowCount                     int
	UAAvailableFieldCount               int
	UAIntegrationReadyFieldCount        int
	UAPlannedFieldCount                 int
	TotalImpressions                    int64
	TotalClicks                         int64
	TotalReach                          int64
	TotalSpend                          string
	TotalConversions                    string
	TotalAllConversions                 string
	TotalConversionsValue               string
	AvgCostPerConversion                string
	AvgCostPerAllConversions            string
	AvgCTR                              string
	AvgCPC                              string
	AvgCPM                              string
	AvgFrequency                        string
	AvgSearchImpressionShare            string
	AvgSearchTopImpressionShare         string
	AvgSearchAbsoluteTopImpressionShare string
	SearchTermClicks                    int64
	SearchTermSpend                     string
	SearchTermConversions               string
	SearchTermConversionsValue          string
	UAInstalls                          int64
	UAActivations                       int64
	UARegistrations                     int64
	UAPurchasers                        int64
	UAPurchaseCount                     int64
	UATutorialCompletions               int64
	UARoleCreations                     int64
	UALevelXUsers                       int64
	UATotalRevenue                      string
	UARevenueD1                         string
	UARevenueD7                         string
	UARevenueD30                        string
	UAD1Retention                       string
	UAD3Retention                       string
	UAD7Retention                       string
	UAD30Retention                      string
	UAROAS                              string
	UACPI                               string
	UAROI                               string
	UAActivationRate                    string
	UARegistrationRate                  string
	UAPayerRate                         string
	UAARPU                              string
	UAARPPU                             string
	UALTVD7                             string
	UALTVD30                            string
	UALTVToCPIRatio                     string
	UAAvgOnlineDurationSeconds          int64
	UATaskCompletionRate                string
	UAHighValuePayerRatio               string
	UADataStatus                        string
	CreativeCount                       int
	CreativeTypeCount                   int
	PlacementCount                      int
	CountryCount                        int
	OSCount                             int
	ImpressionsSVG                      template.HTML
	ClicksSVG                           template.HTML
	SpendSVG                            template.HTML
	UAInstallsSVG                       template.HTML
	UARevenueSVG                        template.HTML
	PlatformSummary                     []biPlatformSummary
	AccountSummary                      []biAccountSummary
	Snapshots                           []biquerydomain.AccountSnapshot
	Campaigns                           []biquerydomain.CampaignView
	Insights                            []biquerydomain.InsightSummaryRow
	InsightDetails                      []biquerydomain.InsightDetailRow
	CampaignDiagnostics                 []biquerydomain.CampaignDiagnosticRow
	SearchTerms                         []biquerydomain.SearchTermRow
	UAReports                           []biquerydomain.UAReportRow
	GameKPIs                            []biquerydomain.GameKPIRecord
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
	uaFilter biquerydomain.UAReportFilter,
	snapshots []biquerydomain.AccountSnapshot,
	campaigns []biquerydomain.CampaignView,
	insights []biquerydomain.InsightSummaryRow,
	insightDetails []biquerydomain.InsightDetailRow,
	campaignDiagnostics []biquerydomain.CampaignDiagnosticRow,
	searchTerms []biquerydomain.SearchTermRow,
	uaReports []biquerydomain.UAReportRow,
	gameKPIs []biquerydomain.GameKPIRecord,
) biPanelView {
	var totalImpressions int64
	var totalClicks int64
	var totalReach int64
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
	var uaInstalls int64
	var uaActivations int64
	var uaRegistrations int64
	var uaPurchasers int64
	var uaPurchaseCount int64
	var uaTutorialCompletions int64
	var uaRoleCreations int64
	var uaLevelXUsers int64
	var uaTotalRevenue float64
	var uaRevenueD1 float64
	var uaRevenueD7 float64
	var uaRevenueD30 float64
	var uaRetentionD1 float64
	var uaRetentionD3 float64
	var uaRetentionD7 float64
	var uaRetentionD30 float64
	var uaROAS float64
	var uaCPI float64
	var uaROI float64
	var uaActivationRate float64
	var uaRegistrationRate float64
	var uaPayerRate float64
	var uaARPU float64
	var uaARPPU float64
	var uaLTVD7 float64
	var uaLTVD30 float64
	var uaLTVToCPIRatio float64
	var uaAvgOnlineDurationSeconds int64
	var uaTaskCompletionRate float64
	var uaHighValuePayerRatio float64
	creativeIDs := map[string]struct{}{}
	creativeTypes := map[string]struct{}{}
	placements := map[string]struct{}{}
	countries := map[string]struct{}{}
	operatingSystems := map[string]struct{}{}
	for _, item := range insights {
		totalImpressions += item.Impressions
		totalClicks += item.Clicks
		totalReach += item.Reach
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
	for _, item := range uaReports {
		uaInstalls += item.Installs
		uaActivations += item.Activations
		uaRegistrations += item.Registrations
		uaPurchasers += item.Purchasers
		uaPurchaseCount += item.PurchaseCount
		uaTutorialCompletions += item.TutorialCompletions
		uaRoleCreations += item.RoleCreations
		uaLevelXUsers += item.LevelXUsers
		uaTotalRevenue += parseMetricFloat(item.TotalRevenue)
		uaRevenueD1 += parseMetricFloat(item.RevenueD1)
		uaRevenueD7 += parseMetricFloat(item.RevenueD7)
		uaRevenueD30 += parseMetricFloat(item.RevenueD30)
		uaRetentionD1 += parseMetricFloat(item.RetentionD1)
		uaRetentionD3 += parseMetricFloat(item.RetentionD3)
		uaRetentionD7 += parseMetricFloat(item.RetentionD7)
		uaRetentionD30 += parseMetricFloat(item.RetentionD30)
		uaROAS += parseMetricFloat(item.ROAS)
		uaCPI += parseMetricFloat(item.CPI)
		uaROI += parseMetricFloat(item.ROI)
		uaActivationRate += parseMetricFloat(item.ActivationRate)
		uaRegistrationRate += parseMetricFloat(item.RegistrationRate)
		uaPayerRate += parseMetricFloat(item.PayerRate)
		uaARPU += parseMetricFloat(item.ARPU)
		uaARPPU += parseMetricFloat(item.ARPPU)
		uaLTVD7 += parseMetricFloat(item.LTVD7)
		uaLTVD30 += parseMetricFloat(item.LTVD30)
		uaLTVToCPIRatio += parseMetricFloat(item.LTVToCPIRatio)
		uaAvgOnlineDurationSeconds += item.AvgOnlineDurationSeconds
		uaTaskCompletionRate += parseMetricFloat(item.TaskCompletionRate)
		uaHighValuePayerRatio += parseMetricFloat(item.HighValuePayerRatio)
		if item.CreativeID != "" {
			creativeIDs[item.CreativeID] = struct{}{}
		}
		if item.CreativeType != "" {
			creativeTypes[item.CreativeType] = struct{}{}
		}
		if item.Placement != "" {
			placements[item.Placement] = struct{}{}
		}
		if item.Country != "" {
			countries[item.Country] = struct{}{}
		}
		if item.OS != "" {
			operatingSystems[item.OS] = struct{}{}
		}
	}
	for _, item := range gameKPIs {
		if item.CreativeID != "" {
			creativeIDs[item.CreativeID] = struct{}{}
		}
		if item.CreativeType != "" {
			creativeTypes[item.CreativeType] = struct{}{}
		}
		if item.Placement != "" {
			placements[item.Placement] = struct{}{}
		}
		if item.Country != "" {
			countries[item.Country] = struct{}{}
		}
		if item.OS != "" {
			operatingSystems[item.OS] = struct{}{}
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
	avgCTR := "0.00"
	if totalImpressions > 0 {
		avgCTR = formatMetricFloat(float64(totalClicks) / float64(totalImpressions))
	}
	avgCPC := "0.00"
	if totalClicks > 0 {
		avgCPC = formatMetricFloat(totalSpend / float64(totalClicks))
	}
	avgCPM := "0.00"
	if totalImpressions > 0 {
		avgCPM = formatMetricFloat(totalSpend * 1000 / float64(totalImpressions))
	}
	avgFrequency := "0.00"
	if totalReach > 0 {
		avgFrequency = formatMetricFloat(float64(totalImpressions) / float64(totalReach))
	}
	uaDataStatus := "广告侧可用，游戏内字段待接入"
	if len(gameKPIs) > 0 {
		uaDataStatus = "广告侧 + 游戏内 KPI 已合并"
	}
	availableFieldCount, integrationReadyFieldCount, plannedFieldCount := uaFieldStats()

	page := biPanelView{
		GeneratedAt:                         time.Now().UTC(),
		Platform:                            filter.Platform,
		AccountID:                           filter.AccountID,
		Country:                             uaFilter.Country,
		OS:                                  uaFilter.OS,
		PlatformCampaignID:                  uaFilter.PlatformCampaignID,
		PlatformAdGroupID:                   uaFilter.PlatformAdGroupID,
		PlatformAdID:                        uaFilter.PlatformAdID,
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
		UAReportRowCount:                    len(uaReports),
		GameKPIRowCount:                     len(gameKPIs),
		UAAvailableFieldCount:               availableFieldCount,
		UAIntegrationReadyFieldCount:        integrationReadyFieldCount,
		UAPlannedFieldCount:                 plannedFieldCount,
		TotalImpressions:                    totalImpressions,
		TotalClicks:                         totalClicks,
		TotalReach:                          totalReach,
		TotalSpend:                          strconv.FormatFloat(totalSpend, 'f', 2, 64),
		TotalConversions:                    strconv.FormatFloat(totalConversions, 'f', 2, 64),
		TotalAllConversions:                 strconv.FormatFloat(totalAllConversions, 'f', 2, 64),
		TotalConversionsValue:               strconv.FormatFloat(totalConversionsValue, 'f', 2, 64),
		AvgCostPerConversion:                avgCostPerConversion,
		AvgCostPerAllConversions:            avgCostPerAllConversions,
		AvgCTR:                              avgCTR,
		AvgCPC:                              avgCPC,
		AvgCPM:                              avgCPM,
		AvgFrequency:                        avgFrequency,
		AvgSearchImpressionShare:            avgSearchImpressionShare,
		AvgSearchTopImpressionShare:         avgSearchTopImpressionShare,
		AvgSearchAbsoluteTopImpressionShare: avgSearchAbsoluteTopImpressionShare,
		SearchTermClicks:                    searchTermClicks,
		SearchTermSpend:                     strconv.FormatFloat(searchTermSpend, 'f', 2, 64),
		SearchTermConversions:               strconv.FormatFloat(searchTermConversions, 'f', 2, 64),
		SearchTermConversionsValue:          strconv.FormatFloat(searchTermConversionsValue, 'f', 2, 64),
		UAInstalls:                          uaInstalls,
		UAActivations:                       uaActivations,
		UARegistrations:                     uaRegistrations,
		UAPurchasers:                        uaPurchasers,
		UAPurchaseCount:                     uaPurchaseCount,
		UATutorialCompletions:               uaTutorialCompletions,
		UARoleCreations:                     uaRoleCreations,
		UALevelXUsers:                       uaLevelXUsers,
		UATotalRevenue:                      formatMetricFloat(uaTotalRevenue),
		UARevenueD1:                         formatMetricFloat(uaRevenueD1),
		UARevenueD7:                         formatMetricFloat(uaRevenueD7),
		UARevenueD30:                        formatMetricFloat(uaRevenueD30),
		UAD1Retention:                       avgMetricString(uaRetentionD1, len(uaReports)),
		UAD3Retention:                       avgMetricString(uaRetentionD3, len(uaReports)),
		UAD7Retention:                       avgMetricString(uaRetentionD7, len(uaReports)),
		UAD30Retention:                      avgMetricString(uaRetentionD30, len(uaReports)),
		UAROAS:                              avgMetricString(uaROAS, len(uaReports)),
		UACPI:                               avgMetricString(uaCPI, len(uaReports)),
		UAROI:                               avgMetricString(uaROI, len(uaReports)),
		UAActivationRate:                    avgMetricString(uaActivationRate, len(uaReports)),
		UARegistrationRate:                  avgMetricString(uaRegistrationRate, len(uaReports)),
		UAPayerRate:                         avgMetricString(uaPayerRate, len(uaReports)),
		UAARPU:                              avgMetricString(uaARPU, len(uaReports)),
		UAARPPU:                             avgMetricString(uaARPPU, len(uaReports)),
		UALTVD7:                             avgMetricString(uaLTVD7, len(uaReports)),
		UALTVD30:                            avgMetricString(uaLTVD30, len(uaReports)),
		UALTVToCPIRatio:                     avgMetricString(uaLTVToCPIRatio, len(uaReports)),
		UAAvgOnlineDurationSeconds:          avgInt64Value(uaAvgOnlineDurationSeconds, len(uaReports)),
		UATaskCompletionRate:                avgMetricString(uaTaskCompletionRate, len(uaReports)),
		UAHighValuePayerRatio:               avgMetricString(uaHighValuePayerRatio, len(uaReports)),
		UADataStatus:                        uaDataStatus,
		CreativeCount:                       len(creativeIDs),
		CreativeTypeCount:                   len(creativeTypes),
		PlacementCount:                      len(placements),
		CountryCount:                        len(countries),
		OSCount:                             len(operatingSystems),
		ImpressionsSVG:                      buildTrendBars(insights, "impressions"),
		ClicksSVG:                           buildTrendBars(insights, "clicks"),
		SpendSVG:                            buildTrendBars(insights, "spend"),
		UAInstallsSVG:                       buildUATrendBars(uaReports, "installs"),
		UARevenueSVG:                        buildUATrendBars(uaReports, "revenue"),
		PlatformSummary:                     buildPlatformSummary(snapshots, insights),
		AccountSummary:                      buildAccountSummary(snapshots, insights),
		Snapshots:                           snapshots,
		Campaigns:                           campaigns,
		Insights:                            insights,
		InsightDetails:                      insightDetails,
		CampaignDiagnostics:                 campaignDiagnostics,
		SearchTerms:                         searchTerms,
		UAReports:                           uaReports,
		GameKPIs:                            gameKPIs,
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

func buildUATrendBars(rows []biquerydomain.UAReportRow, metric string) template.HTML {
	type row struct {
		installs int64
		revenue  float64
	}
	grouped := make(map[string]*row, len(rows))
	for _, item := range rows {
		label := item.StatDate.Format("2006-01-02")
		if _, ok := grouped[label]; !ok {
			grouped[label] = &row{}
		}
		grouped[label].installs += item.Installs
		grouped[label].revenue += parseMetricFloat(item.TotalRevenue)
	}
	if len(grouped) == 0 {
		return template.HTML(`<div class="empty-chart">No data</div>`)
	}
	labels := make([]string, 0, len(grouped))
	for label := range grouped {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	maxValue := 0.0
	values := make([]float64, 0, len(labels))
	for _, label := range labels {
		value := float64(grouped[label].installs)
		if metric == "revenue" {
			value = grouped[label].revenue
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

func parseMetricFloat(raw string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return value
}

func formatMetricFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func avgMetricString(total float64, count int) string {
	if count == 0 {
		return "0.00"
	}
	return formatMetricFloat(total / float64(count))
}

func avgInt64Value(total int64, count int) int64 {
	if count == 0 {
		return 0
	}
	return total / int64(count)
}

func uaFieldStats() (available, integrationReady, planned int) {
	for _, item := range uaFieldCatalog() {
		switch item.Status {
		case "available":
			available++
		case "integration_ready":
			integrationReady++
		case "planned":
			planned++
		}
	}
	return available, integrationReady, planned
}

var dashboardTemplate = template.Must(template.New("dashboard").Funcs(template.FuncMap{
	"localStateClass": localStateClass,
}).Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>be_ads Control Panel</title>
  <style>
    :root { --bg:#f5efe3; --bg-soft:#fbf8f2; --ink:#1d2a24; --muted:#66756d; --card:#fffdfa; --line:#dccfbd; --accent:#176b5b; --accent-strong:#0f584b; --accent-soft:#e2efe9; --warn:#b85c38; --warn-soft:#fff0e8; --shadow:0 18px 50px rgba(59,52,41,0.08); }
    * { box-sizing:border-box; }
    body { margin:0; font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "PingFang SC", "Helvetica Neue", sans-serif; background:
      radial-gradient(circle at top left, rgba(255,255,255,0.8), transparent 28%),
      radial-gradient(circle at top right, rgba(23,107,91,0.08), transparent 24%),
      linear-gradient(180deg,#efe2cc 0%,#f8f4ed 42%,#fcfaf7 100%); color:var(--ink); }
    .wrap { max-width:1380px; margin:0 auto; padding:28px 18px 52px; }
    .hero { display:grid; grid-template-columns:1.4fr 0.8fr; gap:18px; margin-bottom:20px; }
    .hero-card { background:linear-gradient(135deg, rgba(255,253,248,0.98), rgba(247,243,235,0.96)); border:1px solid rgba(220,207,189,0.9); border-radius:28px; padding:26px; box-shadow:var(--shadow); }
    .hero-card.accent { background:linear-gradient(135deg, rgba(23,107,91,0.98), rgba(15,88,75,0.96)); color:#f3fbf8; border-color:rgba(15,88,75,0.95); }
    .hero h1 { margin:0; font-size:42px; letter-spacing:-0.04em; }
    .hero p { margin:10px 0 0; color:var(--muted); font-size:15px; max-width:720px; }
    .hero-card.accent p { color:rgba(243,251,248,0.78); }
    .hero-label { display:inline-flex; align-items:center; gap:8px; padding:7px 12px; border-radius:999px; background:var(--accent-soft); color:var(--accent-strong); font-size:12px; font-weight:700; letter-spacing:0.08em; text-transform:uppercase; }
    .hero-card.accent .hero-label { background:rgba(255,255,255,0.14); color:#f3fbf8; }
    .hero-meta { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:12px; margin-top:18px; }
    .hero-stat { padding:14px 16px; border-radius:18px; background:rgba(255,255,255,0.55); border:1px solid rgba(220,207,189,0.85); }
    .hero-card.accent .hero-stat { background:rgba(255,255,255,0.08); border-color:rgba(255,255,255,0.12); }
    .hero-stat strong { display:block; font-size:24px; margin-top:6px; }
    .grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:16px; margin-bottom:24px; }
    .card { background:rgba(255,253,250,0.98); border:1px solid rgba(220,207,189,0.92); border-radius:24px; padding:20px; box-shadow:var(--shadow); }
    .metric-card { position:relative; overflow:hidden; }
    .metric-card::after { content:""; position:absolute; inset:auto -40px -60px auto; width:140px; height:140px; border-radius:50%; background:radial-gradient(circle, rgba(23,107,91,0.12), transparent 70%); pointer-events:none; }
    .metric { font-size:34px; font-weight:800; margin:8px 0 0; letter-spacing:-0.03em; }
    .label { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:0.12em; }
    .panel { margin-top:18px; }
    .panel h2 { margin:0 0 12px; font-size:18px; }
    table { width:100%; border-collapse:collapse; font-size:14px; }
    th, td { text-align:left; padding:10px 8px; border-bottom:1px solid #ece5d9; }
    th { color:var(--muted); font-weight:600; font-size:12px; text-transform:uppercase; letter-spacing:0.06em; }
    .two { display:grid; grid-template-columns:1.1fr 1fr; gap:16px; }
    .actions { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:16px; margin-bottom:16px; }
    .field { display:flex; flex-direction:column; gap:6px; margin-bottom:10px; }
    .stack-intro { display:flex; justify-content:space-between; align-items:flex-start; gap:14px; margin-bottom:12px; }
    .stack-intro p { margin:6px 0 0; color:var(--muted); }
    input, select, button { font:inherit; border-radius:14px; border:1px solid var(--line); padding:11px 13px; background:white; }
    button { background:linear-gradient(135deg, var(--accent), var(--accent-strong)); color:white; border:none; cursor:pointer; font-weight:700; box-shadow:0 10px 20px rgba(23,107,91,0.18); }
    button.secondary { background:var(--accent-soft); color:var(--accent-strong); box-shadow:none; }
    button.ghost { background:#f4eee5; color:#4f5d55; box-shadow:none; }
    .status-dot { width:10px; height:10px; border-radius:50%; display:inline-block; margin-right:8px; background:#0d7c66; }
    .status-dot.warn { background:#b85c38; }
    .status-dot.dim { background:#b8b1a3; }
    .status-grid { display:grid; grid-template-columns:1fr 1fr; gap:12px; margin:12px 0; }
    .status-list { display:flex; flex-direction:column; gap:8px; }
    .status-item { display:flex; justify-content:space-between; gap:12px; padding:10px 12px; border:1px solid #ece5d9; border-radius:14px; background:#faf7f1; }
    .status-item.warn { border-color:#e6c4b6; background:#fff4ef; }
    .status-item.dim { opacity:0.72; }
    .status-item strong { display:block; font-size:14px; }
    .status-item span { color:var(--muted); font-size:12px; }
    .action-bar { display:flex; gap:10px; flex-wrap:wrap; }
    .output-box { margin-top:12px; background:#1f2824; color:#d9efe6; border-radius:14px; padding:12px; max-height:260px; overflow:auto; white-space:pre-wrap; font:12px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
    .subsection { margin-top:14px; }
    .subsection h3 { margin:0 0 8px; font-size:14px; text-transform:uppercase; letter-spacing:0.08em; color:var(--muted); }
    .pill-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:10px; }
    .pill { border:1px solid #ece5d9; border-radius:14px; padding:12px; background:#faf7f1; }
    .pill.warn { border-color:#e6c4b6; background:#fff4ef; }
    .pill.dim { opacity:0.72; }
    .pill strong { display:block; font-size:14px; }
    .pill span { display:block; color:var(--muted); font-size:12px; margin-top:4px; word-break:break-word; }
    .worker-scale { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:12px; margin-top:12px; }
    .worker-card { border:1px solid #ece5d9; border-radius:18px; padding:14px; background:#faf7f1; }
    .worker-head { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:10px; }
    .worker-head h3 { margin:0; font-size:16px; }
    .worker-count { font-size:28px; font-weight:800; letter-spacing:-0.03em; }
    .instance-list { display:flex; flex-direction:column; gap:8px; margin:10px 0 14px; }
    .instance-chip { display:flex; justify-content:space-between; gap:12px; padding:10px 12px; border-radius:14px; border:1px solid #ece5d9; background:#fffdf9; }
    .instance-chip.warn { border-color:#e6c4b6; background:#fff4ef; }
    .instance-chip.dim { opacity:0.72; }
    .instance-chip small { color:var(--muted); display:block; margin-top:4px; }
    .inline-actions { display:flex; gap:8px; flex-wrap:wrap; }
    .log-grid { display:grid; grid-template-columns:1fr 1fr; gap:12px; margin-top:12px; }
    .log-card { border:1px solid #ece5d9; border-radius:14px; padding:12px; background:#faf7f1; }
    .log-card.warn { border-color:#e6c4b6; background:#fff4ef; }
    .log-card h3 { margin:0 0 8px; font-size:14px; }
    .log-lines { margin:0; padding-left:18px; color:#415047; font:12px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
    .log-lines li { margin-bottom:6px; }
    .tiny { font-size:12px; color:var(--muted); }
    .badge { display:inline-block; padding:4px 8px; border-radius:999px; background:#e4f1ed; color:var(--accent); font-size:12px; font-weight:600; }
    .warn { background:#f8e8e0; color:var(--warn); }
    .dim { background:#ece8df; color:#6f786f; }
    @media (max-width: 1100px) { .hero, .grid, .two, .actions, .status-grid, .pill-grid, .log-grid, .worker-scale, .hero-meta { grid-template-columns:1fr 1fr; } }
    @media (max-width: 640px) { .hero, .grid, .two, .actions, .status-grid, .pill-grid, .log-grid, .worker-scale, .hero-meta { grid-template-columns:1fr; } .wrap { padding:20px 14px 40px; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div class="hero-card">
        <span class="hero-label">Local Ops Console</span>
        <h1>be_ads Control Panel</h1>
        <p>把本地基础设施、worker 扩容、链路验证和控制面观察收在一个页面里。当前页面时间 {{ .GeneratedAt }}</p>
        <div class="hero-meta">
          <div class="hero-stat">
            <span class="label">Worker Leases</span>
            <strong>{{ len .WorkerLeases }}</strong>
          </div>
          <div class="hero-stat">
            <span class="label">Shard Assignments</span>
            <strong>{{ len .ShardAssignments }}</strong>
          </div>
          <div class="hero-stat">
            <span class="label">Outbox Pending</span>
            <strong>{{ .OutboxPending }}</strong>
          </div>
          <div class="hero-stat">
            <span class="label">DLQ Messages</span>
            <strong>{{ .DeadLetterCount }}</strong>
          </div>
        </div>
      </div>
      <div class="hero-card accent">
        <span class="hero-label">Current Focus</span>
        <h2 style="margin:14px 0 6px;font-size:28px;">Worker Scale & Local Debug</h2>
        <p>现在可以直接在网页里启动 infra、启动 worker、增加额外 collector / transformer 实例，并从端口和日志判断本地栈是否健康。</p>
        <div class="hero-meta">
          <div class="hero-stat">
            <span class="label">Snapshots</span>
            <strong>{{ len .Snapshots }}</strong>
          </div>
          <div class="hero-stat">
            <span class="label">Pipelines</span>
            <strong>2</strong>
          </div>
        </div>
      </div>
    </div>

    <div class="grid">
      <div class="card metric-card"><div class="label">Worker Leases</div><div class="metric">{{ len .WorkerLeases }}</div></div>
      <div class="card metric-card"><div class="label">Shard Assignments</div><div class="metric">{{ len .ShardAssignments }}</div></div>
      <div class="card metric-card"><div class="label">Raw Records</div><div class="metric">{{ .RawRecordCount }}</div></div>
      <div class="card metric-card"><div class="label">DLQ Messages</div><div class="metric">{{ .DeadLetterCount }}</div></div>
    </div>

    <div class="grid">
      <div class="card metric-card"><div class="label">Outbox Pending</div><div class="metric">{{ .OutboxPending }}</div></div>
      <div class="card metric-card"><div class="label">Outbox Published</div><div class="metric">{{ .OutboxPublished }}</div></div>
      <div class="card metric-card"><div class="label">Account Snapshots</div><div class="metric">{{ len .Snapshots }}</div></div>
      <div class="card metric-card"><div class="label">Pipelines</div><div class="metric">2</div></div>
    </div>

		    <div class="actions">
		      <div class="card">
		        <div class="stack-intro">
		          <div>
		            <h2>Local Stack</h2>
		            <p>先看基础设施和服务是否就绪，再用按钮做本地启停和整栈验证。</p>
		          </div>
		          <span class="badge">Runbook Friendly</span>
		        </div>
		        {{ if .LocalStack }}
		        <div class="status-grid">
	          <div>
	            <div class="label">Services</div>
	            <div class="status-list">
	              {{ range .LocalStack.Services }}
	              <div class="status-item {{ localStateClass .State }}">
	                <div><strong>{{ .Name }}</strong><span>{{ if .Detail }}{{ .Detail }}{{ else }}no extra detail{{ end }}</span></div>
	                <div><span class="badge {{ localStateClass .State }}">{{ .State }}</span></div>
	              </div>
	              {{ else }}
	              <div class="tiny">No local service state</div>
	              {{ end }}
	            </div>
	          </div>
	          <div>
	            <div class="label">Infra</div>
	            <div class="status-list">
	              {{ range .LocalStack.Infra }}
	              <div class="status-item {{ localStateClass .State }}">
	                <div><strong>{{ .Name }}</strong><span>{{ if .Detail }}{{ .Detail }}{{ else }}docker resource{{ end }}</span></div>
	                <div><span class="badge {{ localStateClass .State }}">{{ .State }}</span></div>
	              </div>
	              {{ else }}
	              <div class="tiny">No local infra state</div>
	              {{ end }}
	            </div>
	          </div>
	        </div>
		        <div class="action-bar">
		          <button onclick="runLocalStackAction('start-infra')">Start Infra</button>
		          <button onclick="runLocalStackAction('start-workers')">Start Workers</button>
	          <button onclick="runLocalStackAction('restart-collector')">Restart Collector</button>
	          <button onclick="runLocalStackAction('start')">Start Local Stack</button>
	          <button onclick="runLocalStackAction('restart')">Restart Stack</button>
	          <button onclick="runLocalStackAction('verify')">Verify Stack</button>
	          <button class="secondary" onclick="refreshLocalStackStatus()">Refresh Status</button>
	          <button class="secondary" onclick="runLocalStackAction('stop-infra')">Stop Infra</button>
	          <button class="secondary" onclick="runLocalStackAction('stop-workers')">Stop Workers</button>
	          <button class="secondary" onclick="runLocalStackAction('stop')">Stop Local Stack</button>
		        </div>
		        <div id="local-stack-result" class="tiny">Last refreshed at {{ .LocalStack.UpdatedAt }}</div>
	        <div class="subsection">
	          <h3>Port Health</h3>
	          <div class="pill-grid">
	            {{ range .LocalStack.Ports }}
	            <div class="pill {{ localStateClass .State }}">
	              <strong>{{ .Name }} :{{ .Port }}</strong>
	              <span>{{ .State }}</span>
	              <span>{{ .Detail }}</span>
	            </div>
	            {{ else }}
	            <div class="tiny">No port data</div>
	            {{ end }}
	          </div>
	        </div>
	        <div class="subsection">
	          <h3>Recent Logs</h3>
	          <div class="log-grid">
	            {{ range .LocalStack.Logs }}
	            <div class="log-card {{ localStateClass .State }}">
	              <h3>{{ .Name }}</h3>
	              <div class="tiny">state: {{ .State }}</div>
	              <ul class="log-lines">
	                {{ range .Lines }}
	                <li>{{ . }}</li>
	                {{ end }}
	              </ul>
	            </div>
	            {{ else }}
	            <div class="tiny">No recent logs</div>
	            {{ end }}
	          </div>
	        </div>
		        <pre id="local-stack-output" class="output-box">{{ .LocalStack.Output }}</pre>
		        {{ else }}
		        <div class="tiny">Local stack controls are disabled in this runtime.</div>
		        {{ end }}
		      </div>
		      <div class="card">
		        <div class="stack-intro">
		          <div>
		            <h2>Worker Scale</h2>
		            <p>这里可以单独增加或回收本地额外 worker 实例，验证 shard 重新分配和吞吐变化。</p>
		          </div>
		          <span class="badge">Scale Test</span>
		        </div>
		        {{ if .LocalStack }}
		        <div class="worker-scale">
		          {{ range .LocalStack.Workers }}
		          <div class="worker-card">
		            <div class="worker-head">
		              <div>
		                <div class="label">{{ .Role }} workers</div>
		                <h3>{{ .Role }}</h3>
		              </div>
		              <div class="worker-count">{{ .RunningCount }} / {{ .TotalCount }}</div>
		            </div>
		            <div class="instance-list">
		              {{ range .Instances }}
		              <div class="instance-chip {{ localStateClass .State }}">
		                <div>
		                  <strong>{{ .Name }}</strong>
		                  <small>{{ .Detail }}</small>
		                </div>
		                <span class="badge {{ localStateClass .State }}">{{ .State }}</span>
		              </div>
		              {{ else }}
		              <div class="tiny">No {{ .Role }} worker instances yet.</div>
		              {{ end }}
		            </div>
		            <div class="inline-actions">
		              <button onclick="scaleWorker('{{ .Role }}','add')">Add {{ .Role }}</button>
		              <button class="secondary" onclick="scaleWorker('{{ .Role }}','remove')">Remove Extra</button>
		            </div>
		          </div>
		          {{ end }}
		        </div>
		        <div id="worker-scale-result" class="tiny">Use add/remove to simulate local horizontal scaling.</div>
		        {{ else }}
		        <div class="tiny">Worker scale controls are disabled in this runtime.</div>
		        {{ end }}
		      </div>
		      <div class="card">
		        <div class="stack-intro">
		          <div>
		            <h2>Recovery & Backfill</h2>
		            <p>把补数据和故障回放集中放在一起，方便在扩容后快速验证链路恢复能力。</p>
		          </div>
		          <span class="badge">Recovery</span>
		        </div>
	        <div class="field"><label>Replay Kind</label><select id="replay-kind"><option value="raw_event">raw_event</option><option value="collect_job">collect_job</option></select></div>
	        <div class="field"><label>Replay Platform</label><select id="replay-platform"><option value="">all</option><option value="facebook">facebook</option><option value="google_ads">google_ads</option><option value="tiktok_ads">tiktok_ads</option></select></div>
	        <div class="field"><label>Replay Limit</label><input id="replay-limit" type="number" value="10"></div>
	        <button onclick="replayDlq()">Replay DLQ</button>
	        <div id="replay-result" class="tiny"></div>
	        <hr style="border:none;border-top:1px solid #ece5d9;margin:16px 0;">
	        <div class="field"><label>Platform</label><select id="backfill-platform"><option value="">all</option><option value="facebook">facebook</option><option value="google_ads">google_ads</option><option value="tiktok_ads">tiktok_ads</option></select></div>
	        <div class="field"><label>Account ID</label><input id="backfill-account" type="text" placeholder="248-390-1805"></div>
	        <div class="field"><label>Object Type</label><select id="backfill-object"><option value="">all</option><option value="campaign">campaign</option><option value="ad_group">ad_group</option><option value="ad">ad</option><option value="insight">insight</option></select></div>
	        <button class="secondary" onclick="dispatchBackfill()">Dispatch Backfill</button>
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
	    function setText(id, text) {
	      const el = document.getElementById(id);
	      if (el) el.textContent = text;
	    }
	    async function replayDlq() {
	      const payload = {
	        kind: document.getElementById('replay-kind').value,
	        platform: document.getElementById('replay-platform').value,
	        limit: Number(document.getElementById('replay-limit').value || '10')
	      };
	      const result = await fetchJSON('/api/control/dlq/replay', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
	      setText('replay-result', 'replayed ' + result.accepted + ' item(s)');
	      setTimeout(() => location.reload(), 800);
	    }
	    async function dispatchBackfill() {
	      const payload = {
	        platform: document.getElementById('backfill-platform').value,
	        account_id: document.getElementById('backfill-account').value,
	        object_type: document.getElementById('backfill-object').value
	      };
	      const result = await fetchJSON('/api/control/backfill', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
	      setText('backfill-result', 'dispatched ' + result.accepted + ' job(s)');
	      setTimeout(() => location.reload(), 800);
	    }
	    async function refreshLocalStackStatus() {
	      setText('local-stack-result', 'refreshing local status...');
	      const result = await fetchJSON('/api/control/local-stack');
	      const output = document.getElementById('local-stack-output');
	      if (output) output.textContent = result.output || 'no output';
	      setText('local-stack-result', 'local status refreshed at ' + result.updated_at);
	      setTimeout(() => location.reload(), 500);
	    }
	    async function runLocalStackAction(action) {
	      setText('local-stack-result', (action === 'start' ? 'starting' : 'stopping') + ' local stack...');
	      try {
	        const result = await fetchJSON('/api/control/local-stack/' + action, {method:'POST'});
	        const output = document.getElementById('local-stack-output');
	        if (output) output.textContent = result.output || 'no output';
	        setText('local-stack-result', action + ' finished at ' + result.finished_at);
	      } catch (err) {
	        setText('local-stack-result', action + ' failed: ' + err.message);
	      }
	      setTimeout(() => location.reload(), 1200);
	    }
	    async function scaleWorker(role, action) {
	      setText('worker-scale-result', action + ' ' + role + ' worker...');
	      try {
	        const result = await fetchJSON('/api/control/local-stack/' + action + '-worker', {
	          method:'POST',
	          headers:{'Content-Type':'application/json'},
	          body:JSON.stringify({role})
	        });
	        const output = document.getElementById('local-stack-output');
	        if (output) output.textContent = result.output || 'no output';
	        setText('worker-scale-result', action + ' ' + role + ' worker finished at ' + result.finished_at);
	      } catch (err) {
	        setText('worker-scale-result', action + ' ' + role + ' worker failed: ' + err.message);
	      }
	      setTimeout(() => location.reload(), 1200);
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
  <title>be_ads BI {{ .PageTitle }}</title>
  <style>
    :root { --card:#fffdfa; --line:#ddd4c6; --ink:#223127; --muted:#68756d; --accent:#146c5a; --accent-soft:#e0efe9; --warn:#c46d37; --warn-soft:#f8e7dc; }
    * { box-sizing:border-box; }
    body { margin:0; background:linear-gradient(180deg,#efe5d2 0%,#f7f4ed 100%); color:var(--ink); font-family:ui-sans-serif, -apple-system, BlinkMacSystemFont, "PingFang SC", "Helvetica Neue", sans-serif; }
    .wrap { max-width:1380px; margin:0 auto; padding:28px 18px 48px; }
    .hero { display:flex; justify-content:space-between; align-items:flex-end; gap:16px; margin-bottom:18px; }
    .hero h1 { margin:0; font-size:34px; letter-spacing:-0.03em; }
    .hero p { margin:8px 0 0; color:var(--muted); }
    .nav { display:flex; gap:10px; flex-wrap:wrap; }
    .nav a { color:var(--accent); text-decoration:none; background:var(--accent-soft); padding:10px 14px; border-radius:999px; font-weight:700; }
    .nav a.active { background:var(--accent); color:#fff; }
    .filters { display:grid; grid-template-columns:repeat(8,minmax(0,1fr)); gap:12px; background:var(--card); border:1px solid var(--line); border-radius:20px; padding:16px; box-shadow:0 12px 30px rgba(34,49,39,0.06); margin-bottom:18px; }
    .field { display:flex; flex-direction:column; gap:6px; }
    .field label { font-size:12px; color:var(--muted); text-transform:uppercase; letter-spacing:0.06em; }
    input, select, button { font:inherit; border-radius:12px; border:1px solid var(--line); padding:10px 12px; background:white; }
    button { background:var(--accent); color:white; border:none; cursor:pointer; font-weight:700; }
    .metrics { display:grid; grid-template-columns:repeat(6,minmax(0,1fr)); gap:14px; margin-bottom:18px; }
    .charts, .two { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:14px; margin-bottom:18px; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:20px; padding:18px; box-shadow:0 12px 30px rgba(34,49,39,0.06); }
    .label { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:0.08em; }
    .metric { margin-top:8px; font-size:30px; font-weight:700; }
    .metric.good { color:var(--accent); }
    .metric.warn { color:var(--warn); }
    .panels { display:grid; grid-template-columns:1fr; gap:16px; }
    .panel-head { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:10px; }
    .panel-head h2 { margin:0; font-size:18px; }
    .panel-head span { font-size:12px; color:var(--muted); }
    .table-wrap { overflow:auto; }
    table { width:100%; border-collapse:collapse; font-size:14px; min-width:920px; }
    th, td { text-align:left; padding:10px 8px; border-bottom:1px solid #ece4d8; vertical-align:top; white-space:nowrap; }
    th { color:var(--muted); font-weight:600; font-size:12px; text-transform:uppercase; letter-spacing:0.06em; }
    .badge { display:inline-block; padding:4px 8px; border-radius:999px; background:var(--accent-soft); color:var(--accent); font-size:12px; font-weight:700; }
    .badge.warn { background:var(--warn-soft); color:var(--warn); }
    .muted { color:var(--muted); }
    .trend-svg { width:100%; height:96px; display:block; margin-top:14px; }
    .trend-svg rect { fill:#2e8973; opacity:0.92; }
    .empty-chart { margin-top:14px; min-height:96px; display:flex; align-items:center; justify-content:center; border-radius:16px; background:#f5f1e7; color:var(--muted); }
    @media (max-width: 1080px) { .filters, .metrics, .charts, .two { grid-template-columns:repeat(2,minmax(0,1fr)); } }
    @media (max-width: 640px) { .hero { flex-direction:column; align-items:flex-start; } .filters, .metrics, .charts, .two { grid-template-columns:1fr; } .wrap { padding:20px 14px 40px; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div>
        <h1>be_ads BI {{ .PageTitle }}</h1>
        <p>{{ .PageDescription }} 当前生成时间 {{ .GeneratedAt }}</p>
      </div>
      <div class="nav">
        <a href="/bi/overview" {{ if eq .PageKey "overview" }}class="active"{{ end }}>Overview</a>
        <a href="/bi/breakdown" {{ if eq .PageKey "breakdown" }}class="active"{{ end }}>Breakdown</a>
        <a href="/bi/creatives" {{ if eq .PageKey "creatives" }}class="active"{{ end }}>Creatives</a>
        <a href="/bi/quality" {{ if eq .PageKey "quality" }}class="active"{{ end }}>Quality</a>
        <a href="/">Control Panel</a>
      </div>
    </div>

    <form class="filters" method="get">
      <div class="field">
        <label>Platform</label>
        <select name="platform">
          <option value="" {{ if eq .Platform "" }}selected{{ end }}>all</option>
          <option value="facebook" {{ if eq .Platform "facebook" }}selected{{ end }}>facebook</option>
          <option value="google_ads" {{ if eq .Platform "google_ads" }}selected{{ end }}>google_ads</option>
          <option value="tiktok_ads" {{ if eq .Platform "tiktok_ads" }}selected{{ end }}>tiktok_ads</option>
        </select>
      </div>
      <div class="field"><label>Account ID</label><input type="text" name="account_id" value="{{ .AccountID }}" placeholder="248-390-1805"></div>
      <div class="field"><label>Date From</label><input type="date" name="date_from" value="{{ .DateFrom }}"></div>
      <div class="field"><label>Date To</label><input type="date" name="date_to" value="{{ .DateTo }}"></div>
      <div class="field">
        <label>Entity Level</label>
        <select name="entity_level">
          <option value="" {{ if eq .EntityLevel "" }}selected{{ end }}>all</option>
          <option value="campaign" {{ if eq .EntityLevel "campaign" }}selected{{ end }}>campaign</option>
          <option value="ad_group" {{ if eq .EntityLevel "ad_group" }}selected{{ end }}>ad_group</option>
          <option value="ad" {{ if eq .EntityLevel "ad" }}selected{{ end }}>ad</option>
        </select>
      </div>
      <div class="field"><label>Country</label><input type="text" name="country" value="{{ .Country }}" placeholder="US"></div>
      <div class="field"><label>OS</label><input type="text" name="os" value="{{ .OS }}" placeholder="ios"></div>
      <div class="field"><label>Device</label><input type="text" name="device" value="{{ .Device }}" placeholder="MOBILE"></div>
      <div class="field"><label>Network</label><input type="text" name="network" value="{{ .Network }}" placeholder="SEARCH"></div>
      <div class="field"><label>Campaign ID</label><input type="text" name="platform_campaign_id" value="{{ .PlatformCampaignID }}"></div>
      <div class="field"><label>Ad Group ID</label><input type="text" name="platform_ad_group_id" value="{{ .PlatformAdGroupID }}"></div>
      <div class="field"><label>Ad ID</label><input type="text" name="platform_ad_id" value="{{ .PlatformAdID }}"></div>
      <div class="field"><label>Detail Limit</label><input type="number" name="detail_limit" value="{{ .DetailLimit }}"></div>
      <div class="field"><label>Match Type</label><input type="text" name="match_type" value="{{ .MatchType }}" placeholder="EXACT"></div>
      <div class="field"><label>Search Term</label><input type="text" name="search_term" value="{{ .SearchTerm }}" placeholder="brand"></div>
      <div class="field"><label>Search Limit</label><input type="number" name="search_term_limit" value="{{ .SearchTermLimit }}"></div>
      <div class="field"><label>Action</label><button type="submit">刷新当前页</button></div>
    </form>

    {{ if eq .PageKey "overview" }}
    <div class="metrics">
      <div class="card"><div class="label">UA Status</div><div class="metric {{ if gt .GameKPIRowCount 0 }}good{{ else }}warn{{ end }}">{{ .UADataStatus }}</div></div>
      <div class="card"><div class="label">Spend / Revenue</div><div class="metric good">{{ .TotalSpend }} / {{ .UATotalRevenue }}</div></div>
      <div class="card"><div class="label">Installs / Purchasers</div><div class="metric">{{ .UAInstalls }} / {{ .UAPurchasers }}</div></div>
      <div class="card"><div class="label">Retention D1 / D7</div><div class="metric">{{ .UAD1Retention }} / {{ .UAD7Retention }}</div></div>
      <div class="card"><div class="label">ROAS / ROI</div><div class="metric">{{ .UAROAS }} / {{ .UAROI }}</div></div>
      <div class="card"><div class="label">LTV D7 / D30</div><div class="metric">{{ .UALTVD7 }} / {{ .UALTVD30 }}</div></div>
      <div class="card"><div class="label">Impressions / Clicks</div><div class="metric">{{ .TotalImpressions }} / {{ .TotalClicks }}</div></div>
      <div class="card"><div class="label">CTR / CPC / CPM</div><div class="metric">{{ .AvgCTR }} / {{ .AvgCPC }} / {{ .AvgCPM }}</div></div>
      <div class="card"><div class="label">Reach / Frequency</div><div class="metric">{{ .TotalReach }} / {{ .AvgFrequency }}</div></div>
      <div class="card"><div class="label">Snapshots / Campaigns</div><div class="metric">{{ .SnapshotCount }} / {{ .CampaignCount }}</div></div>
      <div class="card"><div class="label">Revenue D1 / D7</div><div class="metric">{{ .UARevenueD1 }} / {{ .UARevenueD7 }}</div></div>
      <div class="card"><div class="label">Activation / Registration</div><div class="metric">{{ .UAActivations }} / {{ .UARegistrations }}</div></div>
    </div>

    <div class="charts">
      <div class="card"><div class="panel-head"><h2>Spend Trend</h2><span>ads summary</span></div>{{ .SpendSVG }}</div>
      <div class="card"><div class="panel-head"><h2>Install Trend</h2><span>ua report</span></div>{{ .UAInstallsSVG }}</div>
      <div class="card"><div class="panel-head"><h2>Revenue Trend</h2><span>ua report</span></div>{{ .UARevenueSVG }}</div>
      <div class="card"><div class="panel-head"><h2>Impression Trend</h2><span>insight summary</span></div>{{ .ImpressionsSVG }}</div>
    </div>

    <div class="two">
      <div class="card">
        <div class="panel-head"><h2>Platform Summary</h2><span>{{ len .PlatformSummary }} group(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Platform</th><th>Accounts</th><th>Campaigns</th><th>Ad Groups</th><th>Ads</th><th>Insights</th><th>Impressions</th><th>Clicks</th><th>Spend</th></tr></thead>
            <tbody>{{ range .PlatformSummary }}<tr><td><span class="badge">{{ .Platform }}</span></td><td>{{ .AccountCount }}</td><td>{{ .CampaignCount }}</td><td>{{ .AdGroupCount }}</td><td>{{ .AdCount }}</td><td>{{ .InsightCount }}</td><td>{{ .TotalImpressions }}</td><td>{{ .TotalClicks }}</td><td>{{ .TotalSpend }}</td></tr>{{ else }}<tr><td colspan="9" class="muted">No grouped platform data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="panel-head"><h2>Account Summary</h2><span>{{ len .AccountSummary }} group(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Platform</th><th>Account</th><th>Name</th><th>Source</th><th>Campaigns</th><th>Ad Groups</th><th>Ads</th><th>Insights</th><th>Spend</th></tr></thead>
            <tbody>{{ range .AccountSummary }}<tr><td>{{ .Platform }}</td><td>{{ .AccountID }}</td><td>{{ if .AccountName }}{{ .AccountName }}{{ else }}-{{ end }}</td><td>{{ .SourceMode }}</td><td>{{ .CampaignCount }}</td><td>{{ .AdGroupCount }}</td><td>{{ .AdCount }}</td><td>{{ .InsightCount }}</td><td>{{ .TotalSpend }}</td></tr>{{ else }}<tr><td colspan="9" class="muted">No grouped account data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="panels">
      <div class="card">
        <div class="panel-head"><h2>UA Overview</h2><span>{{ .UAReportRowCount }} row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Account</th><th>Campaign</th><th>Country</th><th>OS</th><th>Spend</th><th>Impr</th><th>Clicks</th><th>CTR</th><th>Installs</th><th>CPI</th><th>Revenue</th><th>D1</th><th>D7</th><th>ROAS</th><th>ROI</th></tr></thead>
            <tbody>{{ range .UAReports }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ .PlatformAccountID }}</td><td>{{ if .PlatformCampaignID }}{{ .PlatformCampaignID }}{{ else }}-{{ end }}</td><td>{{ if .Country }}{{ .Country }}{{ else }}<span class="muted">待接</span>{{ end }}</td><td>{{ if .OS }}{{ .OS }}{{ else }}<span class="muted">待接</span>{{ end }}</td><td>{{ .Spend }}</td><td>{{ .Impressions }}</td><td>{{ .Clicks }}</td><td>{{ .CTR }}</td><td>{{ .Installs }}</td><td>{{ .CPI }}</td><td>{{ .TotalRevenue }}</td><td>{{ .RetentionD1 }}</td><td>{{ .RetentionD7 }}</td><td>{{ .ROAS }}</td><td>{{ .ROI }}</td></tr>{{ else }}<tr><td colspan="17" class="muted">No UA data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="panel-head"><h2>Account Snapshots</h2><span>{{ .SnapshotCount }} account(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Platform</th><th>Account</th><th>Name</th><th>Source</th><th>Last Object</th><th>Campaigns</th><th>Ad Groups</th><th>Ads</th><th>Insights</th></tr></thead>
            <tbody>{{ range .Snapshots }}<tr><td><span class="badge">{{ .Platform }}</span></td><td>{{ .AccountID }}</td><td>{{ .AccountName }}</td><td>{{ .LastSourceMode }}</td><td>{{ .LastObjectType }}</td><td>{{ .CampaignCount }}</td><td>{{ .AdGroupCount }}</td><td>{{ .AdCount }}</td><td>{{ .InsightCount }}</td></tr>{{ else }}<tr><td colspan="9" class="muted">No snapshot data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
    </div>
    {{ end }}

    {{ if eq .PageKey "breakdown" }}
    <div class="metrics">
      <div class="card"><div class="label">Insight Summary Rows</div><div class="metric">{{ .InsightRowCount }}</div></div>
      <div class="card"><div class="label">Insight Detail Rows</div><div class="metric">{{ .InsightDetailRowCount }}</div></div>
      <div class="card"><div class="label">Campaign Diag Rows</div><div class="metric">{{ .CampaignDiagnosticRowCount }}</div></div>
      <div class="card"><div class="label">Search Terms</div><div class="metric">{{ .SearchTermRowCount }}</div></div>
      <div class="card"><div class="label">Conversions / Value</div><div class="metric">{{ .TotalConversions }} / {{ .TotalConversionsValue }}</div></div>
      <div class="card"><div class="label">Search IS / Top</div><div class="metric">{{ .AvgSearchImpressionShare }} / {{ .AvgSearchTopImpressionShare }}</div></div>
    </div>
    <div class="panels">
      <div class="card">
        <div class="panel-head"><h2>Insight Detail Drilldown</h2><span>{{ .InsightDetailRowCount }} row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Account</th><th>Level</th><th>Entity</th><th>Campaign</th><th>Ad Group</th><th>Ad</th><th>Device</th><th>Network</th><th>Impr</th><th>Clicks</th><th>CTR</th><th>CPC</th><th>CPM</th><th>Spend</th><th>Reach</th><th>Conv</th><th>All Conv</th><th>Value</th><th>CPA</th></tr></thead>
            <tbody>{{ range .InsightDetails }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ .PlatformAccountID }}</td><td><span class="badge">{{ .EntityLevel }}</span></td><td>{{ .EntityID }}</td><td>{{ if .PlatformCampaignID }}{{ .PlatformCampaignID }}{{ else }}-{{ end }}</td><td>{{ if .PlatformAdGroupID }}{{ .PlatformAdGroupID }}{{ else }}-{{ end }}</td><td>{{ if .PlatformAdID }}{{ .PlatformAdID }}{{ else }}-{{ end }}</td><td>{{ if .Device }}{{ .Device }}{{ else }}-{{ end }}</td><td>{{ if .Network }}{{ .Network }}{{ else }}-{{ end }}</td><td>{{ .Impressions }}</td><td>{{ .Clicks }}</td><td>{{ .CTR }}</td><td>{{ .CPC }}</td><td>{{ .CPM }}</td><td>{{ .Spend }}</td><td>{{ .Reach }}</td><td>{{ .Conversions }}</td><td>{{ .AllConversions }}</td><td>{{ .ConversionsValue }}</td><td>{{ .CostPerConversion }}</td></tr>{{ else }}<tr><td colspan="21" class="muted">No insight detail data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="panel-head"><h2>Campaigns</h2><span>{{ .CampaignCount }} row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Platform</th><th>Account</th><th>Campaign</th><th>Status</th><th>Objective</th><th>Bidding</th><th>Budget</th><th>Start</th><th>End</th><th>Updated</th></tr></thead>
            <tbody>{{ range .Campaigns }}<tr><td>{{ .Platform }}</td><td>{{ .AccountID }}</td><td>{{ .CampaignName }}</td><td><span class="badge {{ if ne .Status "ACTIVE" }}warn{{ end }}">{{ .Status }}</span></td><td>{{ .Objective }}</td><td>{{ if .BiddingStrategy }}{{ .BiddingStrategy }}{{ else if .BuyingType }}{{ .BuyingType }}{{ else }}-{{ end }}</td><td>{{ if .DailyBudget }}{{ .DailyBudget }}{{ else }}{{ .LifetimeBudget }}{{ end }} {{ .Currency }}</td><td>{{ if .StartTime.IsZero }}-{{ else }}{{ .StartTime.Format "2006-01-02" }}{{ end }}</td><td>{{ if .EndTime.IsZero }}-{{ else }}{{ .EndTime.Format "2006-01-02" }}{{ end }}</td><td>{{ if .SourceUpdatedAt.IsZero }}-{{ else }}{{ .SourceUpdatedAt.Format "2006-01-02 15:04" }}{{ end }}</td></tr>{{ else }}<tr><td colspan="10" class="muted">No campaign data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="panel-head"><h2>Campaign Diagnostics</h2><span>{{ .CampaignDiagnosticRowCount }} row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Account</th><th>Campaign ID</th><th>Search IS</th><th>Top IS</th><th>Abs Top IS</th></tr></thead>
            <tbody>{{ range .CampaignDiagnostics }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ .PlatformAccountID }}</td><td>{{ .PlatformCampaignID }}</td><td>{{ .SearchImpressionShare }}</td><td>{{ .SearchTopImpressionShare }}</td><td>{{ .SearchAbsoluteTopImpressionShare }}</td></tr>{{ else }}<tr><td colspan="7" class="muted">No campaign diagnostic data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="panel-head"><h2>Search Term Diagnostics</h2><span>{{ .SearchTermRowCount }} row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Account</th><th>Campaign</th><th>Ad Group</th><th>Search Term</th><th>Match</th><th>Impr</th><th>Clicks</th><th>Spend</th><th>Conv</th><th>Value</th></tr></thead>
            <tbody>{{ range .SearchTerms }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ .PlatformAccountID }}</td><td>{{ .PlatformCampaignID }}</td><td>{{ if .PlatformAdGroupID }}{{ .PlatformAdGroupID }}{{ else }}-{{ end }}</td><td>{{ .SearchTerm }}</td><td>{{ if .SearchTermMatchType }}{{ .SearchTermMatchType }}{{ else }}-{{ end }}</td><td>{{ .Impressions }}</td><td>{{ .Clicks }}</td><td>{{ .Spend }}</td><td>{{ .Conversions }}</td><td>{{ .ConversionsValue }}</td></tr>{{ else }}<tr><td colspan="12" class="muted">No search term data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
    </div>
    {{ end }}

    {{ if eq .PageKey "creatives" }}
    <div class="metrics">
      <div class="card"><div class="label">Creative IDs</div><div class="metric">{{ .CreativeCount }}</div></div>
      <div class="card"><div class="label">Creative Types</div><div class="metric">{{ .CreativeTypeCount }}</div></div>
      <div class="card"><div class="label">Placements</div><div class="metric">{{ .PlacementCount }}</div></div>
      <div class="card"><div class="label">Revenue D7 / D30</div><div class="metric">{{ .UARevenueD7 }} / {{ .UARevenueD30 }}</div></div>
      <div class="card"><div class="label">Tutorial / Role Create</div><div class="metric">{{ .UATutorialCompletions }} / {{ .UARoleCreations }}</div></div>
      <div class="card"><div class="label">Purchase Count</div><div class="metric">{{ .UAPurchaseCount }}</div></div>
    </div>
    <div class="panels">
      <div class="card">
        <div class="panel-head"><h2>Creative Performance</h2><span>{{ .UAReportRowCount }} ua row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Campaign</th><th>Ad Group</th><th>Ad</th><th>Creative ID</th><th>Creative Type</th><th>Placement</th><th>Goal</th><th>Bid</th><th>Targeting</th><th>Spend</th><th>CTR</th><th>Installs</th><th>CPI</th><th>D1</th><th>D7</th><th>Revenue D7</th><th>ROAS</th><th>ROI</th></tr></thead>
            <tbody>{{ range .UAReports }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ if .PlatformCampaignID }}{{ .PlatformCampaignID }}{{ else }}-{{ end }}</td><td>{{ if .PlatformAdGroupID }}{{ .PlatformAdGroupID }}{{ else }}-{{ end }}</td><td>{{ if .PlatformAdID }}{{ .PlatformAdID }}{{ else }}-{{ end }}</td><td>{{ if .CreativeID }}{{ .CreativeID }}{{ else }}-{{ end }}</td><td>{{ if .CreativeType }}{{ .CreativeType }}{{ else }}-{{ end }}</td><td>{{ if .Placement }}{{ .Placement }}{{ else }}-{{ end }}</td><td>{{ if .OptimizationGoal }}{{ .OptimizationGoal }}{{ else }}-{{ end }}</td><td>{{ if .BidType }}{{ .BidType }}{{ else }}-{{ end }}</td><td>{{ if .Targeting }}{{ .Targeting }}{{ else }}-{{ end }}</td><td>{{ .Spend }}</td><td>{{ .CTR }}</td><td>{{ .Installs }}</td><td>{{ .CPI }}</td><td>{{ .RetentionD1 }}</td><td>{{ .RetentionD7 }}</td><td>{{ .RevenueD7 }}</td><td>{{ .ROAS }}</td><td>{{ .ROI }}</td></tr>{{ else }}<tr><td colspan="20" class="muted">No creative-side UA data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="panel-head"><h2>Creative-linked Game KPIs</h2><span>{{ .GameKPIRowCount }} game row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Creative ID</th><th>Creative Type</th><th>Placement</th><th>Targeting</th><th>Installs</th><th>Activations</th><th>Registrations</th><th>Purchasers</th><th>Purchase Count</th><th>Tutorial</th><th>Role Create</th><th>LevelX</th><th>Revenue D1</th><th>Revenue D7</th><th>Revenue D30</th></tr></thead>
            <tbody>{{ range .GameKPIs }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ if .CreativeID }}{{ .CreativeID }}{{ else }}-{{ end }}</td><td>{{ if .CreativeType }}{{ .CreativeType }}{{ else }}-{{ end }}</td><td>{{ if .Placement }}{{ .Placement }}{{ else }}-{{ end }}</td><td>{{ if .Targeting }}{{ .Targeting }}{{ else }}-{{ end }}</td><td>{{ .Installs }}</td><td>{{ .Activations }}</td><td>{{ .Registrations }}</td><td>{{ .Purchasers }}</td><td>{{ .PurchaseCount }}</td><td>{{ .TutorialCompletions }}</td><td>{{ .RoleCreations }}</td><td>{{ .LevelXUsers }}</td><td>{{ .RevenueD1 }}</td><td>{{ .RevenueD7 }}</td><td>{{ .RevenueD30 }}</td></tr>{{ else }}<tr><td colspan="17" class="muted">No creative-linked game KPI data. 可通过 POST /api/bi/game-kpis 接入。</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
    </div>
    {{ end }}

    {{ if eq .PageKey "quality" }}
    <div class="metrics">
      <div class="card"><div class="label">Country / OS</div><div class="metric">{{ .CountryCount }} / {{ .OSCount }}</div></div>
      <div class="card"><div class="label">Activation / Registration Rate</div><div class="metric">{{ .UAActivationRate }} / {{ .UARegistrationRate }}</div></div>
      <div class="card"><div class="label">Payer Rate / High Value</div><div class="metric">{{ .UAPayerRate }} / {{ .UAHighValuePayerRatio }}</div></div>
      <div class="card"><div class="label">Retention D3 / D30</div><div class="metric">{{ .UAD3Retention }} / {{ .UAD30Retention }}</div></div>
      <div class="card"><div class="label">ARPU / ARPPU</div><div class="metric">{{ .UAARPU }} / {{ .UAARPPU }}</div></div>
      <div class="card"><div class="label">Avg Online Seconds</div><div class="metric">{{ .UAAvgOnlineDurationSeconds }}</div></div>
      <div class="card"><div class="label">Task Completion</div><div class="metric">{{ .UATaskCompletionRate }}</div></div>
      <div class="card"><div class="label">LTV/CPI</div><div class="metric">{{ .UALTVToCPIRatio }}</div></div>
      <div class="card"><div class="label">Available / Ready / Planned</div><div class="metric">{{ .UAAvailableFieldCount }} / {{ .UAIntegrationReadyFieldCount }} / {{ .UAPlannedFieldCount }}</div></div>
      <div class="card"><div class="label">Fraud/Bounce Signals</div><div class="metric warn">planned</div></div>
      <div class="card"><div class="label">UA Rows</div><div class="metric">{{ .UAReportRowCount }}</div></div>
      <div class="card"><div class="label">Game KPI Rows</div><div class="metric">{{ .GameKPIRowCount }}</div></div>
    </div>
    <div class="panels">
      <div class="card">
        <div class="panel-head"><h2>UA Quality Signals</h2><span>{{ .UAReportRowCount }} row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Country</th><th>OS</th><th>Network</th><th>Placement</th><th>Frequency</th><th>CTR</th><th>CPI</th><th>Activation Rate</th><th>Registration Rate</th><th>Payer Rate</th><th>D1</th><th>D3</th><th>D7</th><th>D30</th><th>ARPU</th><th>ARPPU</th><th>LTV D7</th><th>LTV D30</th><th>LTV/CPI</th></tr></thead>
            <tbody>{{ range .UAReports }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ if .Country }}{{ .Country }}{{ else }}-{{ end }}</td><td>{{ if .OS }}{{ .OS }}{{ else }}-{{ end }}</td><td>{{ if .Network }}{{ .Network }}{{ else }}-{{ end }}</td><td>{{ if .Placement }}{{ .Placement }}{{ else }}-{{ end }}</td><td>{{ .Frequency }}</td><td>{{ .CTR }}</td><td>{{ .CPI }}</td><td>{{ .ActivationRate }}</td><td>{{ .RegistrationRate }}</td><td>{{ .PayerRate }}</td><td>{{ .RetentionD1 }}</td><td>{{ .RetentionD3 }}</td><td>{{ .RetentionD7 }}</td><td>{{ .RetentionD30 }}</td><td>{{ .ARPU }}</td><td>{{ .ARPPU }}</td><td>{{ .LTVD7 }}</td><td>{{ .LTVD30 }}</td><td>{{ .LTVToCPIRatio }}</td></tr>{{ else }}<tr><td colspan="21" class="muted">No UA quality data</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="panel-head"><h2>Game-side Quality Signals</h2><span>{{ .GameKPIRowCount }} row(s)</span></div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Date</th><th>Platform</th><th>Country</th><th>OS</th><th>Placement</th><th>Creative ID</th><th>Installs</th><th>Activations</th><th>Registrations</th><th>Purchasers</th><th>D1</th><th>D3</th><th>D7</th><th>D30</th><th>Revenue D1</th><th>Revenue D7</th><th>Revenue D30</th><th>Online Seconds</th><th>Task Completion</th><th>High Value Payer</th></tr></thead>
            <tbody>{{ range .GameKPIs }}<tr><td>{{ .StatDate.Format "2006-01-02" }}</td><td>{{ .Platform }}</td><td>{{ if .Country }}{{ .Country }}{{ else }}-{{ end }}</td><td>{{ if .OS }}{{ .OS }}{{ else }}-{{ end }}</td><td>{{ if .Placement }}{{ .Placement }}{{ else }}-{{ end }}</td><td>{{ if .CreativeID }}{{ .CreativeID }}{{ else }}-{{ end }}</td><td>{{ .Installs }}</td><td>{{ .Activations }}</td><td>{{ .Registrations }}</td><td>{{ .Purchasers }}</td><td>{{ .RetentionD1 }}</td><td>{{ .RetentionD3 }}</td><td>{{ .RetentionD7 }}</td><td>{{ .RetentionD30 }}</td><td>{{ .RevenueD1 }}</td><td>{{ .RevenueD7 }}</td><td>{{ .RevenueD30 }}</td><td>{{ .AvgOnlineDurationSeconds }}</td><td>{{ .TaskCompletionRate }}</td><td>{{ .HighValuePayerRatio }}</td></tr>{{ else }}<tr><td colspan="20" class="muted">No game-side quality data. Bounce/fraud 仍待外部信号接入。</td></tr>{{ end }}</tbody>
          </table>
        </div>
      </div>
    </div>
    {{ end }}
  </div>
  <script>
    setTimeout(() => location.reload(), 30000);
  </script>
</body>
</html>`))

func parseUAReportFilter(r *http.Request) (biquerydomain.UAReportFilter, error) {
	filter := biquerydomain.UAReportFilter{
		Platform:           strings.TrimSpace(r.URL.Query().Get("platform")),
		AccountID:          strings.TrimSpace(r.URL.Query().Get("platform_account_id")),
		EntityLevel:        strings.TrimSpace(r.URL.Query().Get("entity_level")),
		Device:             strings.TrimSpace(r.URL.Query().Get("device")),
		Network:            strings.TrimSpace(r.URL.Query().Get("network")),
		Country:            strings.TrimSpace(r.URL.Query().Get("country")),
		OS:                 strings.TrimSpace(r.URL.Query().Get("os")),
		PlatformCampaignID: strings.TrimSpace(r.URL.Query().Get("platform_campaign_id")),
		PlatformAdGroupID:  strings.TrimSpace(r.URL.Query().Get("platform_ad_group_id")),
		PlatformAdID:       strings.TrimSpace(r.URL.Query().Get("platform_ad_id")),
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return biquerydomain.UAReportFilter{}, fmt.Errorf("invalid limit")
		}
		filter.Limit = limit
	}
	if dateFrom := strings.TrimSpace(r.URL.Query().Get("date_from")); dateFrom != "" {
		parsed, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			return biquerydomain.UAReportFilter{}, fmt.Errorf("invalid date_from")
		}
		filter.DateFrom = parsed
	}
	if dateTo := strings.TrimSpace(r.URL.Query().Get("date_to")); dateTo != "" {
		parsed, err := time.Parse("2006-01-02", dateTo)
		if err != nil {
			return biquerydomain.UAReportFilter{}, fmt.Errorf("invalid date_to")
		}
		filter.DateTo = parsed
	}
	return filter, nil
}

func parseGameKPIQueryFilter(r *http.Request) (biquerydomain.GameKPIQueryFilter, error) {
	filter := biquerydomain.GameKPIQueryFilter{
		Platform:           strings.TrimSpace(r.URL.Query().Get("platform")),
		AccountID:          strings.TrimSpace(r.URL.Query().Get("platform_account_id")),
		Country:            strings.TrimSpace(r.URL.Query().Get("country")),
		OS:                 strings.TrimSpace(r.URL.Query().Get("os")),
		PlatformCampaignID: strings.TrimSpace(r.URL.Query().Get("platform_campaign_id")),
		PlatformAdGroupID:  strings.TrimSpace(r.URL.Query().Get("platform_ad_group_id")),
		PlatformAdID:       strings.TrimSpace(r.URL.Query().Get("platform_ad_id")),
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return biquerydomain.GameKPIQueryFilter{}, fmt.Errorf("invalid limit")
		}
		filter.Limit = limit
	}
	if dateFrom := strings.TrimSpace(r.URL.Query().Get("date_from")); dateFrom != "" {
		parsed, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			return biquerydomain.GameKPIQueryFilter{}, fmt.Errorf("invalid date_from")
		}
		filter.DateFrom = parsed
	}
	if dateTo := strings.TrimSpace(r.URL.Query().Get("date_to")); dateTo != "" {
		parsed, err := time.Parse("2006-01-02", dateTo)
		if err != nil {
			return biquerydomain.GameKPIQueryFilter{}, fmt.Errorf("invalid date_to")
		}
		filter.DateTo = parsed
	}
	return filter, nil
}

func uaFieldCatalog() []biquerydomain.UAFieldDefinition {
	return []biquerydomain.UAFieldDefinition{
		{Key: "platform", Label: "渠道/平台", Category: "dimension", Status: "available", Source: "ads connector", RelatedKeys: []string{"platform_account_id", "platform_campaign_id", "platform_ad_group_id", "platform_ad_id"}},
		{Key: "platform_account_id", Label: "账户 ID", Category: "dimension", Status: "available", Source: "ads connector"},
		{Key: "platform_campaign_id", Label: "Campaign ID", Category: "dimension", Status: "available", Source: "ads connector"},
		{Key: "platform_ad_group_id", Label: "Ad Group ID", Category: "dimension", Status: "available", Source: "ads connector"},
		{Key: "platform_ad_id", Label: "Ad ID", Category: "dimension", Status: "available", Source: "ads connector"},
		{Key: "stat_date", Label: "日期", Category: "dimension", Status: "available", Source: "ads connector"},
		{Key: "hour_of_day", Label: "小时/时段", Category: "dimension", Status: "planned", Source: "ads connector", Notes: "需补 segments.hour 或平台等价字段"},
		{Key: "device", Label: "广告设备", Category: "dimension", Status: "available", Source: "ads connector", Notes: "如 MOBILE/TABLET/DESKTOP"},
		{Key: "network", Label: "广告网络/版位", Category: "dimension", Status: "available", Source: "ads connector"},
		{Key: "age_range", Label: "年龄段", Category: "dimension", Status: "planned", Source: "ads connector"},
		{Key: "gender", Label: "性别", Category: "dimension", Status: "planned", Source: "ads connector"},
		{Key: "interest_tags", Label: "兴趣/行为标签", Category: "dimension", Status: "planned", Source: "ads connector"},
		{Key: "audience_type", Label: "受众类型", Category: "dimension", Status: "planned", Source: "ads connector", Notes: "自定义/相似/DMP 等"},
		{Key: "country", Label: "国家/地区", Category: "dimension", Status: "integration_ready", Source: "game kpi api", ExampleAPI: "/api/bi/game-kpis"},
		{Key: "os", Label: "操作系统", Category: "dimension", Status: "integration_ready", Source: "game kpi api"},
		{Key: "placement", Label: "版位", Category: "dimension", Status: "integration_ready", Source: "game kpi api"},
		{Key: "creative_id", Label: "素材 ID", Category: "dimension", Status: "integration_ready", Source: "game kpi api"},
		{Key: "creative_type", Label: "素材类型", Category: "dimension", Status: "integration_ready", Source: "game kpi api"},
		{Key: "optimization_goal", Label: "优化目标", Category: "dimension", Status: "integration_ready", Source: "game kpi api"},
		{Key: "bid_type", Label: "出价方式", Category: "dimension", Status: "integration_ready", Source: "game kpi api"},
		{Key: "targeting", Label: "定向标签", Category: "dimension", Status: "integration_ready", Source: "game kpi api"},
		{Key: "video_hook_rate_2s", Label: "2 秒钩子率", Category: "traffic", Status: "planned", Source: "ads connector"},
		{Key: "video_view_rate_3s", Label: "3 秒播放率", Category: "traffic", Status: "planned", Source: "ads connector"},
		{Key: "video_completion_rate_6s", Label: "6 秒完播率", Category: "traffic", Status: "planned", Source: "ads connector"},
		{Key: "video_avg_watch_time", Label: "平均播放时长", Category: "traffic", Status: "planned", Source: "ads connector"},
		{Key: "silent_completion_rate", Label: "静音完播率", Category: "traffic", Status: "planned", Source: "ads connector"},
		{Key: "impressions", Label: "展现量", Category: "traffic", Status: "available", Source: "ads connector"},
		{Key: "clicks", Label: "点击量", Category: "traffic", Status: "available", Source: "ads connector"},
		{Key: "ctr", Label: "点击率", Category: "traffic", Status: "available", Source: "derived"},
		{Key: "cpm", Label: "CPM", Category: "cost", Status: "available", Source: "derived"},
		{Key: "cpc", Label: "CPC", Category: "cost", Status: "available", Source: "derived"},
		{Key: "spend", Label: "花费", Category: "cost", Status: "available", Source: "ads connector"},
		{Key: "reach", Label: "覆盖人数", Category: "traffic", Status: "available", Source: "ads connector"},
		{Key: "frequency", Label: "频次", Category: "traffic", Status: "available", Source: "derived"},
		{Key: "conversions", Label: "平台转化", Category: "conversion", Status: "available", Source: "ads connector", Notes: "语义取决于投放优化目标"},
		{Key: "all_conversions", Label: "全量转化", Category: "conversion", Status: "available", Source: "ads connector"},
		{Key: "conversions_value", Label: "平台转化价值", Category: "revenue", Status: "available", Source: "ads connector"},
		{Key: "cost_per_conversion", Label: "CPA", Category: "cost", Status: "available", Source: "derived"},
		{Key: "roas", Label: "ROAS", Category: "revenue", Status: "available", Source: "derived"},
		{Key: "installs", Label: "安装量", Category: "funnel", Status: "integration_ready", Source: "game kpi api"},
		{Key: "cpi", Label: "CPI", Category: "cost", Status: "integration_ready", Source: "derived"},
		{Key: "activations", Label: "激活量", Category: "funnel", Status: "integration_ready", Source: "game kpi api"},
		{Key: "activation_rate", Label: "激活率", Category: "funnel", Status: "integration_ready", Source: "derived"},
		{Key: "registrations", Label: "注册量", Category: "funnel", Status: "integration_ready", Source: "game kpi api"},
		{Key: "cpr", Label: "CPR", Category: "cost", Status: "integration_ready", Source: "derived"},
		{Key: "tutorial_completions", Label: "完成新手教程", Category: "behavior", Status: "integration_ready", Source: "game kpi api"},
		{Key: "role_creations", Label: "创建角色", Category: "behavior", Status: "integration_ready", Source: "game kpi api"},
		{Key: "level_x_users", Label: "达到指定等级", Category: "behavior", Status: "integration_ready", Source: "game kpi api"},
		{Key: "purchasers", Label: "付费人数", Category: "revenue", Status: "integration_ready", Source: "game kpi api"},
		{Key: "payer_rate", Label: "付费率", Category: "revenue", Status: "integration_ready", Source: "derived"},
		{Key: "revenue_d1", Label: "D1 收入", Category: "revenue", Status: "integration_ready", Source: "game kpi api"},
		{Key: "revenue_d7", Label: "D7 收入", Category: "revenue", Status: "integration_ready", Source: "game kpi api"},
		{Key: "revenue_d30", Label: "D30 收入", Category: "revenue", Status: "integration_ready", Source: "game kpi api"},
		{Key: "ad_revenue", Label: "广告收入", Category: "revenue", Status: "integration_ready", Source: "game kpi api"},
		{Key: "total_revenue", Label: "总收入", Category: "revenue", Status: "integration_ready", Source: "game kpi api"},
		{Key: "arpu", Label: "ARPU", Category: "revenue", Status: "integration_ready", Source: "derived"},
		{Key: "arppu", Label: "ARPPU", Category: "revenue", Status: "integration_ready", Source: "derived"},
		{Key: "roi", Label: "ROI", Category: "revenue", Status: "integration_ready", Source: "derived"},
		{Key: "retention_d1", Label: "D1 留存", Category: "retention", Status: "integration_ready", Source: "game kpi api"},
		{Key: "retention_d3", Label: "D3 留存", Category: "retention", Status: "integration_ready", Source: "game kpi api"},
		{Key: "retention_d7", Label: "D7 留存", Category: "retention", Status: "integration_ready", Source: "game kpi api"},
		{Key: "retention_d30", Label: "D30 留存", Category: "retention", Status: "integration_ready", Source: "game kpi api"},
		{Key: "ltv_d7", Label: "D7 LTV", Category: "lifecycle", Status: "integration_ready", Source: "game kpi api"},
		{Key: "ltv_d30", Label: "D30 LTV", Category: "lifecycle", Status: "integration_ready", Source: "game kpi api"},
		{Key: "ltv_to_cpi_ratio", Label: "LTV/CPI", Category: "lifecycle", Status: "integration_ready", Source: "derived"},
		{Key: "avg_online_duration_seconds", Label: "在线时长", Category: "behavior", Status: "integration_ready", Source: "game kpi api"},
		{Key: "task_completion_rate", Label: "任务完成率", Category: "behavior", Status: "integration_ready", Source: "game kpi api"},
		{Key: "high_value_payer_ratio", Label: "高价值付费占比", Category: "revenue", Status: "integration_ready", Source: "game kpi api"},
		{Key: "landing_bounce_rate", Label: "落地页跳出率", Category: "behavior", Status: "planned", Source: "ads connector"},
		{Key: "fraud_install_ratio", Label: "作弊/异常安装占比", Category: "quality", Status: "planned", Source: "external anti-fraud api"},
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
