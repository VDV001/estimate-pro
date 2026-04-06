package domain

import (
	"math"
	"testing"
	"time"
)

func TestPERTEstimate(t *testing.T) {
	tests := []struct {
		name   string
		min    float64
		likely float64
		max    float64
		want   float64
	}{
		{"standard", 2, 4, 6, 4.0},
		{"zeros", 0, 0, 0, 0.0},
		{"same values", 10, 10, 10, 10.0},
		{"skewed pessimistic", 1, 2, 15, 4.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PERTEstimate(tt.min, tt.likely, tt.max)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("PERTEstimate(%v, %v, %v) = %v, want %v", tt.min, tt.likely, tt.max, got, tt.want)
			}
		})
	}
}

func TestAggregate_SingleEstimation(t *testing.T) {
	estimations := []*EstimationWithItems{
		{
			Estimation: &Estimation{SubmittedBy: "user-1", Status: StatusSubmitted},
			Items: []*EstimationItem{
				{TaskName: "Backend API", MinHours: 4, LikelyHours: 8, MaxHours: 16, SortOrder: 0},
				{TaskName: "Frontend UI", MinHours: 2, LikelyHours: 4, MaxHours: 8, SortOrder: 1},
			},
		},
	}

	result := Aggregate(estimations)

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	// Backend API: (4+32+16)/6 = 8.667
	backendPERT := PERTEstimate(4, 8, 16)
	if math.Abs(result.Items[0].AvgPERTHours-backendPERT) > 1e-9 {
		t.Errorf("Backend PERT = %v, want %v", result.Items[0].AvgPERTHours, backendPERT)
	}
	if result.Items[0].EstimatorCount != 1 {
		t.Errorf("Backend estimator count = %d, want 1", result.Items[0].EstimatorCount)
	}

	// Frontend UI: (2+16+8)/6 = 4.333
	frontendPERT := PERTEstimate(2, 4, 8)
	if math.Abs(result.Items[1].AvgPERTHours-frontendPERT) > 1e-9 {
		t.Errorf("Frontend PERT = %v, want %v", result.Items[1].AvgPERTHours, frontendPERT)
	}

	expectedTotal := backendPERT + frontendPERT
	if math.Abs(result.TotalHours-expectedTotal) > 1e-9 {
		t.Errorf("TotalHours = %v, want %v", result.TotalHours, expectedTotal)
	}
}

func TestAggregate_MultipleEstimations(t *testing.T) {
	estimations := []*EstimationWithItems{
		{
			Estimation: &Estimation{ID: "est-1", SubmittedBy: "user-1", Status: StatusSubmitted},
			Items: []*EstimationItem{
				{TaskName: "Database", MinHours: 2, LikelyHours: 4, MaxHours: 8, SortOrder: 0},
			},
		},
		{
			Estimation: &Estimation{ID: "est-2", SubmittedBy: "user-2", Status: StatusSubmitted},
			Items: []*EstimationItem{
				{TaskName: "Database", MinHours: 4, LikelyHours: 6, MaxHours: 12, SortOrder: 0},
			},
		},
		{
			Estimation: &Estimation{ID: "est-3", SubmittedBy: "user-3", Status: StatusSubmitted},
			Items: []*EstimationItem{
				{TaskName: "Database", MinHours: 3, LikelyHours: 5, MaxHours: 10, SortOrder: 0},
			},
		},
	}

	result := Aggregate(estimations)

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.EstimatorCount != 3 {
		t.Errorf("estimator count = %d, want 3", item.EstimatorCount)
	}

	pert1 := PERTEstimate(2, 4, 8)
	pert2 := PERTEstimate(4, 6, 12)
	pert3 := PERTEstimate(3, 5, 10)
	expectedAvg := (pert1 + pert2 + pert3) / 3

	if math.Abs(item.AvgPERTHours-expectedAvg) > 1e-9 {
		t.Errorf("AvgPERTHours = %v, want %v", item.AvgPERTHours, expectedAvg)
	}
	if item.MinOfMins != 2 {
		t.Errorf("MinOfMins = %v, want 2", item.MinOfMins)
	}
	if item.MaxOfMaxes != 12 {
		t.Errorf("MaxOfMaxes = %v, want 12", item.MaxOfMaxes)
	}
}

func TestAggregate_SkipsDrafts(t *testing.T) {
	estimations := []*EstimationWithItems{
		{
			Estimation: &Estimation{SubmittedBy: "user-1", Status: StatusDraft},
			Items: []*EstimationItem{
				{TaskName: "Task A", MinHours: 100, LikelyHours: 200, MaxHours: 300},
			},
		},
		{
			Estimation: &Estimation{SubmittedBy: "user-1", Status: StatusSubmitted},
			Items: []*EstimationItem{
				{TaskName: "Task A", MinHours: 2, LikelyHours: 4, MaxHours: 6},
			},
		},
	}

	result := Aggregate(estimations)

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].EstimatorCount != 1 {
		t.Errorf("should skip draft, estimator count = %d, want 1", result.Items[0].EstimatorCount)
	}
	expectedPERT := PERTEstimate(2, 4, 6)
	if math.Abs(result.Items[0].AvgPERTHours-expectedPERT) > 1e-9 {
		t.Errorf("AvgPERTHours = %v, want %v", result.Items[0].AvgPERTHours, expectedPERT)
	}
}

