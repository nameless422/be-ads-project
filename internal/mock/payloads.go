package mock

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"be_ads_project/internal/shared/ads"
)

func RawRecords(account domain.PlatformAccount, objectType domain.ObjectType) ([]domain.RawRecord, error) {
	switch account.Platform {
	case domain.PlatformFacebook:
		return facebookRawRecords(account, objectType)
	case domain.PlatformGoogleAds:
		return googleAdsRawRecords(account, objectType)
	case domain.PlatformTikTokAds:
		return tikTokRawRecords(account, objectType)
	default:
		return nil, fmt.Errorf("unsupported platform %s", account.Platform)
	}
}

func facebookRawRecords(account domain.PlatformAccount, objectType domain.ObjectType) ([]domain.RawRecord, error) {
	now := time.Now().UTC()
	switch objectType {
	case domain.ObjectTypeCampaign:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"id":               account.AccountID + "-cmp-001",
				"name":             "FB Prospecting 7D",
				"effective_status": "ACTIVE",
				"objective":        "CONVERSIONS",
				"buying_type":      "AUCTION",
				"daily_budget":     "100.00",
				"currency":         account.Currency,
				"updated_time":     now.Format(time.RFC3339),
			},
			{
				"id":               account.AccountID + "-cmp-002",
				"name":             "FB Retargeting 30D",
				"effective_status": "PAUSED",
				"objective":        "SALES",
				"buying_type":      "AUCTION",
				"daily_budget":     "60.00",
				"currency":         account.Currency,
				"updated_time":     now.Add(-20 * time.Minute).Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeAdGroup:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"id":           account.AccountID + "-adset-001",
				"campaign_id":  account.AccountID + "-cmp-001",
				"name":         "FB Broad Audience",
				"status":       "ACTIVE",
				"bid_strategy": "LOWEST_COST_WITHOUT_CAP",
				"daily_budget": "100.00",
				"updated_time": now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeAd:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"id":           account.AccountID + "-ad-001",
				"adset_id":     account.AccountID + "-adset-001",
				"name":         "FB Creative A",
				"status":       "ACTIVE",
				"creative":     map[string]any{"id": "creative-a", "name": "Creative A"},
				"updated_time": now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeInsight:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"campaign_id": account.AccountID + "-cmp-001",
				"date_start":  now.AddDate(0, 0, -1).Format("2006-01-02"),
				"impressions": "12000",
				"clicks":      "320",
				"spend":       "185.45",
				"ctr":         "2.67",
				"cpc":         "0.58",
				"cpm":         "15.45",
				"conversions": "23",
				"reach":       "8800",
			},
		})
	default:
		return nil, fmt.Errorf("unsupported object type %s", objectType)
	}
}

func googleAdsRawRecords(account domain.PlatformAccount, objectType domain.ObjectType) ([]domain.RawRecord, error) {
	now := time.Now().UTC()
	switch objectType {
	case domain.ObjectTypeCampaign:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"campaign": map[string]any{
					"id":     account.AccountID + "-cmp-001",
					"name":   "Google Search Brand",
					"status": "ENABLED",
				},
				"advertising_channel_type": "SEARCH",
				"campaign_budget":          "200.00",
				"updated_at":               now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeAdGroup:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"ad_group": map[string]any{
					"id":     account.AccountID + "-ag-001",
					"name":   "Brand Exact",
					"status": "ENABLED",
				},
				"campaign_id": account.AccountID + "-cmp-001",
				"cpc_bid":     "1.20",
				"updated_at":  now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeAd:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"ad_group_ad": map[string]any{
					"ad": map[string]any{
						"id":   account.AccountID + "-ad-001",
						"name": "Brand RSA 1",
					},
					"status": "ENABLED",
				},
				"ad_group_id": account.AccountID + "-ag-001",
				"updated_at":  now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeInsight:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"campaign_id": account.AccountID + "-cmp-001",
				"segments": map[string]any{
					"date": now.AddDate(0, 0, -1).Format("2006-01-02"),
				},
				"metrics": map[string]any{
					"impressions": "8600",
					"clicks":      "410",
					"cost_micros": "125450000",
					"ctr":         "4.76",
					"average_cpc": "0.31",
					"average_cpm": "14.58",
					"conversions": "18",
				},
			},
		})
	default:
		return nil, fmt.Errorf("unsupported object type %s", objectType)
	}
}

func tikTokRawRecords(account domain.PlatformAccount, objectType domain.ObjectType) ([]domain.RawRecord, error) {
	now := time.Now().UTC()
	switch objectType {
	case domain.ObjectTypeCampaign:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"campaign_id":      account.AccountID + "-cmp-001",
				"campaign_name":    "TikTok Install Boost",
				"operation_status": "ENABLE",
				"objective_type":   "APP_PROMOTION",
				"budget_mode":      "BUDGET_MODE_DAY",
				"budget":           "150.00",
				"modify_time":      now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeAdGroup:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"adgroup_id":       account.AccountID + "-ag-001",
				"campaign_id":      account.AccountID + "-cmp-001",
				"adgroup_name":     "Android Install Group",
				"operation_status": "ENABLE",
				"bid_type":         "BID_TYPE_NO_BID",
				"budget":           "80.00",
				"modify_time":      now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeAd:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"ad_id":            account.AccountID + "-ad-001",
				"adgroup_id":       account.AccountID + "-ag-001",
				"ad_name":          "UGC Video A",
				"operation_status": "ENABLE",
				"creative_id":      "tt-creative-1",
				"modify_time":      now.Format(time.RFC3339),
			},
		})
	case domain.ObjectTypeInsight:
		return marshalRecords(account, objectType, []map[string]any{
			{
				"campaign_id":   account.AccountID + "-cmp-001",
				"stat_time_day": now.AddDate(0, 0, -1).Format("2006-01-02"),
				"metrics": map[string]any{
					"show_cnt":    "14300",
					"click_cnt":   "520",
					"stat_cost":   "212.80",
					"ctr":         "3.64",
					"cpc":         "0.41",
					"cpm":         "14.88",
					"convert_cnt": "31",
					"reach":       "10050",
				},
			},
		})
	default:
		return nil, fmt.Errorf("unsupported object type %s", objectType)
	}
}

func marshalRecords(account domain.PlatformAccount, objectType domain.ObjectType, items []map[string]any) ([]domain.RawRecord, error) {
	records := make([]domain.RawRecord, 0, len(items))
	for idx, item := range items {
		payload, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		records = append(records, domain.RawRecord{
			Platform:          account.Platform,
			PlatformAccountID: account.ID,
			ObjectType:        objectType,
			ResourceID:        resourceID(item, idx),
			Payload:           payload,
		})
	}
	return records, nil
}

func resourceID(item map[string]any, idx int) string {
	for _, key := range []string{"id", "campaign_id", "adgroup_id", "ad_id"} {
		if value, ok := item[key]; ok {
			return fmt.Sprintf("%v", value)
		}
	}
	return "resource-" + strconv.Itoa(idx+1)
}
