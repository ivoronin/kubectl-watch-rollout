package monitor

// Constants vendored from:
// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/deployment/util/deployment_util.go
const (
	// RevisionAnnotation is the revision annotation of a deployment's replica sets which records its rollout sequence
	RevisionAnnotation = "deployment.kubernetes.io/revision"

	// NewReplicaSetAvailable means the deployment has a new RS and all pods are available.
	NewReplicaSetAvailable = "NewReplicaSetAvailable"

	// ProgressDeadlineExceeded means the deployment has failed to progress.
	ProgressDeadlineExceeded = "ProgressDeadlineExceeded"

	// Kubernetes defaults
	defaultReplicaCount = 1 // Default when deployment.spec.replicas is nil
	minActiveReplicas   = 0 // ReplicaSets with > 0 replicas are considered active

	// Parsing constants
	parseIntBase10 = 10 // Decimal base for string to int conversion
	parseIntBits64 = 64 // 64-bit integer size for parsing
)
