# kubectl-watch-rollout

[![GitHub release](https://img.shields.io/github/v/release/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivoronin/kubectl-watch-rollout)](https://goreportcard.com/report/github.com/ivoronin/kubectl-watch-rollout)
[![GitHub last commit](https://img.shields.io/github/last-commit/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout/commits/master)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/ivoronin/kubectl-watch-rollout/goreleaser.yml)](https://github.com/ivoronin/kubectl-watch-rollout/actions)
[![GitHub top language](https://img.shields.io/github/languages/top/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout)

Watch Kubernetes deployment rollouts with live progress updates and status tracking.

![Screenshot](https://raw.githubusercontent.com/ivoronin/kubectl-watch-rollout/master/screenshot.png)

## Features

- **Continuous monitoring** - watches deployments across multiple rollouts by default
- **Real-time monitoring** of deployment rollout progress
- **Live progress bars** for new and old ReplicaSets showing pod lifecycle stages (Current → Ready → Available)
- **Pod status counts** with detailed breakdown of Available, Ready, and Current pods
- **Warning events** from pods with automatic aggregation and deduplication
- **Estimated time to completion** based on current rollout velocity
- **Progress deadline detection** for failed rollouts
- **Automatic detection** of rollout success or failure
- **Automatic rollout detection** - detects and monitors new rollouts as they occur
- **Single-rollout mode** - use `--until-complete` flag to exit after one rollout (for CI/CD automation)
- **Line mode** - use `--line-mode` flag for timestamped, parseable output perfect for CI/CD pipelines and log aggregation

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

### Watch a deployment continuously (default)

By default, the tool monitors deployments continuously across multiple rollouts. Exit with Ctrl+C when done.

```bash
# Continuous monitoring - watches across multiple rollouts
kubectl watch-rollout my-deployment -n production

# With resource type prefix (kubectl-style)
kubectl watch-rollout deployment/my-deployment -n production
kubectl watch-rollout deployments.apps/my-deployment -n production
```

**Perfect for:**
- Incident response (rollback → fix → verify cycles)
- Development iteration (multiple deploys in a session)
- Progressive rollouts (canary stages)

### Single-rollout mode (for automation)

Use `--until-complete` flag to exit after monitoring one rollout to its final state (success or failure). Exits with code 0 on success, code 1 on failure.

```bash
# Exit after current rollout reaches final state (success or failure)
kubectl watch-rollout my-deployment -n production --until-complete
```

**Perfect for:**
- CI/CD pipelines
- Automation scripts
- One-time rollout verification

### Line mode (for CI/CD and log aggregation)

Use `--line-mode` flag for timestamped, single-line output suitable for log aggregation systems and CI/CD pipelines.

```bash
# Line mode with timestamped output
kubectl watch-rollout my-deployment -n production --line-mode --until-complete

# Example output:
# 2025-10-12T14:20:00.123Z replicaset:api-7d8f9c status:progressing new:0/10 old:10/10 eta:-
# 2025-10-12T14:20:05.456Z replicaset:api-7d8f9c status:progressing new:2/10 old:10/10 eta:4m25s
#     WARNING: BackOff: Back-off restarting failed container (2x)
# 2025-10-12T14:24:00.789Z replicaset:api-7d8f9c status:complete new:10/10 old:0/0 duration:4m0s
```

**Perfect for:**
- CI/CD pipelines with log aggregation (CloudWatch, Splunk, Elasticsearch)
- Automated deployment monitoring
- Grep/awk/jq parsing
- Audit trails and post-mortems

### Using kubeconfig context

```bash
kubectl watch-rollout my-app --context=prod-cluster -n production
```

## Configuration

The tool uses your current kubeconfig configuration. You can specify a different context, namespace, or kubeconfig file using standard kubectl flags:

```bash
kubectl watch-rollout my-deployment --kubeconfig=/path/to/config --context=my-context -n my-namespace
```

## Exit Codes

- **0**: Success (rollout completed successfully, or user pressed Ctrl+C)
- **1**: Failure (rollout failed, deployment deleted, or API error)

In continuous mode (default), the tool only exits on terminal errors or Ctrl+C. In single-rollout mode (`--until-complete`), it exits after the rollout reaches a final state (either success or failure).
