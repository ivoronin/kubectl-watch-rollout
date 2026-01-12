package monitor

// This file contains event processing logic using the Drain log parsing algorithm.

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/faceair/drain"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
	corev1 "k8s.io/api/core/v1"
)

// eventData holds a single event's message and timestamp for clustering.
type eventData struct {
	message string
	time    time.Time
}

// SummarizeEvents processes K8s events into clustered summary.
func SummarizeEvents(events []corev1.Event, ignoreRegex *regexp.Regexp, threshold float64) types.EventSummary {
	if len(events) == 0 {
		return types.EventSummary{}
	}

	// Group by Type+Reason, apply ignore filter
	type typeReasonKey struct{ Type, Reason string }

	groups := make(map[typeReasonKey][]eventData)
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
		groups[key] = append(groups[key], eventData{
			message: event.Message,
			time:    getEventTime(&event),
		})
	}

	if len(groups) == 0 {
		return types.EventSummary{IgnoredCount: ignoredCount}
	}

	// Cluster each group using Drain
	var result []types.EventCluster

	for key, group := range groups {
		clusters := clusterWithDrain(group, threshold, key.Type, key.Reason)
		result = append(result, clusters...)
	}

	// Sort: warnings first, then by count, then by reason
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type == corev1.EventTypeWarning
		}

		if result[i].ExemplarCount != result[j].ExemplarCount {
			return result[i].ExemplarCount > result[j].ExemplarCount
		}

		return result[i].Reason < result[j].Reason
	})

	return types.EventSummary{
		Clusters:     result,
		IgnoredCount: ignoredCount,
	}
}

// clusterWithDrain uses Drain algorithm to cluster similar messages.
func clusterWithDrain(events []eventData, threshold float64, eventType, reason string) []types.EventCluster {
	if len(events) == 0 {
		return nil
	}

	// Configure Drain
	config := drain.DefaultConfig()
	config.SimTh = threshold
	d := drain.New(config)

	// Phase 1: Train all messages first (templates evolve as Drain learns)
	type trainedEvent struct {
		cluster *drain.LogCluster
		time    time.Time
	}

	trained := make([]trainedEvent, len(events))

	for i, evt := range events {
		trained[i] = trainedEvent{
			cluster: d.Train(sanitizeMessage(evt.message)),
			time:    evt.time,
		}
	}

	// Phase 2: Group by final cluster (templates may have evolved during training)
	clusterTimes := make(map[*drain.LogCluster][]time.Time)

	for _, te := range trained {
		clusterTimes[te.cluster] = append(clusterTimes[te.cluster], te.time)
	}

	// Phase 3: Build EventClusters with final templates
	result := make([]types.EventCluster, 0, len(clusterTimes))

	for cluster, times := range clusterTimes {
		// Find latest timestamp
		var lastSeen time.Time
		for _, t := range times {
			if t.After(lastSeen) {
				lastSeen = t
			}
		}

		result = append(result, types.EventCluster{
			Type:          eventType,
			Reason:        reason,
			Message:       extractTemplate(cluster.String()),
			ExemplarCount: len(times),
			LastSeen:      lastSeen,
		})
	}

	return result
}

// extractTemplate extracts template from Drain's String() format.
// Input:  "id={1} : size={3} : template content here"
// Output: "template content here"
func extractTemplate(s string) string {
	const sep = " : "

	idx := strings.LastIndex(s, sep)
	if idx == -1 {
		return s
	}

	return s[idx+len(sep):]
}

// getEventTime returns the best timestamp for an event.
func getEventTime(evt *corev1.Event) time.Time {
	if !evt.LastTimestamp.IsZero() {
		return evt.LastTimestamp.Time
	}

	if !evt.EventTime.IsZero() {
		return evt.EventTime.Time
	}

	return evt.CreationTimestamp.Time
}

// sanitizeMessage normalizes whitespace.
func sanitizeMessage(msg string) string {
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", " ")

	return msg
}
