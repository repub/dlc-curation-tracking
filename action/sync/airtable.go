package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehanizm/airtable"
)

const (
	// Airtable columns set during sync
	COL_ID             = "ID"
	COL_TITLE          = "Submission Title"
	COL_LINK           = "Submission Link"
	COL_DEPOSITOR_ID   = "Depositor"
	COL_DEPOSITOR_NAME = "Depositor Name"
	COL_DEPOSIT_DATE   = "Deposit Date"
)

type airClient struct {
	air    *airtable.Client
	baseId string // airtable baseID
	table  string // airtable table name for
}

type curateTask struct {
	ID        string // scholarsphereID
	airRecord *airtable.Record
}

var errTaskNotFound = errors.New("Task not found")

func newAirClient(key string, baseID string, table string) *airClient {
	return &airClient{
		air:    airtable.NewClient(key),
		baseId: baseID,
		table:  table,
	}
}

func (mgr *airClient) getTaskFor(ctx context.Context, scholarID string) (*curateTask, error) {
	tbl := mgr.air.GetTable(mgr.baseId, mgr.table)
	filter := fmt.Sprintf("{ID}='%s'", scholarID)
	recs, err := tbl.GetRecords().WithFilterFormula(filter).MaxRecords(1).Do()
	if err != nil {
		return nil, err
	}
	if len(recs.Records) == 0 {
		return nil, errTaskNotFound
	}
	return &curateTask{
		ID:        scholarID,
		airRecord: recs.Records[0],
	}, nil
}

func (mgr *airClient) addRecor(ctx context.Context, vals map[string]any) error {
	tbl := mgr.air.GetTable(mgr.baseId, mgr.table)
	_, err := tbl.AddRecords(&airtable.Records{
		Records: []*airtable.Record{
			{
				Fields: vals,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("task not created: %w", err)
	}
	return nil
}

// create or update values for task for scholarID
func (mgr *airClient) upsert(ctx context.Context, scholarID string, vals map[string]any) error {
	task, err := mgr.getTaskFor(ctx, scholarID)
	if err != nil && !errors.Is(err, errTaskNotFound) {
		return err
	}
	if task != nil {
		_, err := task.airRecord.UpdateRecordPartial(vals)
		if err != nil {
			return fmt.Errorf("updating existing record: %w", err)
		}
		return nil
	}
	vals[COL_ID] = scholarID
	return mgr.addRecor(ctx, vals)
}
