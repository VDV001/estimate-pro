// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import (
	"math"
	"slices"
)

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

	// Collect all submitted estimations per user, sorted newest-first.
	// For each user+task combination, use only the newest item (dedup by task name).
	userEstimations := make(map[string][]*EstimationWithItems)
	for _, est := range estimations {
		if est.Estimation.Status != StatusSubmitted {
			continue
		}
		userEstimations[est.Estimation.SubmittedBy] = append(
			userEstimations[est.Estimation.SubmittedBy], est,
		)
	}

	for userID, ests := range userEstimations {
		// Sort newest-first so the first occurrence of a task_name wins.
		slices.SortFunc(ests, func(a, b *EstimationWithItems) int {
			return b.Estimation.CreatedAt.Compare(a.Estimation.CreatedAt)
		})

		seen := make(map[string]bool)
		for _, est := range ests {
			for _, item := range est.Items {
				if seen[item.TaskName] {
					continue // already have a newer estimate for this task
				}
				seen[item.TaskName] = true

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
				acc.users[userID] = true
				if item.MinHours < acc.minOfMins {
					acc.minOfMins = item.MinHours
				}
				if item.MaxHours > acc.maxOfMaxes {
					acc.maxOfMaxes = item.MaxHours
				}
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
