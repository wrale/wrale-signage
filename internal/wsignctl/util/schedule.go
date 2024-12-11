// Package util provides shared utilities for the CLI
package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// ParseSchedule parses schedule components into a Schedule object
func ParseSchedule(startTime, endTime string, daysOfWeek []string, timeOfDay string) (*v1alpha1.Schedule, error) {
	if startTime == "" && endTime == "" && len(daysOfWeek) == 0 && timeOfDay == "" {
		return nil, nil
	}

	schedule := &v1alpha1.Schedule{}

	// Parse start/end times
	if startTime != "" {
		t, err := time.Parse(time.RFC3339, startTime)
		if err != nil {
			return nil, fmt.Errorf("invalid start time %q: %w", startTime, err)
		}
		schedule.ActiveFrom = &t
	}
	if endTime != "" {
		t, err := time.Parse(time.RFC3339, endTime)
		if err != nil {
			return nil, fmt.Errorf("invalid end time %q: %w", endTime, err)
		}
		schedule.ActiveUntil = &t
	}

	// Parse days of week
	if len(daysOfWeek) > 0 {
		for _, day := range daysOfWeek {
			switch strings.ToLower(day) {
			case "sun", "sunday":
				schedule.DaysOfWeek = append(schedule.DaysOfWeek, time.Sunday)
			case "mon", "monday":
				schedule.DaysOfWeek = append(schedule.DaysOfWeek, time.Monday)
			case "tue", "tuesday":
				schedule.DaysOfWeek = append(schedule.DaysOfWeek, time.Tuesday)
			case "wed", "wednesday":
				schedule.DaysOfWeek = append(schedule.DaysOfWeek, time.Wednesday)
			case "thu", "thursday":
				schedule.DaysOfWeek = append(schedule.DaysOfWeek, time.Thursday)
			case "fri", "friday":
				schedule.DaysOfWeek = append(schedule.DaysOfWeek, time.Friday)
			case "sat", "saturday":
				schedule.DaysOfWeek = append(schedule.DaysOfWeek, time.Saturday)
			default:
				return nil, fmt.Errorf("invalid day of week: %s", day)
			}
		}
	}

	// Parse time of day range
	if timeOfDay != "" {
		parts := strings.Split(timeOfDay, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid time range %q - use format HH:MM-HH:MM", timeOfDay)
		}
		schedule.TimeOfDay = &v1alpha1.TimeRange{
			Start: parts[0],
			End:   parts[1],
		}
	}

	return schedule, nil
}

// FormatSchedule formats a schedule for display
func FormatSchedule(s *v1alpha1.Schedule) string {
	if s == nil {
		return "Always"
	}

	var parts []string

	if s.ActiveFrom != nil {
		parts = append(parts, fmt.Sprintf("from %s", s.ActiveFrom.Format(time.RFC3339)))
	}
	if s.ActiveUntil != nil {
		parts = append(parts, fmt.Sprintf("until %s", s.ActiveUntil.Format(time.RFC3339)))
	}

	if len(s.DaysOfWeek) > 0 {
		var days []string
		for _, d := range s.DaysOfWeek {
			days = append(days, d.String()[:3])
		}
		parts = append(parts, fmt.Sprintf("on %s", strings.Join(days, ",")))
	}

	if s.TimeOfDay != nil {
		parts = append(parts, fmt.Sprintf("at %s-%s",
			s.TimeOfDay.Start,
			s.TimeOfDay.End))
	}

	if len(parts) == 0 {
		return "Always"
	}
	return strings.Join(parts, " ")
}

// FormatSelectors formats display selectors for display
func FormatSelectors(s v1alpha1.DisplaySelector) string {
	var parts []string

	if s.SiteID != "" {
		parts = append(parts, fmt.Sprintf("site=%s", s.SiteID))
	}
	if s.Zone != "" {
		parts = append(parts, fmt.Sprintf("zone=%s", s.Zone))
	}
	if s.Position != "" {
		parts = append(parts, fmt.Sprintf("pos=%s", s.Position))
	}

	if len(parts) == 0 {
		return "*"
	}
	return strings.Join(parts, ",")
}
