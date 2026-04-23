package domain

import "time"

type AccountStatus string

const (
	AccountStatusActive      AccountStatus = "active"
	AccountStatusDisabled    AccountStatus = "disabled"
	AccountStatusAuthExpired AccountStatus = "auth_expired"
)

type PlatformAccount struct {
	ID          string
	Platform    Platform
	AccountID   string
	AccountName string
	Status      AccountStatus
	Timezone    string
	Currency    string
	BusinessID  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AccountCredential struct {
	ID                string
	PlatformAccountID string
	AuthType          string
	AccessToken       string
	RefreshToken      string
	TokenExpireAt     *time.Time
	ClientID          string
	ClientSecret      string
	ExtraConfig       map[string]string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
