package normalizer

import (
	"context"

	ads "be_ads_project/internal/shared/ads"
)

type GoogleAdsNormalizer struct{}

func NewGoogleAdsNormalizer() *GoogleAdsNormalizer {
	return &GoogleAdsNormalizer{}
}

func (n *GoogleAdsNormalizer) Platform() ads.Platform {
	return ads.PlatformGoogleAds
}

func (n *GoogleAdsNormalizer) Normalize(_ context.Context, records []ads.RawRecord) (*ads.NormalizedPayload, error) {
	result := &ads.NormalizedPayload{}
	for _, record := range records {
		switch record.ObjectType {
		case ads.ObjectTypeCampaign:
			var payload struct {
				Campaign struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Status string `json:"status"`
				} `json:"campaign"`
				AdvertisingChannelType string `json:"advertising_channel_type"`
				CampaignBudget         string `json:"campaign_budget"`
				UpdatedAt              string `json:"updated_at"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Campaigns = append(result.Campaigns, ads.StandardCampaign{
				Platform:           record.Platform,
				PlatformAccountID:  record.PlatformAccountID,
				PlatformCampaignID: payload.Campaign.ID,
				CampaignName:       payload.Campaign.Name,
				Status:             payload.Campaign.Status,
				Objective:          payload.AdvertisingChannelType,
				DailyBudget:        payload.CampaignBudget,
				UpdatedAt:          parseRFC3339Ptr(payload.UpdatedAt),
				RawPayload:         record.Payload,
			})
		case ads.ObjectTypeAdGroup:
			var payload struct {
				AdGroup struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Status string `json:"status"`
				} `json:"ad_group"`
				CampaignID string `json:"campaign_id"`
				CPCBid     string `json:"cpc_bid"`
				UpdatedAt  string `json:"updated_at"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.AdGroups = append(result.AdGroups, ads.StandardAdGroup{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				PlatformAdGroupID: payload.AdGroup.ID,
				PlatformParentID:  payload.CampaignID,
				AdGroupName:       payload.AdGroup.Name,
				Status:            payload.AdGroup.Status,
				BidStrategy:       payload.CPCBid,
				UpdatedAt:         parseRFC3339Ptr(payload.UpdatedAt),
				RawPayload:        record.Payload,
			})
		case ads.ObjectTypeAd:
			var payload struct {
				AdGroupAd struct {
					Status string `json:"status"`
					Ad     struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"ad"`
				} `json:"ad_group_ad"`
				AdGroupID string `json:"ad_group_id"`
				UpdatedAt string `json:"updated_at"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.Ads = append(result.Ads, ads.StandardAd{
				Platform:          record.Platform,
				PlatformAccountID: record.PlatformAccountID,
				PlatformAdID:      payload.AdGroupAd.Ad.ID,
				PlatformParentID:  payload.AdGroupID,
				AdName:            payload.AdGroupAd.Ad.Name,
				Status:            payload.AdGroupAd.Status,
				UpdatedAt:         parseRFC3339Ptr(payload.UpdatedAt),
				RawPayload:        record.Payload,
			})
		case ads.ObjectTypeInsight:
			var payload struct {
				CampaignID string `json:"campaign_id"`
				Segments   struct {
					Date string `json:"date"`
				} `json:"segments"`
				Metrics struct {
					Impressions string `json:"impressions"`
					Clicks      string `json:"clicks"`
					CostMicros  string `json:"cost_micros"`
					CTR         string `json:"ctr"`
					AverageCPC  string `json:"average_cpc"`
					AverageCPM  string `json:"average_cpm"`
					Conversions string `json:"conversions"`
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
				StatDate:          parseDate(payload.Segments.Date),
				Impressions:       parseInt64(payload.Metrics.Impressions),
				Clicks:            parseInt64(payload.Metrics.Clicks),
				Spend:             microsToDecimalString(payload.Metrics.CostMicros),
				CTR:               payload.Metrics.CTR,
				CPC:               payload.Metrics.AverageCPC,
				CPM:               payload.Metrics.AverageCPM,
				Conversions:       payload.Metrics.Conversions,
				RawPayload:        record.Payload,
			})
		}
	}
	return result, nil
}
