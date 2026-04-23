package normalizer

import (
	"context"

	ads "be_ads_project/internal/shared/ads"
)

type TikTokNormalizer struct{}

func NewTikTokNormalizer() *TikTokNormalizer {
	return &TikTokNormalizer{}
}

func (n *TikTokNormalizer) Platform() ads.Platform {
	return ads.PlatformTikTokAds
}

func (n *TikTokNormalizer) Normalize(_ context.Context, records []ads.RawRecord) (*ads.NormalizedPayload, error) {
	result := &ads.NormalizedPayload{}
	for _, record := range records {
		switch record.ObjectType {
		case ads.ObjectTypeCampaign:
			var payload struct {
				CampaignID      string `json:"campaign_id"`
				CampaignName    string `json:"campaign_name"`
				OperationStatus string `json:"operation_status"`
				ObjectiveType   string `json:"objective_type"`
				BudgetMode      string `json:"budget_mode"`
				Budget          string `json:"budget"`
				ModifyTime      string `json:"modify_time"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Campaigns = append(result.Campaigns, ads.StandardCampaign{
				Platform:           record.Platform,
				PlatformAccountID:  record.PlatformAccountID,
				PlatformCampaignID: payload.CampaignID,
				CampaignName:       payload.CampaignName,
				Status:             payload.OperationStatus,
				Objective:          payload.ObjectiveType,
				BuyingType:         payload.BudgetMode,
				DailyBudget:        payload.Budget,
				UpdatedAt:          parseRFC3339Ptr(payload.ModifyTime),
				RawPayload:         record.Payload,
			})
		case ads.ObjectTypeAdGroup:
			var payload struct {
				AdGroupID       string `json:"adgroup_id"`
				CampaignID      string `json:"campaign_id"`
				AdGroupName     string `json:"adgroup_name"`
				OperationStatus string `json:"operation_status"`
				BidType         string `json:"bid_type"`
				Budget          string `json:"budget"`
				ModifyTime      string `json:"modify_time"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.AdGroups = append(result.AdGroups, ads.StandardAdGroup{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				PlatformAdGroupID: payload.AdGroupID,
				PlatformParentID:  payload.CampaignID,
				AdGroupName:       payload.AdGroupName,
				Status:            payload.OperationStatus,
				BidStrategy:       payload.BidType,
				DailyBudget:       payload.Budget,
				UpdatedAt:         parseRFC3339Ptr(payload.ModifyTime),
				RawPayload:        record.Payload,
			})
		case ads.ObjectTypeAd:
			var payload struct {
				AdID            string `json:"ad_id"`
				AdGroupID       string `json:"adgroup_id"`
				AdName          string `json:"ad_name"`
				OperationStatus string `json:"operation_status"`
				CreativeID      string `json:"creative_id"`
				ModifyTime      string `json:"modify_time"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Ads = append(result.Ads, ads.StandardAd{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				PlatformAdID:      payload.AdID,
				PlatformParentID:  payload.AdGroupID,
				AdName:            payload.AdName,
				Status:            payload.OperationStatus,
				CreativeID:        payload.CreativeID,
				UpdatedAt:         parseRFC3339Ptr(payload.ModifyTime),
				RawPayload:        record.Payload,
			})
		case ads.ObjectTypeInsight:
			var payload struct {
				CampaignID  string `json:"campaign_id"`
				StatTimeDay string `json:"stat_time_day"`
				Metrics     struct {
					ShowCnt    string `json:"show_cnt"`
					ClickCnt   string `json:"click_cnt"`
					StatCost   string `json:"stat_cost"`
					CTR        string `json:"ctr"`
					CPC        string `json:"cpc"`
					CPM        string `json:"cpm"`
					ConvertCnt string `json:"convert_cnt"`
					Reach      string `json:"reach"`
				} `json:"metrics"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Insights = append(result.Insights, ads.StandardInsight{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				EntityLevel:       ads.ObjectTypeCampaign,
				EntityID:          payload.CampaignID,
				StatDate:          parseDate(payload.StatTimeDay),
				Impressions:       parseInt64(payload.Metrics.ShowCnt),
				Clicks:            parseInt64(payload.Metrics.ClickCnt),
				Spend:             payload.Metrics.StatCost,
				CTR:               payload.Metrics.CTR,
				CPC:               payload.Metrics.CPC,
				CPM:               payload.Metrics.CPM,
				Conversions:       payload.Metrics.ConvertCnt,
				Reach:             parseInt64(payload.Metrics.Reach),
				RawPayload:        record.Payload,
			})
		}
	}
	return result, nil
}
