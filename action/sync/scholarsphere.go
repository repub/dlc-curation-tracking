package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	graphql "github.com/hasura/go-graphql-client"
)

const (
	catalogURL = "https://scholarsphere.psu.edu/catalog.json?sort=deposited_at_dtsi+desc&per_page=250"
	graphqlURL = "https://scholarsphere.psu.edu/api/public"
	timeLayout = "January 2, 2006 15:04"
)

// deposit is a work or collection in ScholarSphere
type deposit struct {
	ID          string
	Title       string
	DepositedAt time.Time
	Link        string
}

type workMeta struct {
	ID          string `graphql:"id"`
	Title       string
	Visibility  string
	UpdatedAt   string
	DepositedAt string
	Depositor   struct {
		DisplayName string
		Email       string
		GivenName   string
		PSUID       string `graphql:"psuId"`
	}
}

// catalogResp is scholarsphere catalog.json response
type catalogResp struct {
	Links struct {
		Self string `json:"self"`
		Next string `json:"next"`
		Last string `json:"last"`
	} `json:"links"`
	Meta struct {
		Pages struct {
			CurrentPage int  `json:"current_page"`
			NextPage    int  `json:"next_page"`
			PrevPage    int  `json:"prev_page"`
			TotalPages  int  `json:"total_pages"`
			LimitValue  int  `json:"limit_value"`
			OffsetValue int  `json:"offset_value"`
			TotalCount  int  `json:"total_count"`
			FirstPage   bool `json:"first_page?"`
			LastPage    bool `json:"last_page?"`
		} `json:"pages"`
	} `json:"meta"`
	Data []catalogItem `json:"data"`
}

type catalogItem struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Title     attr `json:"title_tesim"`
		Deposited attr `json:"deposited_at_dtsi"`
	} `json:"attributes"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
}

type attr struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Value interface{} `json:"value"`
		Label string      `json:"label"`
	} `json:"attributes"`
}

func httpClient() *http.Client {
	c := http.DefaultClient
	c.Timeout = 15 * time.Second
	return c
}

func GetDepositsAfter(ctx context.Context, after time.Time) ([]deposit, error) {
	var items []deposit
	var pageCount int
	url := catalogURL
	cli := httpClient()
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := cli.Get(url)
		if err != nil {
			return nil, err
		}
		if code := resp.StatusCode; code != http.StatusOK {
			return nil, fmt.Errorf("catalog returned %d", code)
		}
		var catResp catalogResp
		err = json.NewDecoder(resp.Body).Decode(&catResp)
		if err != nil {
			return nil, err
		}
		if catResp.Meta.Pages.CurrentPage != pageCount+1 {
			return nil, errors.New("catalog not paginating as expected")
		}
		var itemsAfter []deposit
		for _, item := range catResp.Data {
			dep := deposit{
				Title: item.Attributes.Title.Attributes.Value.(string),
				ID:    item.ID,
				Link:  item.Links.Self,
			}
			if dep.ID == "" || dep.Link == "" {
				return nil, errors.New("catalog item missing ID/Link")
			}
			dep.DepositedAt, err = time.Parse(timeLayout, item.Attributes.Deposited.Attributes.Value.(string))
			if err != nil {
				return nil, err
			}
			if dep.DepositedAt.After(after) {
				itemsAfter = append(itemsAfter, dep)
			}
		}
		if len(itemsAfter) == 0 {
			// assumes items sorted by deposit date!
			return items, nil
		}
		items = append(items, itemsAfter...)
		if catResp.Meta.Pages.LastPage {
			return items, nil
		}
		url = catResp.Links.Next
		pageCount += 1
	}
}

var ErrResourceNotFound = errors.New("resource not found")
var ErrNotWork = errors.New("resource not a work")

func GetWorkMeta(ctx context.Context, id string) (*workMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cli := graphql.NewClient(graphqlURL, httpClient())
	var query struct {
		Work workMeta `graphql:"work(id: $id)"`
	}
	err := cli.Query(ctx, &query, map[string]any{"id": id})
	if err != nil {
		var gqlErr graphql.Errors
		if errors.As(err, &gqlErr) && strings.Contains(gqlErr.Error(), "404") {
			return nil, fmt.Errorf("%s: %w", id, ErrResourceNotFound)
		}
		return nil, err
	}
	if query.Work.ID == "" {
		return nil, fmt.Errorf("%s: %w", id, ErrNotWork)
	}
	return &query.Work, nil
}
