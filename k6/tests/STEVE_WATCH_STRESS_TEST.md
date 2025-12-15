# Steve Watch Stress Test with SQLite Caching

This test stresses Steve's watch functionality with SQLite caching enabled, inspired by the stress test scripts from [this gist](https://gist.github.com/aruiz14/cf279761268a1458cb3838e6f41388ac).

## Overview

The test simulates a high-stress scenario for Steve by:

1. **Creating 2000 concurrent WebSocket watchers** (configurable) that subscribe to Steve for resource changes
2. **Continuously creating and deleting resources** (ConfigMaps, Secrets, and Custom Resources) via the Kubernetes API
3. **Periodically updating CRD definitions** to trigger schema change events
4. **Running light read tests** (1 per second) to verify Steve remains responsive

## Success Criteria

The test is considered successful if:

- **Steve responsiveness**: p95 response time for light reads stays below 100ms
- **SQLite WAL file size**: Remains below 10 MB after 10 minutes of stress testing (checked automatically via Kubernetes API)
- **Overall success rate**: At least 95% of operations succeed

## Prerequisites

- Access to a Kubernetes cluster with Rancher installed
- Rancher must be running with SQLite caching enabled (`-sql-cache` flag)
- Valid kubeconfig with appropriate permissions
- Rancher credentials (username/password or token)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `STEVE_URL` | Steve server URL (Rancher URL) | `http://localhost:8080` |
| `KUBE_API_URL` | Kubernetes API URL | From `KUBECONFIG` |
| `KUBECONFIG` | Path to kubeconfig file | Required |
| `CONTEXT` | Kubeconfig context to use | Required |
| `USERNAME` | Rancher username | Required (or use TOKEN) |
| `PASSWORD` | Rancher password | Required (or use TOKEN) |
| `TOKEN` | Rancher session token | Alternative to USERNAME/PASSWORD |
| `NAMESPACE` | Test namespace | `test-configmaps` |
| `COUNT` | Number of concurrent watchers | `2000` |
| `WATCH_DURATION` | Test duration in seconds | `600` (10 minutes) |
| `RANCHER_NAMESPACE` | Rancher pod namespace | `cattle-system` |
| `RANCHER_POD_LABEL` | Label to find Rancher pod | `app=rancher` |

## Running the Test

```bash
k6 run \
  --env STEVE_URL=https://rancher.example.com \
  --env KUBECONFIG=/path/to/kubeconfig \
  --env CONTEXT=my-context \
  --env USERNAME=admin \
  --env PASSWORD=secret \
  --env COUNT=2000 \
  --env WATCH_DURATION=600 \
  k6/tests/steve_watch_stress_test.js
```

The test automatically monitors SQLite WAL file size every 10 seconds via the Kubernetes API and reports it as the `sqlite_wal_size_bytes` metric.

## Test Scenarios

The test runs the following scenarios in parallel:

### 1. Watchers Scenario
- **Executor**: `per-vu-iterations`
- **VUs**: Equal to `COUNT` parameter (default: 2000)
- **Duration**: Full test duration
- **Behavior**: Each VU creates a WebSocket connection to Steve and subscribes to:
  - ConfigMaps (with resource.changes mode and 4s debounce)
  - Secrets (with resource.changes mode and 4s debounce)
  - Custom CRD resources (example.com/foos)
- **Connection Lifetime**: Connections remain open for the full test duration plus a small random jitter (up to 10% of duration) to avoid all connections closing simultaneously

### 2. Create/Delete Events Scenario
- **Executor**: `constant-arrival-rate`
- **Rate**: 10 iterations per second
- **Duration**: Full test duration
- **Behavior**: Each iteration:
  1. Creates a ConfigMap with 1MB of data
  2. Creates a Secret with 1MB of data
  3. Creates a Custom Resource instance
  4. Waits 100ms
  5. Deletes all three resources

### 3. Update CRDs Scenario
- **Executor**: `constant-arrival-rate`
- **Rate**: ~0.33 iterations per second (once every 3 seconds)
- **Duration**: Full test duration
- **Behavior**: Alternates between two CRD schema versions:
  - Version 1: Includes `additionalPrinterColumns`
  - Version 2: Minimal schema without extra columns
- Simulates schema changes that Steve must process

### 4. Light Read Test Scenario
- **Executor**: `constant-arrival-rate`
- **Rate**: 1 iteration per second
- **Duration**: Full test duration
- **Behavior**: Performs a light read of ConfigMaps in the test namespace via Steve API
- **Purpose**: Ensures Steve remains responsive under stress
- **Threshold**: p95 response time must be < 100ms

### 5. WAL Size Check Scenario
- **Executor**: `constant-arrival-rate`
- **Rate**: 0.1 iterations per second (every 10 seconds)
- **Duration**: Full test duration
- **Behavior**: Executes command in Rancher pod via Kubernetes API to check WAL file size
- **Purpose**: Monitors SQLite WAL file growth during stress test
- **Threshold**: WAL size must remain below 10 MB

## What the Test Validates

### Automatically Validated
- ✅ Steve API responsiveness under load
- ✅ Success rate of resource operations
- ✅ HTTP request failure rates
- ✅ WebSocket connection stability
- ✅ SQLite WAL file size (via Kubernetes exec API)

## Understanding the Results

### Key Metrics

- `steve_light_read_duration`: Response time for light reads (should stay < 100ms)
- `sqlite_wal_size_bytes`: SQLite WAL file size in bytes (should stay < 10 MB)
- `http_req_duration`: Overall HTTP request duration
- `http_req_failed`: Rate of failed HTTP requests (should be < 5%)
- `checks`: Overall success rate (should be > 95%)
- `watcher_errors`: Number of WebSocket watcher errors
- `event_create_errors`: Number of create/delete cycle errors
- `crd_update_errors`: Number of CRD update errors

### Interpreting SQLite WAL Size

The SQLite WAL (Write-Ahead Log) file grows as Steve processes changes. If it grows beyond 10 MB, it indicates:
- Steve may not be checkpointing the WAL efficiently
- The cache is under too much stress
- Potential performance degradation

## Differences from Original Gist

This k6 test is based on [the original gist](https://gist.github.com/aruiz14/cf279761268a1458cb3838e6f41388ac) but with these adaptations:

1. **WebSocket watchers in JavaScript**: The Go-based watcher logic is reimplemented using k6's WebSocket support
2. **Kubernetes API for resource operations**: All CRUD operations use the Kubernetes API directly (not Steve)
3. **Integrated WAL size check**: Implements Kubernetes exec API via WebSocket to check WAL size from within k6
4. **Parametrized and configurable**: All key parameters are exposed as environment variables
5. **Integrated metrics**: Uses k6's native metrics and thresholds for validation

## Troubleshooting

### WebSocket connection failures
- Verify `STEVE_URL` is accessible
- Check authentication credentials
- Ensure Steve is running with SQLite cache enabled

### Resource creation failures
- Verify kubeconfig has sufficient permissions
- Check if namespace already exists from previous run
- Ensure CRD can be created in the cluster

### High response times
- This is expected under stress - the test is designed to push limits
- Monitor SQLite WAL file size
- Check Rancher pod resource usage (CPU/memory)

## Cleanup

The test automatically cleans up resources in the teardown phase:
- Deletes the test namespace (which removes all ConfigMaps, Secrets, and Custom Resources)
- Deletes the Custom Resource Definition

If the test is interrupted, manually clean up:

```bash
kubectl delete namespace test-configmaps
kubectl delete crd foos.example.com
```
