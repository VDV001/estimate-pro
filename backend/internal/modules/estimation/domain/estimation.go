// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package domain

import "time"

type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
)

func (s Status) IsValid() bool {
	return s == StatusDraft || s == StatusSubmitted
}

type Estimation struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"project_id"`
	DocumentVersionID string    `json:"document_version_id,omitempty"`
	SubmittedBy       string    `json:"submitted_by"`
	Status            Status    `json:"status"`
	SubmittedAt       time.Time `json:"submitted_at,omitzero"`
	CreatedAt         time.Time `json:"created_at,omitzero"`
}

func (e *Estimation) IsSubmitted() bool {
	return e.Status == StatusSubmitted
}

type EstimationItem struct {
	ID           string  `json:"id"`
	EstimationID string  `json:"estimation_id"`
	TaskName     string  `json:"task_name"`
	MinHours     float64 `json:"min_hours"`
	LikelyHours  float64 `json:"likely_hours"`
	MaxHours     float64 `json:"max_hours"`
	SortOrder    int     `json:"sort_order"`
	Note         string  `json:"note,omitempty"`
}

// PERTHours returns the PERT estimate: (min + 4*likely + max) / 6.
func (item *EstimationItem) PERTHours() float64 {
	return (item.MinHours + 4*item.LikelyHours + item.MaxHours) / 6
}

type EstimationWithItems struct {
	Estimation *Estimation      `json:"estimation"`
	Items      []*EstimationItem `json:"items"`
}
