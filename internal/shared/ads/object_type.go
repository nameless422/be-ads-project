package domain

import "fmt"

type ObjectType string

const (
	ObjectTypeAccount    ObjectType = "account"
	ObjectTypeCampaign   ObjectType = "campaign"
	ObjectTypeAdGroup    ObjectType = "ad_group"
	ObjectTypeAd         ObjectType = "ad"
	ObjectTypeInsight    ObjectType = "insight"
	ObjectTypeSearchTerm ObjectType = "search_term"
)

func (t ObjectType) String() string {
	return string(t)
}

func ParseObjectType(raw string) (ObjectType, error) {
	switch ObjectType(raw) {
	case ObjectTypeAccount, ObjectTypeCampaign, ObjectTypeAdGroup, ObjectTypeAd, ObjectTypeInsight, ObjectTypeSearchTerm:
		return ObjectType(raw), nil
	default:
		return "", fmt.Errorf("unsupported object type %q", raw)
	}
}
