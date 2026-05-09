package domain

import "time"

type SyncCheckpoint struct {
	ProfileID       string
	NextCursor      string
	NextWatermark   string
	LastCollectedAt time.Time
}
