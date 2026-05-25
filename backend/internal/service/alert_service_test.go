package service

import (
	"context"
	"testing"
)

func TestAlertService_GetSeverityCounts(t *testing.T) {
	svc := NewAlertService(&MockAlertRepo{
		GetAlertsBySeverityFunc: func(ctx context.Context) (map[string]int64, error) {
			return map[string]int64{"critical": 5}, nil
		},
	})
	
	counts, err := svc.GetSeverityCounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if counts["critical"] != 5 {
		t.Errorf("expected 5, got %d", counts["critical"])
	}
}
