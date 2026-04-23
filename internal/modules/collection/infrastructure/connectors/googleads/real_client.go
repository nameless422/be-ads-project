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
	query := `SELECT campaign.id, campaign.name, campaign.status, campaign.advertising_channel_type, campaign_budget.amount_micros FROM campaign ORDER BY campaign.id LIMIT 50`
	chunks, err := c.searchStream(ctx, req, accessToken, query)
	if err != nil {
		return nil, err
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
				"campaign_budget":          microsToAmount(row.CampaignBudget.AmountMicros),
				"updated_at":               now,
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
	query := `SELECT ad_group.id, ad_group.name, ad_group.status, ad_group.cpc_bid_micros, campaign.id FROM ad_group ORDER BY ad_group.id LIMIT 50`
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
	query := `SELECT ad_group_ad.ad.id, ad_group_ad.ad.name, ad_group_ad.status, ad_group.id FROM ad_group_ad ORDER BY ad_group_ad.ad.id LIMIT 50`
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
	query := `SELECT campaign.id, segments.date, metrics.impressions, metrics.clicks, metrics.cost_micros, metrics.ctr, metrics.average_cpc, metrics.average_cpm, metrics.conversions FROM campaign WHERE segments.date DURING YESTERDAY ORDER BY campaign.id LIMIT 50`
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
			payload := map[string]any{
				"campaign_id": row.Campaign.ID,
				"segments": map[string]any{
					"date": row.Segments.Date,
				},
				"metrics": map[string]any{
					"impressions": row.Metrics.Impressions,
					"clicks":      row.Metrics.Clicks,
					"cost_micros": row.Metrics.CostMicros,
					"ctr":         row.Metrics.CTR,
					"average_cpc": microsToAmount(row.Metrics.AverageCPC),
					"average_cpm": microsToAmount(row.Metrics.AverageCPM),
					"conversions": row.Metrics.Conversions,
				},
			}
			record, err := marshalGoogleRawRecord(req, domain.ObjectTypeInsight, row.Campaign.ID, payload)
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
	Campaign struct {
		ResourceName           string `json:"resourceName"`
		ID                     string `json:"id"`
		Name                   string `json:"name"`
		Status                 string `json:"status"`
		AdvertisingChannelType string `json:"advertisingChannelType"`
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
		Status string `json:"status"`
		Ad     struct {
			ResourceName string `json:"resourceName"`
			ID           string `json:"id"`
			Name         string `json:"name"`
		} `json:"ad"`
	} `json:"adGroupAd"`
	Segments struct {
		Date string `json:"date"`
	} `json:"segments"`
	Metrics struct {
		Impressions string `json:"impressions"`
		Clicks      string `json:"clicks"`
		CostMicros  string `json:"costMicros"`
		CTR         string `json:"ctr"`
		AverageCPC  string `json:"averageCpc"`
		AverageCPM  string `json:"averageCpm"`
		Conversions string `json:"conversions"`
	} `json:"metrics"`
}
