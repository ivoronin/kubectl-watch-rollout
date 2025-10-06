# kubectl-watch-rollout

[![GitHub release](https://img.shields.io/github/v/release/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivoronin/kubectl-watch-rollout)](https://goreportcard.com/report/github.com/ivoronin/kubectl-watch-rollout)
[![GitHub last commit](https://img.shields.io/github/last-commit/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout/commits/master)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/ivoronin/kubectl-watch-rollout/goreleaser.yml)](https://github.com/ivoronin/kubectl-watch-rollout/actions)
[![GitHub top language](https://img.shields.io/github/languages/top/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout)

Watch Kubernetes deployment rollouts with live progress updates and status tracking.

![Screenshot](https://raw.githubusercontent.com/ivoronin/kubectl-watch-rollout/master/screenshot.png)

## Features

- **Real-time monitoring** of deployment rollout progress
- **Live progress bars** for new and old ReplicaSets showing pod lifecycle stages (Current → Ready → Available)
- **Pod status counts** with detailed breakdown of Available, Ready, and Current pods
- **Warning events** from pods with automatic aggregation and deduplication
- **Estimated time to completion** based on current rollout velocity
- **Progress deadline tracking** with warnings when approaching timeout
- **Automatic detection** of rollout success or failure
- **Auto-discovery mode** - automatically finds and monitors active rollouts when no deployment specified

## Installation

### Using Krew (recommended)

```bash
kubectl krew install watch-rollout
```

### Manual Installation

Download the latest binary from the [releases page](https://github.com/ivoronin/kubectl-watch-rollout/releases) and place it in your `PATH`.

### Building from Source

```bash
git clone https://github.com/ivoronin/kubectl-watch-rollout.git
cd kubectl-watch-rollout
make build
```

## Usage

### Watch a specific deployment

```bash
# By name
kubectl watch-rollout my-deployment -n production

# With resource type prefix (kubectl-style)
kubectl watch-rollout deployment/my-deployment -n production
kubectl watch-rollout deployments.apps/my-deployment -n production
```

### Auto-discover and watch active rollouts

```bash
# Automatically finds the first active rollout in namespace
kubectl watch-rollout -n production
```

### Using kubeconfig context

```bash
kubectl watch-rollout my-app --context=prod-cluster -n production
```

## Configuration

The tool uses your current kubeconfig configuration. You can specify a different context, namespace, or kubeconfig file using standard kubectl flags:

```bash
kubectl watch-rollout --kubeconfig=/path/to/config --context=my-context -n my-namespace
```
