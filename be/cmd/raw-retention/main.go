package main

import (
	"context"
	"flag"
	"log"
	"time"

	"be_ads_project/internal/config"
	rawmysql "be_ads_project/internal/modules/collection/infrastructure/rawstore/mysql"
	mysqlplatform "be_ads_project/internal/platform/mysql"
)

func main() {
	var (
		rawDays    = flag.Int("raw-days", 7, "delete raw_records older than N days")
		outboxDays = flag.Int("outbox-days", 7, "delete published outbox_events older than N days")
		batchSize  = flag.Int("batch", 1000, "delete batch size per round")
	)
	flag.Parse()

	cfg := config.Load()
	ctx := context.Background()

	rawDB, err := mysqlplatform.Open(cfg.RawMySQL)
	if err != nil {
		log.Fatalf("open raw mysql: %v", err)
	}
	defer rawDB.Close()

	repo := rawmysql.NewRepository(rawDB)
	if err := repo.Migrate(ctx); err != nil {
		log.Fatalf("migrate raw mysql: %v", err)
	}

	rawBefore := time.Now().UTC().AddDate(0, 0, -*rawDays)
	outboxBefore := time.Now().UTC().AddDate(0, 0, -*outboxDays)

	var rawTotal int64
	for {
		affected, err := repo.DeleteRawRecordsBefore(ctx, rawBefore, *batchSize)
		if err != nil {
			log.Fatalf("delete raw_records: %v", err)
		}
		rawTotal += affected
		if affected < int64(*batchSize) {
			break
		}
	}

	var outboxTotal int64
	for {
		affected, err := repo.DeletePublishedOutboxBefore(ctx, outboxBefore, *batchSize)
		if err != nil {
			log.Fatalf("delete outbox_events: %v", err)
		}
		outboxTotal += affected
		if affected < int64(*batchSize) {
			break
		}
	}

	log.Printf("retention complete raw_deleted=%d outbox_deleted=%d raw_before=%s outbox_before=%s", rawTotal, outboxTotal, rawBefore.Format(time.RFC3339), outboxBefore.Format(time.RFC3339))
}
