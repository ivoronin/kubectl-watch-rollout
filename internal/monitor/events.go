// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains centralized event processing logic used by both renderers.
package monitor

import (
	"regexp"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/xrash/smetrics"
)

const (
	// maxMessageLength is the maximum length for event messages before truncation
	maxMessageLength = 80
)

// groupData holds messages and their timestamps for clustering.
type groupData struct {
	messages []string
	times    []time.Time
}

// SummarizeEvents takes raw K8s events and produces formatted output ready for rendering.
// This is the single entry point for all event processing:
// 1. Filter by ignoreRegex, group by Type+Reason
// 2. Cluster similar messages within each group
// 3. Sort by priority (warnings first, then by count)
func SummarizeEvents(events []corev1.Event, ignoreRegex *regexp.Regexp, threshold float64) EventSummary {
	if len(events) == 0 {
		return EventSummary{}
	}

	// Step 1: Filter by ignoreRegex, group by Type+Reason
	type typeReasonKey struct{ Type, Reason string }
	groups := make(map[typeReasonKey]*groupData)
	ignoredCount := 0

	for _, event := range events {
		if ignoreRegex != nil {
			fullMsg := event.Reason + ": " + event.Message
			if ignoreRegex.MatchString(fullMsg) {
				ignoredCount++
				continue
			}
		}
		key := typeReasonKey{Type: event.Type, Reason: event.Reason}
		if groups[key] == nil {
			groups[key] = &groupData{}
		}
		groups[key].messages = append(groups[key].messages, event.Message)
		groups[key].times = append(groups[key].times, getEventTime(&event))
	}

	if len(groups) == 0 {
		return EventSummary{IgnoredCount: ignoredCount}
	}

	// Step 2: Cluster similar messages and build EventClusters
	var result []EventCluster
	for key, group := range groups {
		clusters := clusterMessages(group, threshold, key.Type, key.Reason)
		result = append(result, clusters...)
	}

	// Step 3: Sort - Warning first, then by count descending, then by reason
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type == corev1.EventTypeWarning
		}
		if result[i].LookAlikeCount != result[j].LookAlikeCount {
			return result[i].LookAlikeCount > result[j].LookAlikeCount
		}
		return result[i].Reason < result[j].Reason
	})

	return EventSummary{
		Clusters:     result,
		IgnoredCount: ignoredCount,
	}
}

// getEventTime returns the best available timestamp for an event.
// Prefers LastTimestamp, falls back to EventTime, then CreationTimestamp.
func getEventTime(evt *corev1.Event) time.Time {
	if !evt.LastTimestamp.IsZero() {
		return evt.LastTimestamp.Time
	}
	if !evt.EventTime.IsZero() {
		return evt.EventTime.Time
	}
	return evt.CreationTimestamp.Time
}

// clusterMessages groups similar messages using Jaro-Winkler similarity.
// Returns EventClusters ready for display.
func clusterMessages(group *groupData, threshold float64, eventType, reason string) []EventCluster {
	if len(group.messages) == 0 {
		return nil
	}

	type clusterData struct {
		count    int
		lastSeen time.Time
	}
	clusters := make(map[string]*clusterData)

	for i, msg := range group.messages {
		truncatedMsg := truncateMessage(msg)
		msgTime := group.times[i]

		bestMatch := ""
		bestSimilarity := 0.0

		// Find the most similar existing cluster
		for existing := range clusters {
			similarity := stringSimilarity(truncatedMsg, existing)
			if similarity >= threshold && similarity > bestSimilarity {
				bestMatch = existing
				bestSimilarity = similarity
			}
		}

		if bestMatch != "" {
			clusters[bestMatch].count++
			if msgTime.After(clusters[bestMatch].lastSeen) {
				clusters[bestMatch].lastSeen = msgTime
			}
		} else {
			clusters[truncatedMsg] = &clusterData{count: 1, lastSeen: msgTime}
		}
	}

	// Build EventClusters
	result := make([]EventCluster, 0, len(clusters))
	for msg, data := range clusters {
		lookAlikeCount := 0
		if data.count > 1 {
			lookAlikeCount = data.count - 1
		}
		result = append(result, EventCluster{
			Type:           eventType,
			Reason:         reason,
			Message:        msg,
			LookAlikeCount: lookAlikeCount,
			LastSeen:       data.lastSeen,
		})
	}

	return result
}

// truncateMessage shortens a message to maxMessageLength, adding ellipsis if truncated.
// Also sanitizes by replacing newlines with spaces.
func truncateMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", " ")

	if len(msg) <= maxMessageLength {
		return msg
	}
	return msg[:maxMessageLength-3] + "..."
}

// stringSimilarity calculates normalized similarity between two strings.
// Returns value between 0 (completely different) and 1 (identical).
// Strings should be pre-truncated for consistent comparison.
//
// Uses Jaro-Winkler algorithm which gives more weight to common prefixes.
// This is ideal for K8s event messages which typically share a common prefix
// (e.g., "Successfully assigned default/nginx-xxx to node-yyy") with varying
// suffixes (pod names, node names).
func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// JaroWinkler with extended prefix size for K8s messages
	return smetrics.JaroWinkler(a, b, 0.7, 20)
}
