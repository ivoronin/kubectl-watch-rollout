// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains shared formatting utilities for rendering.
package monitor

import (
	"fmt"
	"time"
)

// FormatDuration formats duration with seconds precision.
// Examples: "1h10m", "1h10m30s", "20m", "20m30s", "45s"
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		if m > 0 {
			if s > 0 {
				return fmt.Sprintf("%dh%dm%ds", h, m, s)
			}
			return fmt.Sprintf("%dh%dm", h, m)
		}
		if s > 0 {
			return fmt.Sprintf("%dh%ds", h, s)
		}
		return fmt.Sprintf("%dh", h)
	}

	if m > 0 {
		if s > 0 {
			return fmt.Sprintf("%dm%ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}

	return fmt.Sprintf("%ds", s)
}
