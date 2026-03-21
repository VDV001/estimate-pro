// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "math"

// AggregatedItem represents a single task aggregated across multiple estimations.
type AggregatedItem struct {
	TaskName       string  `json:"task_name"`
	AvgPERTHours   float64 `json:"avg_pert_hours"`
	MinOfMins      float64 `json:"min_of_mins"`
	MaxOfMaxes     float64 `json:"max_of_maxes"`
	EstimatorCount int     `json:"estimator_count"`
}

// AggregatedResult is the full aggregation result for a project.
type AggregatedResult struct {
	Items      []*AggregatedItem `json:"items"`
	TotalHours float64           `json:"total_hours"`
}

// PERTEstimate calculates (min + 4*likely + max) / 6.
func PERTEstimate(min, likely, max float64) float64 {
	return (min + 4*likely + max) / 6
}

// Aggregate computes averaged PERT estimates across multiple estimations, grouped by task name.
// Only submitted estimations are included.
func Aggregate(estimations []*EstimationWithItems) *AggregatedResult {
	type accumulator struct {
		pertSum    float64
		minOfMins  float64
		maxOfMaxes float64
		count      int
		users      map[string]bool
		firstOrder int
	}

	taskMap := make(map[string]*accumulator)
	var taskOrder []string

	// Use only the latest submitted estimation per user
	latestByUser := make(map[string]*EstimationWithItems)
	for _, est := range estimations {
		if est.Estimation.Status != StatusSubmitted {
			continue
		}
		existing, ok := latestByUser[est.Estimation.SubmittedBy]
		if !ok || est.Estimation.CreatedAt.After(existing.Estimation.CreatedAt) {
			latestByUser[est.Estimation.SubmittedBy] = est
		}
	}

	for _, est := range latestByUser {
		for _, item := range est.Items {
			acc, exists := taskMap[item.TaskName]
			if !exists {
				acc = &accumulator{
					minOfMins:  math.MaxFloat64,
					maxOfMaxes: -math.MaxFloat64,
					users:      make(map[string]bool),
					firstOrder: item.SortOrder,
				}
				taskMap[item.TaskName] = acc
				taskOrder = append(taskOrder, item.TaskName)
			}
			pert := item.PERTHours()
			acc.pertSum += pert
			acc.count++
			acc.users[est.Estimation.SubmittedBy] = true
			if item.MinHours < acc.minOfMins {
				acc.minOfMins = item.MinHours
			}
			if item.MaxHours > acc.maxOfMaxes {
				acc.maxOfMaxes = item.MaxHours
			}
		}
	}

	result := &AggregatedResult{}
	for _, taskName := range taskOrder {
		acc := taskMap[taskName]
		item := &AggregatedItem{
			TaskName:       taskName,
			AvgPERTHours:   acc.pertSum / float64(acc.count),
			MinOfMins:      acc.minOfMins,
			MaxOfMaxes:     acc.maxOfMaxes,
			EstimatorCount: len(acc.users),
		}
		result.Items = append(result.Items, item)
		result.TotalHours += item.AvgPERTHours
	}

	return result
}
