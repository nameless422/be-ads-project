package domain

import "fmt"

type Platform string

const (
	PlatformFacebook  Platform = "facebook"
	PlatformGoogleAds Platform = "google_ads"
	PlatformTikTokAds Platform = "tiktok_ads"
)

func (p Platform) String() string {
	return string(p)
}

func ParsePlatform(raw string) (Platform, error) {
	switch Platform(raw) {
	case PlatformFacebook, PlatformGoogleAds, PlatformTikTokAds:
		return Platform(raw), nil
	default:
		return "", fmt.Errorf("unsupported platform %q", raw)
	}
}
