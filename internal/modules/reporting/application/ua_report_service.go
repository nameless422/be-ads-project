package application

import (
	"context"
	"fmt"
	"strconv"
	"time"

	biquerydomain "be_ads_project/internal/modules/reporting/domain"
	rootdomain "be_ads_project/internal/shared/ads"
)

type UAAdReader interface {
	QueryUAReportRows(context.Context, biquerydomain.UAReportFilter) ([]biquerydomain.UAAdReportRow, error)
}

type GameKPIReader interface {
	ListGameKPIs(context.Context, biquerydomain.GameKPIQueryFilter) ([]biquerydomain.GameKPIRecord, error)
	UpsertGameKPIs(context.Context, []biquerydomain.GameKPIRecord) error
}

type UAReportService struct {
	adReader   UAAdReader
	gameReader GameKPIReader
}

func NewUAReportService(adReader UAAdReader, gameReader GameKPIReader) *UAReportService {
	return &UAReportService{adReader: adReader, gameReader: gameReader}
}

func (s *UAReportService) QueryReport(ctx context.Context, filter biquerydomain.UAReportFilter) ([]biquerydomain.UAReportRow, error) {
	adRows, err := s.adReader.QueryUAReportRows(ctx, filter)
	if err != nil {
		return nil, err
	}
	gameRows, err := s.gameReader.ListGameKPIs(ctx, biquerydomain.GameKPIQueryFilter{
		Platform:           filter.Platform,
		AccountID:          filter.AccountID,
		DateFrom:           filter.DateFrom,
		DateTo:             filter.DateTo,
		Country:            filter.Country,
		OS:                 filter.OS,
		PlatformCampaignID: filter.PlatformCampaignID,
		PlatformAdGroupID:  filter.PlatformAdGroupID,
		PlatformAdID:       filter.PlatformAdID,
		Limit:              filter.Limit,
	})
	if err != nil {
		return nil, err
	}

	rows := make(map[string]*biquerydomain.UAReportRow, len(adRows)+len(gameRows))
	order := make([]string, 0, len(adRows)+len(gameRows))

	for _, item := range adRows {
		key := buildUAKey(
			item.StatDate,
			item.Platform,
			item.PlatformAccountID,
			item.PlatformCampaignID,
			item.PlatformAdGroupID,
			item.PlatformAdID,
			item.EntityLevel,
			item.EntityID,
			"",
			"",
		)
		if _, ok := rows[key]; !ok {
			rows[key] = &biquerydomain.UAReportRow{
				Platform:           item.Platform,
				PlatformAccountID:  item.PlatformAccountID,
				PlatformCampaignID: item.PlatformCampaignID,
				PlatformAdGroupID:  item.PlatformAdGroupID,
				PlatformAdID:       item.PlatformAdID,
				EntityLevel:        item.EntityLevel,
				EntityID:           item.EntityID,
				StatDate:           item.StatDate,
			}
			order = append(order, key)
		}
		row := rows[key]
		row.Device = item.Device
		row.Network = item.Network
		row.Impressions = item.Impressions
		row.Clicks = item.Clicks
		row.CTR = item.CTR
		row.CPM = item.CPM
		row.CPC = item.CPC
		row.Spend = item.Spend
		row.Reach = item.Reach
		row.Frequency = item.Frequency
		row.Conversions = item.Conversions
		row.AllConversions = item.AllConversions
		row.ConversionsValue = item.ConversionsValue
		row.CostPerConversion = item.CostPerConversion
		row.CostPerAllConversions = item.CostPerAllConversions
		row.ROAS = item.ROAS
	}

	for _, item := range gameRows {
		entityLevel, entityID := entityFromGameKPI(item)
		key := buildUAKey(
			item.StatDate,
			item.Platform,
			item.PlatformAccountID,
			item.PlatformCampaignID,
			item.PlatformAdGroupID,
			item.PlatformAdID,
			entityLevel,
			entityID,
			item.Country,
			item.OS,
		)
		if _, ok := rows[key]; !ok {
			rows[key] = &biquerydomain.UAReportRow{
				Platform:           item.Platform,
				PlatformAccountID:  item.PlatformAccountID,
				PlatformCampaignID: item.PlatformCampaignID,
				PlatformAdGroupID:  item.PlatformAdGroupID,
				PlatformAdID:       item.PlatformAdID,
				EntityLevel:        entityLevel,
				EntityID:           entityID,
				StatDate:           item.StatDate,
			}
			order = append(order, key)
		}
		row := rows[key]
		row.Country = item.Country
		row.OS = item.OS
		row.Placement = item.Placement
		row.CreativeID = item.CreativeID
		row.CreativeType = item.CreativeType
		row.OptimizationGoal = item.OptimizationGoal
		row.BidType = item.BidType
		row.Targeting = item.Targeting
		row.Installs = item.Installs
		row.Activations = item.Activations
		row.Registrations = item.Registrations
		row.TutorialCompletions = item.TutorialCompletions
		row.RoleCreations = item.RoleCreations
		row.LevelXUsers = item.LevelXUsers
		row.Purchasers = item.Purchasers
		row.PurchaseCount = item.PurchaseCount
		row.FirstPurchaseAmount = item.FirstPurchaseAmount
		row.RevenueD1 = item.RevenueD1
		row.RevenueD7 = item.RevenueD7
		row.RevenueD30 = item.RevenueD30
		row.AdRevenue = item.AdRevenue
		row.TotalRevenue = item.TotalRevenue
		row.RetentionD1 = item.RetentionD1
		row.RetentionD3 = item.RetentionD3
		row.RetentionD7 = item.RetentionD7
		row.RetentionD30 = item.RetentionD30
		row.LTVD7 = item.LTVD7
		row.LTVD30 = item.LTVD30
		row.AvgOnlineDurationSeconds = item.AvgOnlineDurationSeconds
		row.TaskCompletionRate = item.TaskCompletionRate
		row.HighValuePayerRatio = item.HighValuePayerRatio
	}

	result := make([]biquerydomain.UAReportRow, 0, len(order))
	for _, key := range order {
		row := rows[key]
		fillUADerivedMetrics(row)
		result = append(result, *row)
	}
	return result, nil
}

