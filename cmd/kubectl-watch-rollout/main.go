// Package main implements the kubectl-watch-rollout plugin.
//
// This kubectl plugin provides real-time monitoring of Kubernetes deployment rollouts,
// displaying progress bars, pod status, warnings, and completion estimates.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ivoronin/kubectl-watch-rollout/internal/monitor"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// parseDeploymentArg extracts deployment name from argument with optional resource type prefix.
// Supports kubectl-style specs like "deployment/my-app" or "deployments.apps/my-app".
// Returns extracted deployment name or error if format invalid or not a deployment type.
func parseDeploymentArg(arg string) (string, error) {
	if !strings.Contains(arg, "/") {
		return arg, nil // No prefix, return as-is
	}

	parts := strings.Split(arg, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid resource format '%s': expected TYPE/NAME (e.g., deployment/my-app)", arg)
	}

	resourceType, name := parts[0], parts[1]

	// Validate it's a deployment resource type
	validTypes := []string{
		"deployment", "deployments", "deploy",
		"deployment.apps", "deployments.apps",
		"deployment.v1.apps", "deployments.v1.apps",
	}

	isValid := false
	for _, validType := range validTypes {
		if resourceType == validType {
			isValid = true
			break
		}
	}

	if !isValid {
		return "", fmt.Errorf("resource type '%s' is not a deployment (use: deployment, deployments, or deploy)", resourceType)
	}

	return name, nil
}

func main() {
	configFlags := genericclioptions.NewConfigFlags(true)

	cmd := &cobra.Command{
		Use:   "kubectl watch-rollout [DEPLOYMENT]",
		Short: "Watch Kubernetes deployment rollouts with live progress updates",
		Long: `Watch Kubernetes deployment rollouts with live progress updates and status tracking.

This command monitors your deployment rollout in real-time, showing:
  • Progress bars for new and old ReplicaSets
  • Pod status counts (Available, Ready, Current)
  • Warning events and error messages
  • Estimated time to completion
  • Automatic detection of rollout success or failure

When DEPLOYMENT is not specified, the command automatically discovers and monitors
the first active rollout in your namespace. An "active rollout" is a deployment
currently transitioning between ReplicaSet versions.`,
		Example: `  # Watch a specific deployment by name
  kubectl watch-rollout my-deployment -n production

  # Watch using resource type prefix (kubectl-style)
  kubectl watch-rollout deployment/my-deployment -n production
  kubectl watch-rollout deployments.apps/my-deployment -n production

  # Auto-discover and watch the first active rollout in namespace
  kubectl watch-rollout -n production

  # Watch in default namespace
  kubectl watch-rollout my-app`,
		Args:              cobra.MaximumNArgs(1),
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			restConfig, err := configFlags.ToRESTConfig()
			if err != nil {
				return fmt.Errorf("failed to load kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				return fmt.Errorf("failed to connect to Kubernetes cluster (check cluster access and credentials): %w", err)
			}

			namespace, _, err := configFlags.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return fmt.Errorf("failed to determine namespace (use -n flag to specify): %w", err)
			}

			// Setup signal handling for graceful shutdown
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			// Create repository
			repo := monitor.NewDeploymentRepository(clientset, namespace)

			// Determine deployment name (from arg or auto-discovery)
			var deploymentName string
			if len(args) > 0 {
				deploymentName, err = parseDeploymentArg(args[0])
				if err != nil {
					return err
				}
			} else {
				// Auto-discover first active rollout
				deploymentName, err = repo.FindActiveRollout(ctx)
				if err != nil {
					return fmt.Errorf("failed to search for active rollouts: %w", err)
				}
				if deploymentName == "" {
					fmt.Println("No active deployment rollouts found in this namespace.")
					return nil
				}
			}

			m, err := monitor.New(repo, deploymentName)
			if err != nil {
				return fmt.Errorf("failed to initialize monitoring: %w", err)
			}

			return m.Run(ctx)
		},
	}

	configFlags.AddFlags(cmd.Flags())

	if err := cmd.Execute(); err != nil {
		// Silent exit for progress deadline exceeded
		if !errors.Is(err, monitor.ErrProgressDeadlineExceeded) {
			fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		}

		os.Exit(1)
	}
}
