// Package monitor provides Kubernetes deployment rollout monitoring functionality.
//
// This file contains the repository layer for Kubernetes API access.
package monitor

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeploymentRepository handles all Kubernetes API interactions.
// Implements the repository pattern, isolating API concerns from business logic.
type DeploymentRepository struct {
	clientset *kubernetes.Clientset
	namespace string
}

// NewDeploymentRepository creates a new repository instance
func NewDeploymentRepository(clientset *kubernetes.Clientset, namespace string) *DeploymentRepository {
	return &DeploymentRepository{
		clientset: clientset,
		namespace: namespace,
	}
}

// GetDeployment retrieves a deployment by name from the Kubernetes API
func (r *DeploymentRepository) GetDeployment(ctx context.Context, name string) (*appsv1.Deployment, error) {
	deployment, err := r.clientset.AppsV1().Deployments(r.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("deployment '%s' not found in namespace '%s': %w", name, r.namespace, err)
	}
	return deployment, nil
}

// FindActiveRollout finds the first deployment with an active rollout in the namespace.
// Returns empty string if no active rollouts are found.
func (r *DeploymentRepository) FindActiveRollout(ctx context.Context) (string, error) {
	deployments, err := r.clientset.AppsV1().Deployments(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list deployments in namespace '%s': %w", r.namespace, err)
	}

	for _, deployment := range deployments.Items {
		status := deployment.Status
		// Active rollout = not complete and not failed
		if !isDeploymentComplete(status) && !isDeploymentFailed(status) {
			return deployment.Name, nil
		}
	}

	return "", nil
}

// GetReplicaSets returns old and new ReplicaSets for a deployment.
// Returns the newest ReplicaSet and a list of older active ReplicaSets.
func (r *DeploymentRepository) GetReplicaSets(ctx context.Context, deployment *appsv1.Deployment) ([]*appsv1.ReplicaSet, *appsv1.ReplicaSet, error) {
	replicaSets, err := r.clientset.AppsV1().ReplicaSets(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch ReplicaSets for deployment '%s': %w", deployment.Name, err)
	}

	var oldRS []*appsv1.ReplicaSet
	var newRS *appsv1.ReplicaSet
	maxRevision := int64(0)

	for i := range replicaSets.Items {
		rs := &replicaSets.Items[i]

		// Check if owned by deployment
		if !metav1.IsControlledBy(rs, deployment) {
			continue
		}

		// Get revision
		revision := int64(0)
		if revisionStr, ok := rs.Annotations[RevisionAnnotation]; ok {
			parsedRev, err := strconv.ParseInt(revisionStr, parseIntBase10, parseIntBits64)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse revision %q for ReplicaSet %s in deployment '%s': %w",
					revisionStr, rs.Name, deployment.Name, err)
			}
			revision = parsedRev
		}

		if revision > maxRevision {
			if newRS != nil {
				oldRS = append(oldRS, newRS)
			}
			maxRevision = revision
			newRS = rs
		} else {
			oldRS = append(oldRS, rs)
		}
	}

	return filterActiveReplicaSets(oldRS), newRS, nil
}

// filterActiveReplicaSets returns only ReplicaSets with desired replicas > 0.
// Inactive ReplicaSets (scaled to 0) excluded for clearer OLD metrics visualization.
func filterActiveReplicaSets(rss []*appsv1.ReplicaSet) []*appsv1.ReplicaSet {
	var active []*appsv1.ReplicaSet
	for _, rs := range rss {
		if getInt32OrDefault(rs.Spec.Replicas, minActiveReplicas) > minActiveReplicas {
			active = append(active, rs)
		}
	}
	return active
}

// GetPodWarnings fetches warning events for pods belonging to a specific ReplicaSet.
// Returns a map of warning messages to their count in the current poll cycle.
func (r *DeploymentRepository) GetPodWarnings(ctx context.Context, rs *appsv1.ReplicaSet) (map[string]int, error) {
	// Use label selector from ReplicaSet to filter pods
	labelSelector := metav1.FormatLabelSelector(rs.Spec.Selector)
	pods, err := r.clientset.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pods for ReplicaSet '%s': %w", rs.Name, err)
	}

	// Build set of pod names for this ReplicaSet
	podNames := make(map[string]struct{}, len(pods.Items))
	for _, pod := range pods.Items {
		// Double-check ownership to be safe
		if metav1.IsControlledBy(&pod, rs) {
			podNames[pod.Name] = struct{}{}
		}
	}

	// Fetch only Warning events for Pods
	eventList, err := r.clientset.CoreV1().Events(r.namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning,involvedObject.kind=Pod",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pod events: %w", err)
	}

	// Count warnings in this poll cycle
	warningCounts := make(map[string]int)
	for _, event := range eventList.Items {
		// Check if event is for one of our pods
		if _, ok := podNames[event.InvolvedObject.Name]; !ok {
			continue
		}

		warningMsg := fmt.Sprintf("%s: %s", event.Reason, event.Message)
		warningCounts[warningMsg]++
	}

	return warningCounts, nil
}
