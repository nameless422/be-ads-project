package domain

import "fmt"

type ObjectType string

const (
	ObjectTypeAccount  ObjectType = "account"
	ObjectTypeCampaign ObjectType = "campaign"
	ObjectTypeAdGroup  ObjectType = "ad_group"
	ObjectTypeAd       ObjectType = "ad"
	ObjectTypeInsight  ObjectType = "insight"
)

func (t ObjectType) String() string {
	return string(t)
}

func ParseObjectType(raw string) (ObjectType, error) {
	switch ObjectType(raw) {
	case ObjectTypeAccount, ObjectTypeCampaign, ObjectTypeAdGroup, ObjectTypeAd, ObjectTypeInsight:
		return ObjectType(raw), nil
	default:
		return "", fmt.Errorf("unsupported object type %q", raw)
	}
}