func buildUAKey(statDate time.Time, platform rootdomain.Platform, accountID, campaignID, adGroupID, adID string, entityLevel rootdomain.ObjectType, entityID, country, os string) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		statDate.Format("2006-01-02"),
		platform,
		accountID,
		campaignID,
		adGroupID,
		adID,
		entityLevel,
		entityID,
		country,
		os,
	)
}

func entityFromGameKPI(item biquerydomain.GameKPIRecord) (rootdomain.ObjectType, string) {
	switch {
	case item.PlatformAdID != "":
		return rootdomain.ObjectTypeAd, item.PlatformAdID
	case item.PlatformAdGroupID != "":
		return rootdomain.ObjectTypeAdGroup, item.PlatformAdGroupID
	case item.PlatformCampaignID != "":
		return rootdomain.ObjectTypeCampaign, item.PlatformCampaignID
	default:
		return rootdomain.ObjectTypeCampaign, ""
	}
}

func fillUADerivedMetrics(row *biquerydomain.UAReportRow) {
	spend := parseFloat(row.Spend)
	installs := float64(row.Installs)
	activations := float64(row.Activations)
	registrations := float64(row.Registrations)
	purchasers := float64(row.Purchasers)
	totalRevenue := parseFloat(row.TotalRevenue)
	revenueD30 := parseFloat(row.RevenueD30)
	ltvD30 := parseFloat(row.LTVD30)

	row.CPI = ratioString(spend, installs)
	row.ActivationRate = ratioString(activations, installs)
	row.CPR = ratioString(spend, registrations)
	row.RegistrationRate = ratioString(registrations, installs)
	row.PayerRate = ratioString(purchasers, activations)
	row.ARPU = ratioString(totalRevenue, activations)
	row.ARPPU = ratioString(totalRevenue, purchasers)
	row.ROI = ratioString(totalRevenue-spend, spend)
	if totalRevenue == 0 && revenueD30 > 0 {
		row.ROI = ratioString(revenueD30-spend, spend)
	}
	if row.TotalRevenue == "" && revenueD30 > 0 {
		row.TotalRevenue = formatFloat(revenueD30)
	}
	if row.LTVD30 == "" && ltvD30 == 0 && row.Installs > 0 {
		row.LTVD30 = ratioString(totalRevenue, installs)
		ltvD30 = parseFloat(row.LTVD30)
	}
	row.LTVToCPIRatio = ratioString(ltvD30, parseFloat(row.CPI))
}

func parseFloat(raw string) float64 {
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}

func ratioString(numerator, denominator float64) string {
	if denominator == 0 {
		return "0.00"
	}
	return formatFloat(numerator / denominator)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}
