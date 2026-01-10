# kubectl-watch-rollout

Watch Kubernetes deployment rollouts with live progress updates and status tracking

[![CI](https://github.com/ivoronin/kubectl-watch-rollout/actions/workflows/test.yml/badge.svg)](https://github.com/ivoronin/kubectl-watch-rollout/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout/releases)

[Overview](#overview) · [Features](#features) · [Installation](#installation) · [Usage](#usage) · [Configuration](#configuration) · [Requirements](#requirements) · [License](#license)

```bash
kubectl watch-rollout my-deployment -n production
```

![Screenshot](https://raw.githubusercontent.com/ivoronin/kubectl-watch-rollout/master/screenshot.png)

## Overview

This kubectl plugin monitors deployment rollouts by polling the Kubernetes API every 5 seconds to track ReplicaSet transitions. It calculates rollout progress by comparing Available, Ready, and Current pod counts between old and new ReplicaSets. Warning events from pods are clustered using Jaro-Winkler similarity to reduce noise while preserving distinct issues.

## Features

- Live progress bars showing pod lifecycle stages (Current, Ready, Available) for new and old ReplicaSets
- Estimated time to completion based on rollout velocity
- Warning event aggregation with deduplication using configurable similarity threshold
- Progress deadline detection with automatic failure recognition
- Continuous monitoring mode for incident response and development iteration
- Single-rollout mode (`--until-complete`) for CI/CD automation with exit code 0/1
- Line mode (`--line-mode`) for timestamped, parseable output suitable for log aggregation

## Installation

### Krew

```bash
kubectl krew install watch-rollout
```

### GitHub Releases

Download from [Releases](https://github.com/ivoronin/kubectl-watch-rollout/releases).

### Build from Source

```bash
git clone https://github.com/ivoronin/kubectl-watch-rollout.git
cd kubectl-watch-rollout
make build
```

## Usage

### Continuous Monitoring

Monitor a deployment across multiple rollouts. Exit with Ctrl+C when done.

```bash
kubectl watch-rollout my-deployment -n production

# With resource type prefix
kubectl watch-rollout deployment/my-deployment -n production
kubectl watch-rollout deployments.apps/my-deployment -n production
```

### Single-Rollout Mode

Exit after one rollout completes. Returns exit code 0 on success, 1 on failure.

```bash
kubectl watch-rollout my-deployment -n production --until-complete
```

### Line Mode

Timestamped, single-line output for log aggregation systems.

```bash
kubectl watch-rollout my-deployment -n production --line-mode --until-complete

# Example output:
# 2025-10-12T14:20:00.123Z replicaset:api-7d8f9c status:progressing new:0/10 old:10/10 eta:-
# 2025-10-12T14:20:05.456Z replicaset:api-7d8f9c status:progressing new:2/10 old:10/10 eta:4m25s
#     WARNING: BackOff: Back-off restarting failed container (2x)
# 2025-10-12T14:24:00.789Z replicaset:api-7d8f9c status:complete new:10/10 old:0/0 duration:4m0s
```

### Event Filtering

Ignore events matching a regular expression (matched against "Reason: Message").

```bash
kubectl watch-rollout my-deployment --ignore-events "Pulling|Pulled"
```

### Event Clustering

Adjust similarity threshold for event deduplication (0.0-1.0, lower = more aggressive clustering).

```bash
kubectl watch-rollout my-deployment --similarity-threshold 0.7
```

### Kubeconfig Options

```bash
kubectl watch-rollout my-deployment --kubeconfig=/path/to/config --context=my-context -n my-namespace
```

## Configuration

### Command-Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--until-complete` | Exit after one rollout completes | `false` |
| `--line-mode` | Use line-based output format | `false` |
| `--ignore-events` | Regex to filter events by "Reason: Message" | none |
| `--similarity-threshold` | Event clustering threshold (0.0-1.0) | `0.85` |
| `-n`, `--namespace` | Target namespace | current context |
| `--context` | Kubeconfig context | current context |
| `--kubeconfig` | Path to kubeconfig file | `~/.kube/config` |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Rollout completed successfully, or user pressed Ctrl+C |
| `1` | Rollout failed (progress deadline exceeded), deployment deleted, or API error |

## Requirements

### RBAC Permissions

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: watch-rollout
rules:
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods", "events"]
  verbs: ["get", "list", "watch"]
```

### Runtime

- Go 1.25+ (for building from source)
- kubectl configured with cluster access

## License

[GPL-3.0](LICENSE)
