package domain

type RawRecord struct {
	Platform          Platform
	PlatformAccountID string
	ObjectType        ObjectType
	ResourceID        string
	Payload           []byte
}
