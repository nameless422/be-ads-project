package googleads

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"be_ads_project/internal/modules/collection/infrastructure/connectors/meta"
	"be_ads_project/internal/shared/ads"
)

const defaultTimeout = 20 * time.Second

type realClient struct {
	httpClient *http.Client
}

func newRealClient() *realClient {
	return &realClient{
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

func (c *realClient) fetch(ctx context.Context, req meta.FetchRequest) (*meta.FetchResult, error) {
	accessToken, err := c.refreshAccessToken(ctx, req.AccountContext.Credential)
	if err != nil {
		return nil, err
	}

	switch req.ObjectType {
	case domain.ObjectTypeCampaign:
		return c.fetchCampaigns(ctx, req, accessToken)
	case domain.ObjectTypeAdGroup:
		return c.fetchAdGroups(ctx, req, accessToken)
	case domain.ObjectTypeAd:
		return c.fetchAds(ctx, req, accessToken)
	case domain.ObjectTypeInsight:
		return c.fetchInsights(ctx, req, accessToken)
	case domain.ObjectTypeSearchTerm:
		return c.fetchSearchTerms(ctx, req, accessToken)
	default:
		return nil, fmt.Errorf("google ads real mode does not support object type %s", req.ObjectType)
	}
}

func (c *realClient) refreshAccessToken(ctx context.Context, credential domain.AccountCredential) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", credential.ClientID)
	if strings.TrimSpace(credential.ClientSecret) != "" {
		form.Set("client_secret", credential.ClientSecret)
	}
	form.Set("refresh_token", credential.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.googleapis.com/oauth2/v3/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("refresh google oauth token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("refresh google oauth token failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("google oauth token response missing access_token")
	}
	return payload.AccessToken, nil
}

func (c *realClient) fetchCampaigns(ctx context.Context, req meta.FetchRequest, accessToken string) (*meta.FetchResult, error) {
	query := `SELECT campaign.resource_name, campaign.id, campaign.name, campaign.status, campaign.advertising_channel_type, campaign.start_date, campaign.end_date, campaign.bidding_strategy_type, campaign_budget.amount_micros FROM campaign`
	if resourceNames, ok, err := c.changedResourceNames(ctx, req, accessToken, "CAMPAIGN", "change_status.campaign"); err != nil {
		return nil, err
	} else if ok {
		if len(resourceNames) == 0 {
			return buildFetchResult(req, nil), nil
		}
		query += " WHERE campaign.resource_name IN (" + quoteStrings(resourceNames) + ")"
	}
	query += " ORDER BY campaign.id LIMIT 200"
	chunks, err := c.searchStream(ctx, req, accessToken, query)
	if err != nil {
		return nil, err
	}
	diagnosticQuery := `SELECT campaign.id, segments.date, metrics.search_impression_share, metrics.search_top_impression_share, metrics.search_absolute_top_impression_share FROM campaign WHERE ` + googleDateRangePredicate(req) + ` ORDER BY campaign.id LIMIT 200`
	diagnosticChunks, err := c.searchStream(ctx, req, accessToken, diagnosticQuery)
	if err != nil {
		return nil, err
	}
	diagnosticsByCampaignID := make(map[string]map[string]any, 64)
	for _, chunk := range diagnosticChunks {
		for _, row := range chunk.Results {
			campaignID := firstNonEmpty(row.Campaign.ID, resourceIDFromResourceName(row.Campaign.ResourceName))
			if campaignID == "" || row.Segments.Date == "" {
				continue
			}
			diagnosticsByCampaignID[campaignID] = map[string]any{
				"date":                                 row.Segments.Date,
				"search_impression_share":              row.Metrics.SearchImpressionShare,
				"search_top_impression_share":          row.Metrics.SearchTopImpressionShare,
				"search_absolute_top_impression_share": row.Metrics.SearchAbsoluteTopImpressionShare,
			}
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	records := make([]domain.RawRecord, 0)
	for _, chunk := range chunks {
		for _, row := range chunk.Results {
			id := firstNonEmpty(row.Campaign.ID, resourceIDFromResourceName(row.Campaign.ResourceName))
			if id == "" {
				continue
			}
			payload := map[string]any{
				"campaign": map[string]any{
					"id":     id,
					"name":   row.Campaign.Name,
					"status": row.Campaign.Status,
				},
				"advertising_channel_type": row.Campaign.AdvertisingChannelType,
				"start_date":               row.Campaign.StartDate,
				"end_date":                 row.Campaign.EndDate,
				"bidding_strategy_type":    row.Campaign.BiddingStrategyType,
				"campaign_budget":          microsToAmount(row.CampaignBudget.AmountMicros),
				"updated_at":               now,
			}
			if diagnostics, ok := diagnosticsByCampaignID[id]; ok {
				payload["diagnostics"] = diagnostics
			}
			record, err := marshalGoogleRawRecord(req, domain.ObjectTypeCampaign, id, payload)
			if err != nil {
				return nil, err
			}
			records = append(records, record)
		}
	}
	return buildFetchResult(req, records), nil
}

func (c *realClient) fetchAdGroups(ctx context.Context, req meta.FetchRequest, accessToken string) (*meta.FetchResult, error) {
	query := `SELECT ad_group.resource_name, ad_group.id, ad_group.name, ad_group.status, ad_group.cpc_bid_micros, campaign.id FROM ad_group`
	if resourceNames, ok, err := c.changedResourceNames(ctx, req, accessToken, "AD_GROUP", "change_status.ad_group"); err != nil {
		return nil, err
	} else if ok {
		if len(resourceNames) == 0 {
			return buildFetchResult(req, nil), nil
		}
		query += " WHERE ad_group.resource_name IN (" + quoteStrings(resourceNames) + ")"
	}
	query += " ORDER BY ad_group.id LIMIT 50"
	chunks, err := c.searchStream(ctx, req, accessToken, query)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	records := make([]domain.RawRecord, 0)
	for _, chunk := range chunks {
		for _, row := range chunk.Results {
			id := firstNonEmpty(row.AdGroup.ID, resourceIDFromResourceName(row.AdGroup.ResourceName))
			if id == "" {
				continue
			}
			payload := map[string]any{
				"ad_group": map[string]any{
					"id":     id,
					"name":   row.AdGroup.Name,
					"status": row.AdGroup.Status,
				},
				"campaign_id": row.Campaign.ID,
				"cpc_bid":     microsToAmount(row.AdGroup.CPCBidMicros),
				"updated_at":  now,
			}
			record, err := marshalGoogleRawRecord(req, domain.ObjectTypeAdGroup, id, payload)
			if err != nil {
				return nil, err
			}
			records = append(records, record)
		}
	}
	return buildFetchResult(req, records), nil
}

func (c *realClient) fetchAds(ctx context.Context, req meta.FetchRequest, accessToken string) (*meta.FetchResult, error) {
	query := `SELECT ad_group_ad.resource_name, ad_group_ad.ad.id, ad_group_ad.ad.name, ad_group_ad.status, ad_group.id FROM ad_group_ad`
	if resourceNames, ok, err := c.changedResourceNames(ctx, req, accessToken, "AD_GROUP_AD", "change_status.ad_group_ad"); err != nil {
		return nil, err
	} else if ok {
		if len(resourceNames) == 0 {
			return buildFetchResult(req, nil), nil
		}
		query += " WHERE ad_group_ad.resource_name IN (" + quoteStrings(resourceNames) + ")"
	}
	query += " ORDER BY ad_group_ad.ad.id LIMIT 50"
	chunks, err := c.searchStream(ctx, req, accessToken, query)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	records := make([]domain.RawRecord, 0)
	for _, chunk := range chunks {
		for _, row := range chunk.Results {
			id := firstNonEmpty(row.AdGroupAd.Ad.ID, resourceIDFromResourceName(row.AdGroupAd.Ad.ResourceName))
			if id == "" {
				continue
			}
			payload := map[string]any{
				"ad_group_ad": map[string]any{
					"ad": map[string]any{
						"id":   id,
						"name": row.AdGroupAd.Ad.Name,
					},
					"status": row.AdGroupAd.Status,
				},
				"ad_group_id": row.AdGroup.ID,
				"updated_at":  now,
			}
			record, err := marshalGoogleRawRecord(req, domain.ObjectTypeAd, id, payload)
			if err != nil {
				return nil, err
			}
			records = append(records, record)
		}
	}
	return buildFetchResult(req, records), nil
}

func (c *realClient) fetchInsights(ctx context.Context, req meta.FetchRequest, accessToken string) (*meta.FetchResult, error) {
	query := `SELECT campaign.id, ad_group.id, ad_group_ad.ad.id, segments.date, segments.device, segments.ad_network_type, metrics.impressions, metrics.clicks, metrics.cost_micros, metrics.ctr, metrics.average_cpc, metrics.average_cpm, metrics.conversions, metrics.all_conversions, metrics.conversions_value, metrics.cost_per_conversion, metrics.cost_per_all_conversions FROM ad_group_ad WHERE ` + googleDateRangePredicate(req) + ` ORDER BY campaign.id, ad_group.id, ad_group_ad.ad.id LIMIT 1000`
	chunks, err := c.searchStream(ctx, req, accessToken, query)
	if err != nil {
		return nil, err
	}

	records := make([]domain.RawRecord, 0)
	for _, chunk := range chunks {
		for _, row := range chunk.Results {
			if row.Campaign.ID == "" {
				continue
			}
			resourceID := firstNonEmpty(row.AdGroupAd.Ad.ID, row.AdGroup.ID, row.Campaign.ID)
			payload := map[string]any{
				"campaign_id": row.Campaign.ID,
				"ad_group_id": row.AdGroup.ID,
				"ad_id":       row.AdGroupAd.Ad.ID,
				"segments": map[string]any{
					"date":            row.Segments.Date,
					"device":          row.Segments.Device,
					"ad_network_type": row.Segments.AdNetworkType,
				},
				"metrics": map[string]any{
					"impressions":              row.Metrics.Impressions,
					"clicks":                   row.Metrics.Clicks,
					"cost_micros":              row.Metrics.CostMicros,
					"ctr":                      row.Metrics.CTR,
					"average_cpc":              microsToAmount(row.Metrics.AverageCPC),
					"average_cpm":              microsToAmount(row.Metrics.AverageCPM),
					"conversions":              row.Metrics.Conversions,
					"all_conversions":          row.Metrics.AllConversions,
					"conversions_value":        row.Metrics.ConversionsValue,
					"cost_per_conversion":      microsToAmount(row.Metrics.CostPerConversion),
					"cost_per_all_conversions": microsToAmount(row.Metrics.CostPerAllConversions),
				},
			}
			record, err := marshalGoogleRawRecord(req, domain.ObjectTypeInsight, resourceID, payload)
			if err != nil {
				return nil, err
			}
			records = append(records, record)
		}
	}
	return buildFetchResult(req, records), nil
}

func (c *realClient) fetchSearchTerms(ctx context.Context, req meta.FetchRequest, accessToken string) (*meta.FetchResult, error) {
	query := `SELECT campaign.id, ad_group.id, search_term_view.search_term, segments.date, segments.search_term_match_type, metrics.impressions, metrics.clicks, metrics.cost_micros, metrics.conversions, metrics.conversions_value FROM search_term_view WHERE ` + googleDateRangePredicate(req) + ` ORDER BY campaign.id, ad_group.id, search_term_view.search_term LIMIT 1000`
	chunks, err := c.searchStream(ctx, req, accessToken, query)
	if err != nil {
		return nil, err
	}

	records := make([]domain.RawRecord, 0)
	for _, chunk := range chunks {
		for _, row := range chunk.Results {
			searchTerm := strings.TrimSpace(row.SearchTermView.SearchTerm)
			if row.Campaign.ID == "" || searchTerm == "" {
				continue
			}
			resourceID := firstNonEmpty(searchTerm, row.AdGroup.ID, row.Campaign.ID)
			payload := map[string]any{
				"campaign_id": row.Campaign.ID,
				"ad_group_id": row.AdGroup.ID,
				"search_term": searchTerm,
				"segments": map[string]any{
					"date":                   row.Segments.Date,
					"search_term_match_type": row.Segments.SearchTermMatchType,
				},
				"metrics": map[string]any{
					"impressions":       row.Metrics.Impressions,
					"clicks":            row.Metrics.Clicks,
					"cost_micros":       row.Metrics.CostMicros,
					"conversions":       row.Metrics.Conversions,
					"conversions_value": row.Metrics.ConversionsValue,
				},
			}
			record, err := marshalGoogleRawRecord(req, domain.ObjectTypeSearchTerm, resourceID, payload)
			if err != nil {
				return nil, err
			}
			records = append(records, record)
		}
	}
	return buildFetchResult(req, records), nil
}

func (c *realClient) searchStream(ctx context.Context, req meta.FetchRequest, accessToken, query string) ([]googleAdsSearchChunk, error) {
	credential := req.AccountContext.Credential
	customerID := normalizeCustomerID(firstNonEmpty(credential.ExtraConfig["customer_id"], req.AccountContext.Account.AccountID))
	if customerID == "" {
		return nil, fmt.Errorf("google ads customer_id is required")
	}

	apiVersion := firstNonEmpty(credential.ExtraConfig["api_version"], "v20")
	endpoint := fmt.Sprintf("https://googleads.googleapis.com/%s/customers/%s/googleAds:searchStream", apiVersion, customerID)

	body, err := json.Marshal(map[string]any{
		"query": query,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("developer-token", credential.ExtraConfig["developer_token"])

	if loginCustomerID := normalizeCustomerID(credential.ExtraConfig["login_customer_id"]); loginCustomerID != "" {
		httpReq.Header.Set("login-customer-id", loginCustomerID)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("google ads searchStream %s: %w", req.ObjectType, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google ads searchStream %s failed status=%d body=%s", req.ObjectType, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var chunks []googleAdsSearchChunk
	if err := json.Unmarshal(responseBody, &chunks); err != nil {
		return nil, fmt.Errorf("decode google ads searchStream response: %w", err)
	}
	return chunks, nil
}

func marshalGoogleRawRecord(req meta.FetchRequest, objectType domain.ObjectType, resourceID string, payload map[string]any) (domain.RawRecord, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.RawRecord{}, err
	}
	return domain.RawRecord{
		Platform:          req.AccountContext.Account.Platform,
		PlatformAccountID: req.AccountContext.Account.ID,
		ObjectType:        objectType,
		ResourceID:        resourceID,
		Payload:           body,
	}, nil
}

func buildFetchResult(req meta.FetchRequest, records []domain.RawRecord) *meta.FetchResult {
	return &meta.FetchResult{
		RawRecords:        records,
		NextPageCursor:    "",
		NextTimeWatermark: req.EndTime,
		HasMore:           false,
	}
}

func (c *realClient) changedResourceNames(ctx context.Context, req meta.FetchRequest, accessToken, resourceType, selectField string) ([]string, bool, error) {
	start := incrementalStartTime(req)
	end := requestEndTime(req)
	if start == nil || end.IsZero() {
		return nil, false, nil
	}

	query := fmt.Sprintf(
		`SELECT %s FROM change_status WHERE change_status.resource_type = %s AND change_status.last_change_date_time >= '%s' AND change_status.last_change_date_time <= '%s' ORDER BY change_status.last_change_date_time LIMIT 10000`,
		selectField,
		resourceType,
		start.UTC().Format("2006-01-02 15:04:05"),
		end.UTC().Format("2006-01-02 15:04:05"),
	)
	chunks, err := c.searchStream(ctx, req, accessToken, query)
	if err != nil {
		return nil, false, err
	}

	seen := make(map[string]struct{}, 128)
	items := make([]string, 0, 128)
	for _, chunk := range chunks {
		for _, row := range chunk.Results {
			value := firstNonEmpty(
				row.ChangeStatus.Campaign,
				row.ChangeStatus.AdGroup,
				row.ChangeStatus.AdGroupAd,
			)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			items = append(items, value)
		}
	}
	return items, true, nil
}

func googleDateRangePredicate(req meta.FetchRequest) string {
	start := incrementalStartTime(req)
	end := requestEndTime(req)
	if end.IsZero() {
		end = time.Now().UTC()
	}
	if start == nil {
		return "segments.date DURING YESTERDAY"
	}

	startDate := start.UTC().Format("2006-01-02")
	endDate := end.UTC().Format("2006-01-02")
	return fmt.Sprintf("segments.date BETWEEN '%s' AND '%s'", startDate, endDate)
}

func incrementalStartTime(req meta.FetchRequest) *time.Time {
	if req.Checkpoint.TimeWatermark == nil {
		return nil
	}
	start := req.Checkpoint.TimeWatermark.UTC()
	if req.Checkpoint.LookbackWindow > 0 {
		start = start.Add(-req.Checkpoint.LookbackWindow)
	}
	return &start
}

func requestEndTime(req meta.FetchRequest) time.Time {
	if req.EndTime != nil {
		return req.EndTime.UTC()
	}
	return time.Now().UTC()
}

func quoteStrings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := strings.ReplaceAll(value, `'`, `\'`)
		quoted = append(quoted, fmt.Sprintf("'%s'", cleaned))
	}
	return strings.Join(quoted, ",")
}

func normalizeCustomerID(raw string) string {
	replacer := strings.NewReplacer("-", "", " ", "")
	return replacer.Replace(strings.TrimSpace(raw))
}

func microsToAmount(raw string) string {
	if raw == "" {
		return "0"
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw
	}
	return strconv.FormatFloat(value/1_000_000, 'f', 2, 64)
}

func resourceIDFromResourceName(resourceName string) string {
	if resourceName == "" {
		return ""
	}
	parts := strings.Split(resourceName, "/")
	return parts[len(parts)-1]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type googleAdsSearchChunk struct {
	Results []googleAdsRow `json:"results"`
}

type googleAdsRow struct {
	ChangeStatus struct {
		Campaign  string `json:"campaign"`
		AdGroup   string `json:"adGroup"`
		AdGroupAd string `json:"adGroupAd"`
	} `json:"changeStatus"`
	Campaign struct {
		ResourceName           string `json:"resourceName"`
		ID                     string `json:"id"`
		Name                   string `json:"name"`
		Status                 string `json:"status"`
		AdvertisingChannelType string `json:"advertisingChannelType"`
		StartDate              string `json:"startDate"`
		EndDate                string `json:"endDate"`
		BiddingStrategyType    string `json:"biddingStrategyType"`
	} `json:"campaign"`
	CampaignBudget struct {
		ResourceName string `json:"resourceName"`
		AmountMicros string `json:"amountMicros"`
	} `json:"campaignBudget"`
	AdGroup struct {
		ResourceName string `json:"resourceName"`
		ID           string `json:"id"`
		Name         string `json:"name"`
		Status       string `json:"status"`
		CPCBidMicros string `json:"cpcBidMicros"`
	} `json:"adGroup"`
	AdGroupAd struct {
		ResourceName string `json:"resourceName"`
		Status       string `json:"status"`
		Ad           struct {
			ResourceName string `json:"resourceName"`
			ID           string `json:"id"`
			Name         string `json:"name"`
		} `json:"ad"`
	} `json:"adGroupAd"`
	SearchTermView struct {
		SearchTerm string `json:"searchTerm"`
	} `json:"searchTermView"`
	Segments struct {
		Date                string `json:"date"`
		Device              string `json:"device"`
		AdNetworkType       string `json:"adNetworkType"`
		SearchTermMatchType string `json:"searchTermMatchType"`
	} `json:"segments"`
	Metrics struct {
		Impressions                      string `json:"impressions"`
		Clicks                           string `json:"clicks"`
		CostMicros                       string `json:"costMicros"`
		CTR                              string `json:"ctr"`
		AverageCPC                       string `json:"averageCpc"`
		AverageCPM                       string `json:"averageCpm"`
		Conversions                      string `json:"conversions"`
		AllConversions                   string `json:"allConversions"`
		ConversionsValue                 string `json:"conversionsValue"`
		CostPerConversion                string `json:"costPerConversion"`
		CostPerAllConversions            string `json:"costPerAllConversions"`
		SearchImpressionShare            string `json:"searchImpressionShare"`
		SearchTopImpressionShare         string `json:"searchTopImpressionShare"`
		SearchAbsoluteTopImpressionShare string `json:"searchAbsoluteTopImpressionShare"`
	} `json:"metrics"`
}
