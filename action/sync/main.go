package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	airTableName = "Submissions"
)

func main() {
	flag.Parse()
	ctx := context.Background()

	airkey := os.Getenv("AIRTABLE_APIKEY")
	airbase := os.Getenv("AIRTABLE_BASEID")
	airtable := airTableName
	mgr := NewAirtableCurateMgr(airkey, airbase, airtable)

	if flag.Arg(0) == "gh" {
		// sync github
		ghKey := os.Getenv("GITHUB_APIKEY")
		ghc := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghKey},
		)))
		if err := ghsync(ctx, ghc, mgr); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	if err := syncFromCatalog(ctx, mgr); err != nil {
		log.Fatal(err)
	}

}

func syncFromCatalog(ctx context.Context, mgr *curateMgr) error {
	// get submissions from the last 30 days
	items, err := GetDepositsAfter(ctx, daysAgo(30))
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
		w, err := GetWorkMeta(ctx, i.ID)
		if err != nil {
			log.Println(err)
		}
		if w != nil {
			values[COL_DEPOSITOR_ID] = w.Depositor.PSUID
			values[COL_DEPOSITOR_NAME] = w.Depositor.DisplayName
			values[COL_DEPOSIT_DATE] = w.DepositedAt
		}
		err = mgr.Upsert(ctx, i.ID, values)
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
