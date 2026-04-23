package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	ingestiondomain "be_ads_project/internal/modules/collection/domain"
	"be_ads_project/internal/modules/collection/infrastructure/connectors/meta"
	rootdomain "be_ads_project/internal/shared/ads"
)

type MetaCollector struct {
	registry *meta.Registry
	logger   *log.Logger
}

func NewMetaCollector(registry *meta.Registry, logger *log.Logger) *MetaCollector {
	return &MetaCollector{
		registry: registry,
		logger:   logger,
	}
}

func (c *MetaCollector) Collect(ctx context.Context, target ingestiondomain.SyncTarget) (*ingestiondomain.CollectedBatch, error) {
	connector, err := c.registry.Get(target.Profile.Platform)
	if err != nil {
		return nil, err
	}

	accountCtx := meta.AccountContext{
		Account:    target.Bundle.Account,
		Credential: target.Bundle.Credential,
	}
	if err := connector.Validate(ctx, accountCtx); err != nil {
		return nil, fmt.Errorf("validate connector config: %w", err)
	}

	sourceMode := target.Bundle.Credential.ExtraConfig["mode"]
	c.logger.Printf(
		"[ingestion] collect start profile=%s platform=%s account=%s object=%s mode=%s source_mode=%s watermark=%s cursor=%s",
		target.Profile.ID,
		target.Profile.Platform,
		target.Bundle.Account.AccountID,
		target.Profile.ObjectType,
		target.Profile.SyncMode,
		sourceMode,
		target.Profile.WatermarkValue,
		target.Profile.PageToken,
	)

	req := meta.FetchRequest{
		AccountContext: accountCtx,
		ObjectType:     target.Profile.ObjectType,
		PageSize:       200,
		EndTime:        timePtr(time.Now().UTC()),
		Checkpoint: meta.Checkpoint{
			PageCursor:     target.Profile.PageToken,
			LookbackWindow: target.Profile.LookbackWindow(),
		},
	}
	if target.Profile.WatermarkValue != "" {
		if watermark, err := time.Parse(time.RFC3339, target.Profile.WatermarkValue); err == nil {
			req.Checkpoint.TimeWatermark = &watermark
		}
	}

	result, err := c.fetchByObjectType(ctx, connector, req)
	if err != nil {
		return nil, err
	}

	return &ingestiondomain.CollectedBatch{
		Target:         target,
		Records:        result.RawRecords,
		HasMore:        result.HasMore,
		NextCursor:     result.NextPageCursor,
		NextWatermark:  result.NextTimeWatermark,
		SourceMode:     sourceMode,
		CollectedAtUTC: time.Now().UTC(),
	}, nil
}

func (c *MetaCollector) fetchByObjectType(ctx context.Context, connector meta.Connector, req meta.FetchRequest) (*meta.FetchResult, error) {
	switch req.ObjectType {
	case rootdomain.ObjectTypeAccount:
		return connector.FetchAccounts(ctx, req)
	case rootdomain.ObjectTypeCampaign:
		return connector.FetchCampaigns(ctx, req)
	case rootdomain.ObjectTypeAdGroup:
		return connector.FetchAdGroups(ctx, req)
	case rootdomain.ObjectTypeAd:
		return connector.FetchAds(ctx, req)
	case rootdomain.ObjectTypeInsight:
		return connector.FetchInsights(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported object type %s", req.ObjectType)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
