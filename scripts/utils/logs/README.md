# Dartboard Log Collection and Analysis Scripts

This directory contains scripts for collecting and analyzing logs from Kubernetes clusters deployed by dartboard.

## Overview

These scripts automate the process of collecting diagnostic logs from all nodes in a dartboard deployment and analyzing them for common issues in Rancher components.

### Scripts

- **[collect_logs.sh](#collect_logssh)** - Collects logs from all cluster nodes using `rancher2_logs_collector.sh`
- **[analyze_logs.sh](#analyze_logssh)** - Analyzes collected logs for errors in key Rancher components

## Prerequisites

- A dartboard deployment with SSH access configured
- SSH keys and bastion host access (automatically handled via dartboard's SSH scripts)
- Internet access on cluster nodes to download the logs collector
- `curl`, `tar`, `gzip`, and standard Unix utilities

## collect_logs.sh

Runs the [Rancher logs collector](https://github.com/rancherlabs/support-tools/tree/master/collection/rancher/v2.x/logs-collector) on each node in your dartboard deployment and downloads the resulting tarballs.

### Features

- **Auto-detection**: Automatically finds the dartboard config directory
- **Parallel execution**: Collects logs from multiple nodes simultaneously (default: 5 concurrent)
- **SSH tunneling**: Works with bastion hosts and SSH proxy configurations
- **Progress tracking**: Shows real-time status for each node
- **Error handling**: Continues on failures and reports which nodes failed
- **Automatic cleanup**: Removes remote files after successful download

### Usage

```bash
# Basic usage - auto-detect config and collect all logs
./scripts/collect_logs.sh

# Specify config directory explicitly
./scripts/collect_logs.sh -c ./tofu/main/aws/my-workspace_config

# Collect logs from last 3 days only
./scripts/collect_logs.sh -f "-s 3"

# Enable obfuscation of IP addresses and hostnames
./scripts/collect_logs.sh -f "-o"

# Collect with obfuscation and last 5 days
./scripts/collect_logs.sh -f "-s 5 -o"

# Change output directory and parallel count
./scripts/collect_logs.sh -o ./my-logs -p 10

# Dry run to see what would happen
./scripts/collect_logs.sh -n

# Verbose mode for debugging
./scripts/collect_logs.sh -v
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `-c, --config-dir DIR` | Path to dartboard config directory | Auto-detect |
| `-o, --output-dir DIR` | Directory to store collected tarballs | `./collected_logs` |
| `-p, --parallel N` | Number of nodes to process in parallel | `5` |
| `-f, --collector-flags FLAGS` | Additional flags to pass to rancher2_logs_collector.sh | None |
| `-n, --dry-run` | Show what would be done without executing | `false` |
| `-v, --verbose` | Enable verbose output | `false` |
| `-h, --help` | Show help message | - |

### Collector Flags

The `-f` option accepts any flags supported by `rancher2_logs_collector.sh`:

- `-s N` - Start day of log collection (N days ago)
- `-e N` - End day of log collection (N days ago)
- `-o` - Obfuscate IP addresses and hostnames
- `-p` - Use default nice/ionice priorities instead of lowest
- `-f` - Force collection even if minimum space isn't available
- `-r DISTRO` - Override k8s distribution detection (rke|k3s|rke2|kubeadm)

### How It Works

1. **Discovery**: Finds the dartboard config directory (or uses specified path)
2. **Node detection**: Locates all `ssh-to-*.sh` scripts in the config directory
3. **Parallel collection**:
   - Downloads the logs collector script to each node
   - Runs collection with `sudo` in `/var/tmp` (avoids tmpfs issues)
   - Extracts the tarball path from output
4. **Download**: Transfers tarballs via SSH pipe (`sudo cat file`)
5. **Verification**: Validates downloaded files are valid gzip archives
6. **Cleanup**: Removes remote collector script and tarball

### Output

Downloaded tarballs are named: `<node-name>-<hostname>-<timestamp>.tar.gz`

Example:
```
collected_logs/
├── downstream-custom-0-0-ip-172-31-32-124-2026-02-13_23_22_08.tar.gz
├── downstream-custom-0-1-ip-172-31-32-9-2026-02-13_23_22_08.tar.gz
├── upstream-server-0-ip-172-31-32-237-2026-02-13_22_55_54.tar.gz
└── upstream-server-1-ip-172-31-32-92-2026-02-13_22_55_55.tar.gz
```

### What Gets Collected

The Rancher logs collector gathers:

- **Node-level**: OS configuration, network settings, iptables, journald logs
- **Kubernetes**: Pod logs from system namespaces, kubectl output, events
- **Distribution**: RKE2/K3s configuration files, static pod manifests
- **Control plane**: kube-apiserver, etcd, kubelet logs and configuration
- **Container runtime**: containerd/docker logs and configuration

See the [collection details](https://github.com/rancherlabs/support-tools/blob/master/collection/rancher/v2.x/logs-collector/collection-details.md) for complete information.

### Troubleshooting

**Problem**: Nodes fail with "No tarball found"
- **Solution**: Check if nodes have enough disk space in `/var/tmp`

**Problem**: SSH connection failures
- **Solution**: Verify SSH keys are accessible and bastion host is reachable

**Problem**: Slow collection
- **Solution**: Reduce parallel count with `-p 3` or adjust collector flags

## analyze_logs.sh

Analyzes collected log tarballs for errors and issues in key Rancher components.

### Features

- **Component-focused**: Searches for issues in cattle-agent, rancher, CAPI, fleet, webhooks
- **Pattern matching**: Detects errors, failures, timeouts, crashes, and evictions
- **Summary report**: Shows issue counts by component across all nodes
- **Detailed findings**: Provides file paths and actual error messages
- **Kubernetes issues**: Identifies CrashLoopBackOff, OOMKilled, ImagePullBackOff, etc.
- **etcd monitoring**: Detects etcd-specific issues and raft problems
- **Most common errors**: Lists the top 20 most frequent error patterns

### Usage

```bash
# Basic usage - analyze logs in default directory
./scripts/analyze_logs.sh

# Analyze specific directory
./scripts/analyze_logs.sh -d ./collected_logs

# Save report to file
./scripts/analyze_logs.sh -o analysis-report.txt

# Include warnings in addition to errors
./scripts/analyze_logs.sh -w

# More context lines around errors
./scripts/analyze_logs.sh -c 5

# Show more errors per component
./scripts/analyze_logs.sh -m 100

# Verbose mode to see analysis progress
./scripts/analyze_logs.sh -v

# Combined options
./scripts/analyze_logs.sh -d ./my-logs -w -o report.txt -c 5 -v
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `-d, --logs-dir DIR` | Directory containing collected log tarballs | `./collected_logs` |
| `-o, --output FILE` | Write report to file instead of stdout | stdout |
| `-w, --warnings` | Include warnings in addition to errors | `false` |
| `-c, --context N` | Number of context lines around errors | `3` |
| `-m, --max-errors N` | Maximum errors to show per component | `50` |
| `-v, --verbose` | Show verbose output during analysis | `false` |
| `-h, --help` | Show help message | - |

### Analyzed Components

The script searches for issues in:

- **cattle-agent / cattle-cluster-agent**: Cluster registration and management
- **rancher**: Rancher server pods
- **system-upgrade-controller**: Automated upgrades
- **CAPI / cluster-api**: Cluster API provisioning
- **fleet**: GitOps and fleet management
- **webhook**: Admission webhooks

Additionally searches for:
- **Kubernetes issues**: Pod crashes, scheduling failures, image pull errors
- **etcd issues**: Raft consensus, compaction, snapshot failures

### Report Structure

The analysis report contains two main sections:

#### 1. Summary Section

```
================================================================================
                        LOG ANALYSIS SUMMARY
================================================================================

Analysis Date: Fri Feb 13 07:00:26 PM EST 2026
Logs Directory: ./collected_logs
Include Warnings: false

ISSUE COUNTS BY COMPONENT:
--------------------------
  cattle-cluster-agent:          10 issues across 10 nodes
  rancher:                       1032 issues across 16 nodes
  system-upgrade-controller:     30 issues across 13 nodes
  capi:                          50 issues across 3 nodes
  fleet:                         802 issues across 14 nodes
  webhook:                       215 issues across 14 nodes
  etcd Issues:                   found on 4 nodes

MOST COMMON ERROR PATTERNS:
---------------------------
      9 {"level":"error","ts":"2026-02-13T18:42:34Z",...
      6 W0213 21:19:04.234137 Failed calling webhook...
      ...
```

#### 2. Detailed Findings Section

Organized by node, then by component:

```
================================================================================
NODE: upstream-server-0
================================================================================

--------------------------------------------------------------------------------
COMPONENT: rancher (15 issues found)
--------------------------------------------------------------------------------

--- File: podlogs/cattle-system-rancher-7d5f4c8d9-abc123 ---
2026-02-13T18:45:23Z level=error msg="Failed to sync cluster" ...

--- File: journald/rancher-system-agent ---
Feb 13 18:46:42 error syncing 'fleet-default/custom-...'
...
```

### Search Patterns

The analyzer searches for these error patterns:

**Errors**:
- error, Error, ERROR
- fatal, Fatal, FATAL
- panic, Panic, PANIC
- failed, Failed, FAILED
- exception, Exception
- timeout, Timeout
- refused, Refused
- unavailable, Unavailable

**Kubernetes-specific**:
- OOMKilled, CrashLoopBackOff
- ImagePullBackOff, ErrImagePull
- evicted, Evicted
- BackOff

**Warnings (with `-w` flag)**:
- warn, Warning, WARNING
- deprecated, Deprecated

### Performance

- Processes 16 nodes (16 tarballs) in ~30 seconds
- Extracts each tarball temporarily and removes after analysis
- Memory-efficient: only one node extracted at a time

### Example Workflow

Complete workflow from deployment to analysis:

```bash
# 1. Deploy with dartboard
dartboard deploy -f darts/my-config.yaml

# 2. Wait for deployment to complete and run tests
# ...

# 3. Collect logs from all nodes
./scripts/collect_logs.sh -f "-s 7"  # Last 7 days

# 4. Analyze collected logs
./scripts/analyze_logs.sh -w -o analysis-report-$(date +%Y%m%d).txt

# 5. Review the report
less analysis-report-20260213.txt

# 6. Search for specific component issues
grep -A 10 "COMPONENT: fleet" analysis-report-20260213.txt
```

## Integration with CI/CD

### Jenkins Pipeline Example

```groovy
stage('Collect Logs on Failure') {
    when { expression { currentBuild.result == 'FAILURE' } }
    steps {
        sh './scripts/collect_logs.sh -o logs-${BUILD_NUMBER}'
        sh './scripts/analyze_logs.sh -d logs-${BUILD_NUMBER} -o analysis-${BUILD_NUMBER}.txt'
        archiveArtifacts artifacts: 'logs-${BUILD_NUMBER}/*.tar.gz, analysis-${BUILD_NUMBER}.txt'
    }
}
```

### GitHub Actions Example

```yaml
- name: Collect and Analyze Logs
  if: failure()
  run: |
    ./scripts/collect_logs.sh -o collected_logs
    ./scripts/analyze_logs.sh -o analysis-report.txt

- name: Upload Logs
  if: failure()
  uses: actions/upload-artifact@v3
  with:
    name: cluster-logs
    path: |
      collected_logs/*.tar.gz
      analysis-report.txt
```

## Tips and Best Practices

### Log Collection

1. **Collect early**: Run collection as soon as issues are noticed, before logs rotate
2. **Adjust time range**: Use `-f "-s N"` to collect appropriate history
3. **Parallel tuning**: Adjust `-p` based on network capacity and node count
4. **Obfuscation**: Use `-f "-o"` when sharing logs externally
5. **Storage**: Allocate ~100-500MB per node for collected tarballs

### Analysis

1. **Start with summary**: Review issue counts to identify problem areas
2. **Filter by component**: Use `grep` to focus on specific components
3. **Track over time**: Save dated reports to track issue trends
4. **Combine with warnings**: Use `-w` for comprehensive analysis during troubleshooting
5. **Cross-reference**: Compare errors across multiple nodes to identify cluster-wide issues

### Debugging Collection Issues

Enable verbose mode to see detailed execution:

```bash
./scripts/collect_logs.sh -v -n  # Dry run with verbose
./scripts/collect_logs.sh -v     # Real run with verbose
```

Check SSH connectivity manually:

```bash
# Test SSH to a specific node
./tofu/main/aws/my-workspace_config/ssh-to-upstream-server-0.sh hostname

# Test sudo access
./tofu/main/aws/my-workspace_config/ssh-to-upstream-server-0.sh sudo whoami
```

## FAQ

**Q: How long does log collection take?**
A: Typically 1-3 minutes per node. With 5 parallel workers, 16 nodes complete in ~5-10 minutes.

**Q: How much disk space do I need?**
A: Plan for ~200-500MB per node collected, plus analysis requires temporary space (deleted after processing).

**Q: Can I collect from specific nodes only?**
A: Yes, temporarily move unwanted `ssh-to-*.sh` scripts out of the config directory, or collect all and analyze specific tarballs.

**Q: What if a node is unreachable?**
A: The script continues with other nodes and reports failures at the end.

**Q: How do I share logs with Rancher support?**
A: Use obfuscation: `./scripts/collect_logs.sh -f "-o"`, then share the tarballs and analysis report.

**Q: Can I run collection multiple times?**
A: Yes, each run creates uniquely timestamped tarballs.

**Q: Do the scripts work with non-dartboard clusters?**
A: `collect_logs.sh` is dartboard-specific, but `analyze_logs.sh` works with any Rancher logs collector tarballs.

## Contributing

When modifying these scripts:

1. Test with dry-run mode first
2. Verify error handling with intentional failures
3. Update this README with new features or options
4. Test with different dartboard configurations (AWS, Azure, k3d, Harvester)

## References

- [Rancher Logs Collector](https://github.com/rancherlabs/support-tools/tree/master/collection/rancher/v2.x/logs-collector)
- [Rancher Support Matrix](https://rancher.com/support-maintenance-terms)
- [Dartboard Documentation](../../../README.md)
