package main

import (
	"context"
	"errors"
	"log"
	"regexp"

	"github.com/google/go-github/github"
)

const (
	repoOwner = "repub"
	repoName  = "dlc-curation-tracking"
)

var re = regexp.MustCompile("https://scholarsphere.psu.edu/resources/([a-z0-9-]{36})")

var userMap = map[string]string{
	"srerickson":   "sre53@psu.edu",
	"CindySCE2018": "xzx1@psu.edu",
	"bdezray":      "bde125@psu.edu",
	"PaulinaKrys":  "pmk5516@psu.edu",
}

var labelMap = map[string]string{
	"Attention":                              "Attention",
	"DCN":                                    "DCN",
	"draft":                                  "Draft",
	"embargoed":                              "Embargoed",
	"No Response":                            "No Response",
	"Pending contact":                        "Pending Contact",
	"PII (personal identifying information)": "PII",
	"Specialist":                             "Specialist",
}

type ghTask struct {
	ID     string
	Labels []string
	Status string
}

func ghsync(ctx context.Context, client *github.Client, mgr *curateMgr) error {

	// curation manager config
	// airkey := os.Getenv("AIRTABLE_APIKEY")
	// airbase := os.Getenv("AIRTABLE_BASEID")
	// airtable := airTable

	opt := &github.IssueListByRepoOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var allIssues []*github.Issue

	for {
		// list all repositories for the authenticated user
		issues, resp, err := client.Issues.ListByRepo(ctx, repoOwner, repoName, opt)
		if err != nil {
			return err
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	for _, i := range allIssues {
		match := re.FindStringSubmatch(*i.Body)
		if len(match) != 2 {
			continue
		}
		ssLInk := match[0]
		id := match[1]
		status := *i.State // open, closed
		var labels []string
		for _, l := range i.Labels {
			labels = append(labels, labelMap[*l.Name])
		}
		comments, err := joinComments(ctx, client, *i.Number)
		var curator string
		if len(i.Assignees) == 1 {
			curator = userMap[*i.Assignees[0].Login]
		}
		var title, depositorID, depositorName, depositedAt string
		w, err := GetWorkMeta(ctx, id)
		if err != nil {
			if errors.Is(err, ErrNotWork) {
				labels = append(labels, "Collection")
			}
			if errors.Is(err, ErrResourceNotFound) {
				labels = append(labels, "Deleted")
			}
			log.Printf("err: %s: %s", id, err)
		}
		if err == nil {
			title = w.Title
			depositorID = w.Depositor.PSUID
			depositedAt = w.DepositedAt
			depositorName = w.Depositor.DisplayName
		}
		values := map[string]any{
			COL_TITLE:          title,
			COL_LINK:           ssLInk,
			COL_STATUS:         status,
			COL_LABELS:         labels,
			COL_DEPOSITOR_ID:   depositorID,
			COL_DEPOSITOR_NAME: depositorName,
			COL_COMMENTS:       comments,
		}
		if curator != "" {
			values[COL_CURATOR] = []map[string]any{{"email": curator}}
		}
		if depositedAt != "" {
			values[COL_DEPOSIT_DATE] = depositedAt
		}
		err = mgr.Upsert(ctx, id, values)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func joinComments(ctx context.Context, client *github.Client, num int) (string, error) {
	var comment string
	opts := &github.IssueListCommentsOptions{
		Sort:        "created",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		comments, resp, err := client.Issues.ListComments(ctx, repoOwner, repoName, num, opts)
		if err != nil {
			return "", err
		}
		for _, c := range comments {
			comment = comment + *c.Body + "\n"
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return comment, nil
}
