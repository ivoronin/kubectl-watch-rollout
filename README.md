# kubectl-watch-rollout

Watch Kubernetes deployment rollouts with live progress updates and status tracking

[![CI](https://github.com/ivoronin/kubectl-watch-rollout/actions/workflows/test.yml/badge.svg)](https://github.com/ivoronin/kubectl-watch-rollout/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/ivoronin/kubectl-watch-rollout)](https://github.com/ivoronin/kubectl-watch-rollout/releases)

[Overview](#overview) · [Features](#features) · [Installation](#installation) · [Usage](#usage) · [Configuration](#configuration) · [Requirements](#requirements) · [License](#license)

```bash
kubectl watch-rollout nginx-server
```

![Screenshot](https://raw.githubusercontent.com/ivoronin/kubectl-watch-rollout/master/screenshot.png)

**CI/CD mode** — line output for Jenkins, GitHub Actions, GitLab CI, and other automation:

```bash
kubectl watch-rollout nginx-server --line-mode --until-complete
```

```
09:40:21 ▶ [REPLICASET nginx-server-646b99584c] [ROLLOUT PROGRESSING] [NEW 5/100] [OLD 95/100] [ETA 5m15s]
         └─ ℹ Created: Created container: nginx (10 exemplars, last 3s ago)
         └─ ℹ Pulled: Container image "123456789012.dkr.ecr... (10 exemplars, last 3s ago)
         └─ ℹ Scheduled: Successfully assigned <*> to <*> (10 exemplars, last 3s ago)
         └─ ℹ Started: Started container nginx (10 exemplars, last 3s ago)

09:40:36 ▶ [REPLICASET nginx-server-646b99584c] [ROLLOUT PROGRESSING] [NEW 10/100] [OLD 90/100] [ETA 4m44s]
         └─ ℹ Created: Created container: nginx (15 exemplars, last 5s ago)
         └─ ℹ Pulled: Container image "123456789012.dkr.ecr... (15 exemplars, last 5s ago)
         └─ ℹ Scheduled: Successfully assigned <*> to <*> (15 exemplars, last 5s ago)
         └─ ℹ Started: Started container nginx (15 exemplars, last 5s ago)

...
```

## Overview

This kubectl plugin monitors deployment rollouts by polling the Kubernetes API every 5 seconds to track ReplicaSet transitions. It calculates rollout progress by comparing Available, Ready, and Current pod counts between old and new ReplicaSets. Warning events from pods are clustered to reduce noise while preserving distinct issues.

## Features

- Live progress bars showing pod lifecycle stages (Current, Ready, Available) for new and old ReplicaSets
- Pod grid visualization showing individual pod states at a glance
- Estimated time to completion based on rollout velocity
- Warning event aggregation with deduplication using configurable similarity threshold
- Progress deadline detection with automatic failure recognition
- Continuous monitoring mode for incident response and development iteration
- Single-rollout mode (`--until-complete`) for CI/CD automation with exit code 0/1
- Line mode (`--line-mode`) for timestamped output in CI/CD pipelines

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
kubectl watch-rollout my-deployment
```

### Single-Rollout Mode

Exit after one rollout completes. Returns exit code 0 on success, 1 on failure.

```bash
kubectl watch-rollout my-deployment --until-complete
```

### Line Mode

Line output for CI/CD pipelines (see example above).

```bash
kubectl watch-rollout my-deployment --line-mode --until-complete
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
| `--similarity-threshold` | Event clustering threshold (0.0-1.0) | `0.5` |
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
