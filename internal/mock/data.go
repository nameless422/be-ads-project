package mock

import (
	"fmt"
	"os"
	"strings"
	"time"

	"be_ads_project/internal/shared/ads"
)

func Accounts() []domain.PlatformAccount {
	now := time.Now()
	return []domain.PlatformAccount{
		{
			ID:          "acct_fb_001",
			Platform:    domain.PlatformFacebook,
			AccountID:   "fb-act-1001",
			AccountName: "Facebook North America",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "USD",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "acct_fb_002",
			Platform:    domain.PlatformFacebook,
			AccountID:   "fb-act-1002",
			AccountName: "Facebook Europe",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "EUR",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "acct_gg_001",
			Platform:    domain.PlatformGoogleAds,
			AccountID:   "248-390-1805",
			AccountName: "Google Ads Test Account 01",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "USD",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "acct_gg_002",
			Platform:    domain.PlatformGoogleAds,
			AccountID:   "492-825-2952",
			AccountName: "Google Ads Test Account 02",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "USD",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "acct_gg_003",
			Platform:    domain.PlatformGoogleAds,
			AccountID:   "691-332-4649",
			AccountName: "Google Ads Test Account 03",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "USD",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "acct_gg_004",
			Platform:    domain.PlatformGoogleAds,
			AccountID:   "400-404-3492",
			AccountName: "Google Ads Test Account 04",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "USD",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "acct_gg_005",
			Platform:    domain.PlatformGoogleAds,
			AccountID:   "608-174-8445",
			AccountName: "Google Ads Test Account 05",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "USD",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "acct_tt_001",
			Platform:    domain.PlatformTikTokAds,
			AccountID:   "tt-act-3001",
			AccountName: "TikTok Growth Lab",
			Status:      domain.AccountStatusActive,
			Timezone:    "Asia/Shanghai",
			Currency:    "USD",
			BusinessID:  "biz-demo",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

func Profiles(accounts []domain.PlatformAccount) []domain.SyncProfile {
	now := time.Now()
	var profiles []domain.SyncProfile
	for _, account := range accounts {
		for _, objectType := range []domain.ObjectType{
			domain.ObjectTypeCampaign,
			domain.ObjectTypeAdGroup,
			domain.ObjectTypeAd,
			domain.ObjectTypeInsight,
		} {
			profiles = append(profiles, domain.SyncProfile{
				ID:                    fmt.Sprintf("profile_%s_%s", account.ID, objectType),
				PlatformAccountID:     account.ID,
				Platform:              account.Platform,
				ObjectType:            objectType,
				SyncMode:              domain.SyncModeIncremental,
				ScheduleType:          domain.ScheduleTypeCron,
				ScheduleExpr:          "*/10 * * * * *",
				LookbackWindowMinutes: 180,
				WatermarkField:        "updated_at",
				WatermarkValue:        now.Add(-2 * time.Hour).Format(time.RFC3339),
				IsEnabled:             true,
				UpdatedAt:             now,
			})
		}
	}
	return profiles
}

func Credentials(accounts []domain.PlatformAccount) map[string]domain.AccountCredential {
	now := time.Now()
	items := make(map[string]domain.AccountCredential, len(accounts))
	googleMode := googleAdsMode()
	googleLoginCustomerID := envOrDefault("BE_GOOGLE_ADS_LOGIN_CUSTOMER_ID", "357-594-0005")
	googleDeveloperToken := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_DEVELOPER_TOKEN"))
	googleClientID := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_CLIENT_ID"))
	googleClientSecret := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_CLIENT_SECRET"))
	googleRefreshToken := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_REFRESH_TOKEN"))

	for _, account := range accounts {
		extraConfig := map[string]string{
			"mode": "mock",
		}
		authType := "mock_token"
		accessToken := fmt.Sprintf("mock-access-token-%s", account.ID)
		refreshToken := fmt.Sprintf("mock-refresh-token-%s", account.ID)
		clientID := fmt.Sprintf("client-%s", account.Platform)
		clientSecret := "mock-secret"

		if account.Platform == domain.PlatformGoogleAds {
			authType = "google_ads"
			accessToken = ""
			refreshToken = googleRefreshToken
			clientID = googleClientID
			clientSecret = googleClientSecret
			extraConfig = map[string]string{
				"mode":              googleMode,
				"developer_token":   googleDeveloperToken,
				"login_customer_id": googleLoginCustomerID,
				"customer_id":       account.AccountID,
				"api_version":       envOrDefault("BE_GOOGLE_ADS_API_VERSION", "v20"),
			}
		}

		items[account.ID] = domain.AccountCredential{
			ID:                fmt.Sprintf("cred_%s", account.ID),
			PlatformAccountID: account.ID,
			AuthType:          authType,
			AccessToken:       accessToken,
			RefreshToken:      refreshToken,
			ClientID:          clientID,
			ClientSecret:      clientSecret,
			ExtraConfig:       extraConfig,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}
	return items
}

func googleAdsMode() string {
	mode := strings.TrimSpace(os.Getenv("BE_GOOGLE_ADS_MODE"))
	if mode != "" {
		return mode
	}
	if hasGoogleAdsOAuth() {
		return "real"
	}
	return "seeded_test"
}

func hasGoogleAdsOAuth() bool {
	required := []string{
		"BE_GOOGLE_ADS_DEVELOPER_TOKEN",
		"BE_GOOGLE_ADS_CLIENT_ID",
		"BE_GOOGLE_ADS_REFRESH_TOKEN",
	}
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return false
		}
	}
	return true
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