func TestAggregate_Empty(t *testing.T) {
	result := Aggregate(nil)
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for nil input, got %d", len(result.Items))
	}
	if result.TotalHours != 0 {
		t.Errorf("expected 0 total hours, got %v", result.TotalHours)
	}
}

func TestAggregate_MultipleEstimationsSameUser(t *testing.T) {
	now := time.Now()
	estimations := []*EstimationWithItems{
		{
			Estimation: &Estimation{ID: "est-1", SubmittedBy: "user-1", Status: StatusSubmitted, CreatedAt: now.Add(-2 * time.Hour)},
			Items: []*EstimationItem{
				{TaskName: "Task A", MinHours: 1, LikelyHours: 2, MaxHours: 3, SortOrder: 0},
				{TaskName: "Task B", MinHours: 2, LikelyHours: 4, MaxHours: 6, SortOrder: 1},
			},
		},
		{
			Estimation: &Estimation{ID: "est-2", SubmittedBy: "user-1", Status: StatusSubmitted, CreatedAt: now.Add(-1 * time.Hour)},
			Items: []*EstimationItem{
				{TaskName: "Task C", MinHours: 3, LikelyHours: 5, MaxHours: 8, SortOrder: 0},
				{TaskName: "Task D", MinHours: 1, LikelyHours: 3, MaxHours: 5, SortOrder: 1},
			},
		},
		{
			Estimation: &Estimation{ID: "est-3", SubmittedBy: "user-1", Status: StatusSubmitted, CreatedAt: now},
			Items: []*EstimationItem{
				{TaskName: "Task E", MinHours: 2, LikelyHours: 4, MaxHours: 7, SortOrder: 0},
			},
		},
	}

	result := Aggregate(estimations)

	// All 5 tasks from 3 estimations by same user should appear
	if len(result.Items) != 5 {
		t.Fatalf("expected 5 items (all tasks from all estimations), got %d", len(result.Items))
	}
	if result.Items[0].EstimatorCount != 1 {
		t.Errorf("estimator count = %d, want 1 (single user)", result.Items[0].EstimatorCount)
	}
}

func TestAggregate_DedupSameTaskAcrossEstimations(t *testing.T) {
	now := time.Now()
	estimations := []*EstimationWithItems{
		{
			Estimation: &Estimation{ID: "old", SubmittedBy: "user-1", Status: StatusSubmitted, CreatedAt: now.Add(-1 * time.Hour)},
			Items: []*EstimationItem{
				{TaskName: "Task A", MinHours: 10, LikelyHours: 20, MaxHours: 30, SortOrder: 0},
			},
		},
		{
			Estimation: &Estimation{ID: "new", SubmittedBy: "user-1", Status: StatusSubmitted, CreatedAt: now},
			Items: []*EstimationItem{
				{TaskName: "Task A", MinHours: 1, LikelyHours: 2, MaxHours: 3, SortOrder: 0},
			},
		},
	}

	result := Aggregate(estimations)

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item (deduped), got %d", len(result.Items))
	}

	// Should use the newer estimation's values
	expectedPERT := PERTEstimate(1, 2, 3)
	if math.Abs(result.Items[0].AvgPERTHours-expectedPERT) > 1e-9 {
		t.Errorf("AvgPERTHours = %v, want %v (from newer estimation)", result.Items[0].AvgPERTHours, expectedPERT)
	}
}

func TestAggregate_PreservesTaskOrder(t *testing.T) {
	estimations := []*EstimationWithItems{
		{
			Estimation: &Estimation{SubmittedBy: "user-1", Status: StatusSubmitted},
			Items: []*EstimationItem{
				{TaskName: "Alpha", MinHours: 1, LikelyHours: 2, MaxHours: 3, SortOrder: 0},
				{TaskName: "Beta", MinHours: 1, LikelyHours: 2, MaxHours: 3, SortOrder: 1},
				{TaskName: "Gamma", MinHours: 1, LikelyHours: 2, MaxHours: 3, SortOrder: 2},
			},
		},
	}

	result := Aggregate(estimations)

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	if result.Items[0].TaskName != "Alpha" {
		t.Errorf("first item = %q, want Alpha", result.Items[0].TaskName)
	}
	if result.Items[1].TaskName != "Beta" {
		t.Errorf("second item = %q, want Beta", result.Items[1].TaskName)
	}
	if result.Items[2].TaskName != "Gamma" {
		t.Errorf("third item = %q, want Gamma", result.Items[2].TaskName)
	}
}
