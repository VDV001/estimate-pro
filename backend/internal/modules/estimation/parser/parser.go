package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/modules/estimation/domain"
)

// knownHeaders are column names that indicate a header row.
var knownHeaders = map[string]bool{
	"task_name": true, "task": true, "name": true, "задача": true,
}

// Parse parses a CSV-like text into estimation items.
// Format: task_name,min_hours,likely_hours,max_hours[,note]
// Skips empty lines and recognized header rows.
func Parse(input string) ([]*domain.EstimationItem, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("parser.Parse: empty input")
	}

	lines := strings.Split(input, "\n")
	var items []*domain.EstimationItem
	order := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			return nil, fmt.Errorf("parser.Parse: line %d: expected at least 4 columns (task,min,likely,max), got %d", i+1, len(parts))
		}

		taskName := strings.TrimSpace(parts[0])

		// Skip header row.
		if order == 0 && knownHeaders[strings.ToLower(taskName)] {
			continue
		}

		minH, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("parser.Parse: line %d: invalid min_hours %q: %w", i+1, parts[1], err)
		}
		likelyH, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return nil, fmt.Errorf("parser.Parse: line %d: invalid likely_hours %q: %w", i+1, parts[2], err)
		}
		maxH, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err != nil {
			return nil, fmt.Errorf("parser.Parse: line %d: invalid max_hours %q: %w", i+1, parts[3], err)
		}

		if minH < 0 || likelyH < 0 || maxH < 0 {
			return nil, fmt.Errorf("parser.Parse: line %d: hours must be non-negative", i+1)
		}
		if minH > likelyH {
			return nil, fmt.Errorf("parser.Parse: line %d: min_hours (%v) must be <= likely_hours (%v)", i+1, minH, likelyH)
		}
		if likelyH > maxH {
			return nil, fmt.Errorf("parser.Parse: line %d: likely_hours (%v) must be <= max_hours (%v)", i+1, likelyH, maxH)
		}

		var note string
		if len(parts) >= 5 {
			note = strings.TrimSpace(parts[4])
		}

		items = append(items, &domain.EstimationItem{
			TaskName:    taskName,
			MinHours:    minH,
			LikelyHours: likelyH,
			MaxHours:    maxH,
			SortOrder:   order,
			Note:        note,
		})
		order++
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("parser.Parse: no valid items found")
	}

	return items, nil
}
