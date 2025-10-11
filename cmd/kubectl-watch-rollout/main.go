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
	var untilComplete bool

	cmd := &cobra.Command{
		Use:   "kubectl watch-rollout DEPLOYMENT",
		Short: "Watch Kubernetes deployment rollouts with live progress updates",
		Long: `Watch Kubernetes deployment rollouts with live progress updates and status tracking.

By default, monitors deployments continuously across multiple rollouts. Exit with Ctrl+C when done.

This command monitors your deployment rollout in real-time, showing:
  • Progress bars for new and old ReplicaSets
  • Pod status counts (Available, Ready, Current)
  • Warning events and error messages
  • Estimated time to completion
  • Automatic detection of rollout success or failure
  • Continuous monitoring across multiple rollouts (default behavior)`,
		Example: `  # Continuous monitoring (default) - watches across multiple rollouts
  kubectl watch-rollout my-deployment -n production

  # Single-rollout mode - exit after one rollout completes (for CI/CD)
  kubectl watch-rollout my-deployment -n production --until-complete

  # Watch using resource type prefix (kubectl-style)
  kubectl watch-rollout deployment/my-deployment -n production
  kubectl watch-rollout deployments.apps/my-deployment -n production`,
		Args:              cobra.ExactArgs(1),
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

			// Parse deployment name from argument
			deploymentName, err := parseDeploymentArg(args[0])
			if err != nil {
				return err
			}

			// Create repository
			repo := monitor.NewDeploymentRepository(clientset, namespace)

			// Create monitor with configuration
			cfg := monitor.DefaultConfig()
			cfg.UntilComplete = untilComplete

			m, err := monitor.NewWithConfig(repo, deploymentName, cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize monitoring: %w", err)
			}

			return m.Run(ctx)
		},
	}

	configFlags.AddFlags(cmd.Flags())
	cmd.Flags().BoolVar(&untilComplete, "until-complete", false, "Exit after monitoring one rollout to completion (default: continuous monitoring)")

	if err := cmd.Execute(); err != nil {
		// Silent exit for progress deadline exceeded
		if !errors.Is(err, monitor.ErrProgressDeadlineExceeded) {
			fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		}

		os.Exit(1)
	}
}
