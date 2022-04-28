package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehanizm/airtable"
)

const (
	COL_ID             = "ID"
	COL_TITLE          = "Submission Title"
	COL_LINK           = "Submission Link"
	COL_DEPOSITOR_ID   = "Depositor"
	COL_DEPOSITOR_NAME = "Depositor Name"
	COL_DEPOSIT_DATE   = "Deposit Date"
	COL_STATUS         = "Status"
	COL_LABELS         = "Labels"
	COL_CURATOR        = "Curation By"
	COL_COMMENTS       = "Comments"
)

type curateMgr struct {
	air    *airtable.Client
	baseId string // airtable baseID
	table  string // airtable table name for
}

type curateTask struct {
	ID        string // scholarsphereID
	airRecord *airtable.Record
}

var ErrTaskNotFound = errors.New("Task not found")
var ErrTaskExists = errors.New("Task already exists")

func NewAirtableCurateMgr(key string, baseID string, table string) *curateMgr {
	return &curateMgr{
		air:    airtable.NewClient(key),
		baseId: baseID,
		table:  table,
	}
}

func (mgr *curateMgr) GetTaskFor(ctx context.Context, scholarID string) (*curateTask, error) {
	tbl := mgr.air.GetTable(mgr.baseId, mgr.table)
	filter := fmt.Sprintf("{ID}='%s'", scholarID)
	recs, err := tbl.GetRecords().WithFilterFormula(filter).MaxRecords(1).Do()
	if err != nil {
		return nil, err
	}
	if len(recs.Records) == 0 {
		return nil, ErrTaskNotFound
	}
	return &curateTask{
		ID:        scholarID,
		airRecord: recs.Records[0],
	}, nil
}

func (mgr *curateMgr) addRecor(ctx context.Context, vals map[string]any) error {
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

func (mgr *curateMgr) Upsert(ctx context.Context, scholarID string, vals map[string]any) error {
	task, err := mgr.GetTaskFor(ctx, scholarID)
	if err != nil && !errors.Is(err, ErrTaskNotFound) {
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
