package domain

import "time"

type SyncMode string

const (
	SyncModeBootstrap   SyncMode = "full_bootstrap"
	SyncModeIncremental SyncMode = "incremental"
)

type ScheduleType string

const (
	ScheduleTypeCron ScheduleType = "cron"
)

type SyncProfile struct {
	ID                    string
	PlatformAccountID     string
	Platform              Platform
	ObjectType            ObjectType
	SyncMode              SyncMode
	ScheduleType          ScheduleType
	ScheduleExpr          string
	LookbackWindowMinutes int
	WatermarkField        string
	WatermarkValue        string
	PageToken             string
	IsEnabled             bool
	LastSuccessAt         *time.Time
	LastErrorAt           *time.Time
	UpdatedAt             time.Time
}

func (p SyncProfile) LookbackWindow() time.Duration {
	if p.LookbackWindowMinutes <= 0 {
		return 0
	}
	return time.Duration(p.LookbackWindowMinutes) * time.Minute
}
