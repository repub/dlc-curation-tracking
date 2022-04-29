package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	//environment variables
	envAIRTABLE_APIKEY = "AIRTABLE_APIKEY"
	envAIRTABLE_BASEID = "AIRTABLE_BASEID"
	envSYNC_DAYSAGO    = "SYNC_DAYSAGO"

	defaultDaysAgo = 60
	airTableName   = "Submissions"
)

func main() {
	flag.Parse()
	ctx := context.Background()
	airkey := os.Getenv(envAIRTABLE_APIKEY)
	airbase := os.Getenv(envAIRTABLE_BASEID)
	airtable := airTableName
	mgr := newAirClient(airkey, airbase, airtable)
	if err := syncFromCatalog(ctx, mgr); err != nil {
		log.Fatal(err)
	}
}

// syncFromCatalog is the primary sync function: fetches entries from
// ScholarSphere catalog, retrieves additional metadata from GraphQL endpoint,
// and creates/updates entries in Airtable
func syncFromCatalog(ctx context.Context, mgr *airClient) error {
	days := defaultDaysAgo
	if envDays := os.Getenv(envSYNC_DAYSAGO); envDays != "" {
		d, err := strconv.Atoi(envDays)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", envSYNC_DAYSAGO, err)
		}
		days = d
	}
	after := daysAgo(days)
	log.Printf("syncing deposits since %s", after.Format("2006-01-02"))
	items, err := getDepositsAfter(ctx, after)
	if err != nil {
		return fmt.Errorf("failed to get deposits from the catalog: %w", err)
	}
	for _, i := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		values := map[string]any{
			COL_TITLE: i.Title,
			COL_LINK:  i.Link,
		}
		w, err := getWorkMeta(ctx, i.ID)
		if err != nil {
			log.Println(err) // not fatal
		}
		if w != nil {
			values[COL_DEPOSITOR_ID] = w.Depositor.PSUID
			values[COL_DEPOSITOR_NAME] = w.Depositor.DisplayName
			values[COL_DEPOSIT_DATE] = w.DepositedAt
		}
		err = mgr.upsert(ctx, i.ID, values)
		if err != nil {
			return fmt.Errorf("failed to sync task for %s: %w", i.ID, err)
		}
		log.Println("✔️ ", i.ID)
	}
	return nil
}

func daysAgo(num int) time.Time {
	return time.Now().Add(-1 * time.Duration(num*24) * time.Hour)
}
