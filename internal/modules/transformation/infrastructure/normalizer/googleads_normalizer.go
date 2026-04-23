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
				StartDate              string `json:"start_date"`
				EndDate                string `json:"end_date"`
				BiddingStrategyType    string `json:"bidding_strategy_type"`
				CampaignBudget         string `json:"campaign_budget"`
				UpdatedAt              string `json:"updated_at"`
				Diagnostics            struct {
					Date                             string `json:"date"`
					SearchImpressionShare            string `json:"search_impression_share"`
					SearchTopImpressionShare         string `json:"search_top_impression_share"`
					SearchAbsoluteTopImpressionShare string `json:"search_absolute_top_impression_share"`
				} `json:"diagnostics"`
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
				BuyingType:         payload.BiddingStrategyType,
				BiddingStrategy:    payload.BiddingStrategyType,
				DailyBudget:        payload.CampaignBudget,
				StartTime:          parseDatePtr(payload.StartDate),
				EndTime:            parseDatePtr(payload.EndDate),
				UpdatedAt:          parseRFC3339Ptr(payload.UpdatedAt),
				RawPayload:         record.Payload,
			})
			if payload.Diagnostics.Date != "" {
				result.CampaignDiagnostics = append(result.CampaignDiagnostics, ads.StandardCampaignDiagnostic{
					Platform:                         record.Platform,
					PlatformAccountID:                record.PlatformAccountID,
					PlatformCampaignID:               payload.Campaign.ID,
					StatDate:                         parseDate(payload.Diagnostics.Date),
					SearchImpressionShare:            payload.Diagnostics.SearchImpressionShare,
					SearchTopImpressionShare:         payload.Diagnostics.SearchTopImpressionShare,
					SearchAbsoluteTopImpressionShare: payload.Diagnostics.SearchAbsoluteTopImpressionShare,
					RawPayload:                       record.Payload,
				})
			}
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
				AdGroupID  string `json:"ad_group_id"`
				AdID       string `json:"ad_id"`
				Segments   struct {
					Date          string `json:"date"`
					Device        string `json:"device"`
					AdNetworkType string `json:"ad_network_type"`
				} `json:"segments"`
				Metrics struct {
					Impressions           string `json:"impressions"`
					Clicks                string `json:"clicks"`
					CostMicros            string `json:"cost_micros"`
					CTR                   string `json:"ctr"`
					AverageCPC            string `json:"average_cpc"`
					AverageCPM            string `json:"average_cpm"`
					Conversions           string `json:"conversions"`
					AllConversions        string `json:"all_conversions"`
					ConversionsValue      string `json:"conversions_value"`
					CostPerConversion     string `json:"cost_per_conversion"`
					CostPerAllConversions string `json:"cost_per_all_conversions"`
				} `json:"metrics"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			entityLevel := ads.ObjectTypeCampaign
			entityID := payload.CampaignID
			if payload.AdID != "" {
				entityLevel = ads.ObjectTypeAd
				entityID = payload.AdID
			} else if payload.AdGroupID != "" {
				entityLevel = ads.ObjectTypeAdGroup
				entityID = payload.AdGroupID
			}
			result.Insights = append(result.Insights, ads.StandardInsight{
				Platform:              record.Platform,
				PlatformAccountID:     record.PlatformAccountID,
				PlatformCampaignID:    payload.CampaignID,
				EntityLevel:           entityLevel,
				EntityID:              entityID,
				PlatformAdGroupID:     payload.AdGroupID,
				PlatformAdID:          payload.AdID,
				StatDate:              parseDate(payload.Segments.Date),
				Device:                payload.Segments.Device,
				Network:               payload.Segments.AdNetworkType,
				Impressions:           parseInt64(payload.Metrics.Impressions),
				Clicks:                parseInt64(payload.Metrics.Clicks),
				Spend:                 microsToDecimalString(payload.Metrics.CostMicros),
				CTR:                   payload.Metrics.CTR,
				CPC:                   payload.Metrics.AverageCPC,
				CPM:                   payload.Metrics.AverageCPM,
				Conversions:           payload.Metrics.Conversions,
				AllConversions:        payload.Metrics.AllConversions,
				ConversionsValue:      payload.Metrics.ConversionsValue,
				CostPerConversion:     payload.Metrics.CostPerConversion,
				CostPerAllConversions: payload.Metrics.CostPerAllConversions,
				RawPayload:            record.Payload,
			})
		case ads.ObjectTypeSearchTerm:
			var payload struct {
				CampaignID string `json:"campaign_id"`
				AdGroupID  string `json:"ad_group_id"`
				SearchTerm string `json:"search_term"`
				Segments   struct {
					Date                string `json:"date"`
					SearchTermMatchType string `json:"search_term_match_type"`
				} `json:"segments"`
				Metrics struct {
					Impressions      string `json:"impressions"`
					Clicks           string `json:"clicks"`
					CostMicros       string `json:"cost_micros"`
					Conversions      string `json:"conversions"`
					ConversionsValue string `json:"conversions_value"`
				} `json:"metrics"`
			}
			if err := decode(record.Payload, &payload); err != nil {
				return nil, err
			}
			result.SearchTerms = append(result.SearchTerms, ads.StandardSearchTerm{
				Platform:            record.Platform,
				PlatformAccountID:   record.PlatformAccountID,
				PlatformCampaignID:  payload.CampaignID,
				PlatformAdGroupID:   payload.AdGroupID,
				SearchTerm:          payload.SearchTerm,
				SearchTermMatchType: payload.Segments.SearchTermMatchType,
				StatDate:            parseDate(payload.Segments.Date),
				Impressions:         parseInt64(payload.Metrics.Impressions),
				Clicks:              parseInt64(payload.Metrics.Clicks),
				Spend:               microsToDecimalString(payload.Metrics.CostMicros),
				Conversions:         payload.Metrics.Conversions,
				ConversionsValue:    payload.Metrics.ConversionsValue,
				RawPayload:          record.Payload,
			})
		}
	}
	return result, nil
}
