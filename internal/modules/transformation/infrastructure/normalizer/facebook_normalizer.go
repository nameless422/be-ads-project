package normalizer

import (
	"context"

	ads "be_ads_project/internal/shared/ads"
)

type FacebookNormalizer struct{}

func NewFacebookNormalizer() *FacebookNormalizer {
	return &FacebookNormalizer{}
}

func (n *FacebookNormalizer) Platform() ads.Platform {
	return ads.PlatformFacebook
}

func (n *FacebookNormalizer) Normalize(_ context.Context, records []ads.RawRecord) (*ads.NormalizedPayload, error) {
	result := &ads.NormalizedPayload{}
	for _, record := range records {
		switch record.ObjectType {
		case ads.ObjectTypeCampaign:
			var payload struct {
				ID              string `json:"id"`
				Name            string `json:"name"`
				EffectiveStatus string `json:"effective_status"`
				Objective       string `json:"objective"`
				BuyingType      string `json:"buying_type"`
				DailyBudget     string `json:"daily_budget"`
				Currency        string `json:"currency"`
				UpdatedTime     string `json:"updated_time"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Campaigns = append(result.Campaigns, ads.StandardCampaign{
				Platform:           record.Platform,
				PlatformAccountID:  record.PlatformAccountID,
				PlatformCampaignID: payload.ID,
				CampaignName:       payload.Name,
				Status:             payload.EffectiveStatus,
				Objective:          payload.Objective,
				BuyingType:         payload.BuyingType,
				DailyBudget:        payload.DailyBudget,
				Currency:           payload.Currency,
				UpdatedAt:          parseRFC3339Ptr(payload.UpdatedTime),
				RawPayload:         record.Payload,
			})
		case ads.ObjectTypeAdGroup:
			var payload struct {
				ID          string `json:"id"`
				CampaignID  string `json:"campaign_id"`
				Name        string `json:"name"`
				Status      string `json:"status"`
				BidStrategy string `json:"bid_strategy"`
				DailyBudget string `json:"daily_budget"`
				UpdatedTime string `json:"updated_time"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.AdGroups = append(result.AdGroups, ads.StandardAdGroup{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				PlatformAdGroupID: payload.ID,
				PlatformParentID:  payload.CampaignID,
				AdGroupName:       payload.Name,
				Status:            payload.Status,
				BidStrategy:       payload.BidStrategy,
				DailyBudget:       payload.DailyBudget,
				UpdatedAt:         parseRFC3339Ptr(payload.UpdatedTime),
				RawPayload:        record.Payload,
			})
		case ads.ObjectTypeAd:
			var payload struct {
				ID          string `json:"id"`
				AdSetID     string `json:"adset_id"`
				Name        string `json:"name"`
				Status      string `json:"status"`
				UpdatedTime string `json:"updated_time"`
				Creative    struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"creative"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Ads = append(result.Ads, ads.StandardAd{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				PlatformAdID:      payload.ID,
				PlatformParentID:  payload.AdSetID,
				AdName:            payload.Name,
				Status:            payload.Status,
				CreativeID:        payload.Creative.ID,
				CreativeName:      payload.Creative.Name,
				UpdatedAt:         parseRFC3339Ptr(payload.UpdatedTime),
				RawPayload:        record.Payload,
			})
		case ads.ObjectTypeInsight:
			var payload struct {
				CampaignID  string `json:"campaign_id"`
				DateStart   string `json:"date_start"`
				Impressions string `json:"impressions"`
				Clicks      string `json:"clicks"`
				Spend       string `json:"spend"`
				CTR         string `json:"ctr"`
				CPC         string `json:"cpc"`
				CPM         string `json:"cpm"`
				Conversions string `json:"conversions"`
				Reach       string `json:"reach"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Insights = append(result.Insights, ads.StandardInsight{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				EntityLevel:       ads.ObjectTypeCampaign,
				EntityID:          payload.CampaignID,
				StatDate:          parseDate(payload.DateStart),
				Impressions:       parseInt64(payload.Impressions),
				Clicks:            parseInt64(payload.Clicks),
				Spend:             payload.Spend,
				CTR:               payload.CTR,
				CPC:               payload.CPC,
				CPM:               payload.CPM,
				Conversions:       payload.Conversions,
				Reach:             parseInt64(payload.Reach),
				RawPayload:        record.Payload,
			})
		}
	}
	return result, nil
}
