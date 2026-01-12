// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains event processing logic using the Drain log parsing algorithm.
package monitor

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/faceair/drain"
	"github.com/ivoronin/kubectl-watch-rollout/internal/types"
	corev1 "k8s.io/api/core/v1"
)

// maskPattern defines a regex pattern and its semantic replacement token.
type maskPattern struct {
	pattern *regexp.Regexp
	token   string
}

// Pre-compiled K8s masking patterns for clustering.
// Order matters: more specific patterns first.
var defaultMaskPatterns = []maskPattern{
	// AWS ECR images (incl China): 123456789.dkr.ecr.region.amazonaws.com[.cn]/path/image:tag → <IMAGE>
	{regexp.MustCompile(`\b\d+\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com(?:\.cn)?/[a-zA-Z0-9/:._@-]+`), "<IMAGE>"},
	// Namespace/pod paths: default/nginx-abc123-xyz → <NS>/<POD>
	{regexp.MustCompile(`\b[a-z0-9-]+/[a-z0-9]+-[a-z0-9]{5,10}-[a-z0-9]{5}\b`), "<NS>/<POD>"},
	// Pod names: nginx-deployment-abc123-xyz → <POD>
	{regexp.MustCompile(`\b[a-z0-9]+-[a-z0-9]{5,10}-[a-z0-9]{5}\b`), "<POD>"},
	// AWS EC2 node names: ip-172-28-129-199.ec2.internal → <NODE>
	{regexp.MustCompile(`\bip-\d+-\d+-\d+-\d+\.[a-z0-9.-]+\.internal\b`), "<NODE>"},
	// GKE node names: gke-cluster-default-pool-abc123 → <NODE>
	{regexp.MustCompile(`\bgke-[a-z0-9-]+\b`), "<NODE>"},
	// IP with port: 10.0.0.1:8080 → <IP:PORT>
	{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+\b`), "<IP:PORT>"},
	// IP without port: 10.0.0.1 → <IP>
	{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "<IP>"},
	// UUIDs: 550e8400-e29b-41d4-a716-446655440000 → <UUID>
	{regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`), "<UUID>"},
}

// groupData holds messages and timestamps for clustering.
type groupData struct {
	messages []string
	times    []time.Time
}

// SummarizeEvents processes K8s events into clustered summary.
func SummarizeEvents(events []corev1.Event, ignoreRegex *regexp.Regexp, threshold float64) types.EventSummary {
	if len(events) == 0 {
		return types.EventSummary{}
	}

	// Group by Type+Reason, apply ignore filter
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

// clusterMeta tracks data for a cluster.
type clusterMeta struct {
	times []time.Time
}

// clusterWithDrain uses Drain algorithm to cluster similar messages.
func clusterWithDrain(group *groupData, threshold float64, eventType, reason string) []types.EventCluster {
	if len(group.messages) == 0 {
		return nil
	}

	// Configure Drain
	config := drain.DefaultConfig()
	config.SimTh = threshold
	d := drain.New(config)

	// Phase 1: Train all messages first (templates evolve as Drain learns)
	type msgCluster struct {
		cluster *drain.LogCluster
		time    time.Time
	}
	trained := make([]msgCluster, len(group.messages))

	for i, msg := range group.messages {
		masked := maskMessage(sanitizeMessage(msg))
		trained[i] = msgCluster{
			cluster: d.Train(masked),
			time:    group.times[i],
		}
	}

	// Phase 2: Now read FINAL templates (after all learning is done)
	// Use cluster pointer as key since template strings can change during training
	clusterMetas := make(map[*drain.LogCluster]*clusterMeta)

	for _, tc := range trained {
		if clusterMetas[tc.cluster] == nil {
			clusterMetas[tc.cluster] = &clusterMeta{}
		}
		clusterMetas[tc.cluster].times = append(clusterMetas[tc.cluster].times, tc.time)
	}

	// Phase 3: Build EventClusters with FINAL templates
	result := make([]types.EventCluster, 0, len(clusterMetas))

	for cluster, meta := range clusterMetas {
		// Find latest timestamp
		var lastSeen time.Time
		for _, t := range meta.times {
			if t.After(lastSeen) {
				lastSeen = t
			}
		}

		result = append(result, types.EventCluster{
			Type:          eventType,
			Reason:        reason,
			Message:       extractTemplate(cluster.String()), // Read final evolved template
			ExemplarCount: len(meta.times),
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

// maskMessage applies K8s-specific masking patterns for clustering.
func maskMessage(msg string) string {
	for _, p := range defaultMaskPatterns {
		msg = p.pattern.ReplaceAllString(msg, p.token)
	}
	return msg
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
