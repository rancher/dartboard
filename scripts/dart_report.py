#!/usr/bin/env python3
"""
Generate a 'dart report' markdown that combines dart configuration with the
metrics collected by `dartboard collect-metrics`. Calls plot_metrics.py to
produce per-query charts and stats, then prepends a dart-config summary so
the report is self-describing for cross-version / cross-scenario comparison.

    pip install pandas matplotlib pyyaml
    scripts/dart_report.py <metrics_dir> [--out <path>]
    scripts/dart_report.py <dart.yaml> <metrics_dir> [--out <path>]   # explicit dart

When dart is omitted, reads <metrics_dir>/dart.yaml (the snapshot written by
`dartboard collect-metrics`).

Default output: <metrics_dir>/dart_report.md
"""

import argparse
import json
import subprocess
import sys
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).resolve().parent
PLOT_SCRIPT = SCRIPT_DIR / "plot_metrics.py"


def fmt_kv(rows):
    return "\n".join(f"- **{k}:** {v}" for k, v in rows if v not in (None, "", [], {}))


def chart_section(chart_vars):
    if not chart_vars:
        return "_(no chart_variables in dart)_"
    rows = [
        ("rancher_version", chart_vars.get("rancher_version")),
        ("rancher_replicas", chart_vars.get("rancher_replicas")),
        ("rancher_image_override", chart_vars.get("rancher_image_override")),
        ("rancher_image_tag_override", chart_vars.get("rancher_image_tag_override")),
        ("rancher_chart_repo_override", chart_vars.get("rancher_chart_repo_override")),
        ("rancher_monitoring_version", chart_vars.get("rancher_monitoring_version")),
        ("cert_manager_version", chart_vars.get("cert_manager_version")),
        ("downstream_rancher_monitoring", chart_vars.get("downstream_rancher_monitoring")),
        ("force_prime_registry", chart_vars.get("force_prime_registry")),
    ]
    extras = chart_vars.get("extra_environment_variables") or []
    out = fmt_kv(rows)
    if extras:
        out += "\n- **extra_environment_variables:**\n"
        for e in extras:
            out += f"  - `{e.get('name', '?')}={e.get('value', '')}`\n"
    return out


def test_section(test_vars):
    if not test_vars:
        return "_(no test_variables in dart)_"
    return fmt_kv([
        ("test_config_maps", test_vars.get("test_config_maps")),
        ("test_secrets", test_vars.get("test_secrets")),
        ("test_roles", test_vars.get("test_roles")),
        ("test_users", test_vars.get("test_users")),
        ("test_projects", test_vars.get("test_projects")),
    ])


def downstream_section(templates):
    if not templates:
        return "_(no downstream_cluster_templates in dart)_"
    out = [
        "| # | clusters | servers | agents | distro | cpu | mem (GiB) | disk(s) | custom |",
        "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
    ]
    total_clusters = 0
    for i, t in enumerate(templates):
        count = t.get("cluster_count", 0)
        total_clusters += count
        nm = t.get("node_module_variables") or {}
        disks = nm.get("disks") or []
        disk_str = ", ".join(f"{d.get('name', '?')}:{d.get('size', '?')}GiB" for d in disks) or "—"
        pools = t.get("machine_pools") or []
        pool_qty = sum((p.get("machine_pool_config") or {}).get("quantity", 0) for p in pools)
        servers = t.get("server_count", 0)
        agents = t.get("agent_count", 0)
        if t.get("is_custom_cluster") and pool_qty:
            servers = f"{servers} (pool×{pool_qty})"
        out.append(
            f"| {i} | {count} | {servers} | {agents} | "
            f"{t.get('distro_version', '?')} | {nm.get('cpu', '?')} | "
            f"{nm.get('memory', '?')} | {disk_str} | "
            f"{'yes' if t.get('is_custom_cluster') else 'no'} |"
        )
    out.append(f"\n**Total downstream clusters:** {total_clusters}")
    return "\n".join(out)


def upstream_section(upstream, tofu_vars):
    if not upstream and not tofu_vars:
        return "_(no upstream_cluster info)_"
    distro = (tofu_vars or {}).get("upstream_cluster_distro_module")
    rows = [
        ("kubeconfig", upstream.get("kubeconfig") if upstream else None),
        ("public_url", (upstream.get("kubernetes_addresses") or {}).get("public") if upstream else None),
        ("distro_module", distro),
        ("reserve_node_for_monitoring", upstream.get("reserve_node_for_monitoring") if upstream else None),
    ]
    return fmt_kv(rows)


def manifest_section(metrics_dir: Path):
    mf = metrics_dir / "manifest.json"
    if not mf.exists():
        return "_(no manifest.json — was `dartboard collect-metrics` run against this dir?)_"
    m = json.loads(mf.read_text())
    return fmt_kv([
        ("window", f"{m.get('start')} → {m.get('end')}"),
        ("step", f"{m.get('step_seconds', 0) / 1e9:g}s"),
        ("workspace", m.get("tofu_workspace") or "(default)"),
        ("dart_file", m.get("dart_file")),
        ("prom_url", m.get("prom_url")),
        ("queries_collected", m.get("queries")),
    ])


def run_plot_script(metrics_dir: Path) -> Path:
    cmd = [sys.executable, str(PLOT_SCRIPT), str(metrics_dir)]
    print(f"running: {' '.join(cmd)}")
    subprocess.run(cmd, check=True)
    return metrics_dir / "report.md"


def metrics_section(metrics_report: Path) -> str:
    if not metrics_report.exists():
        return "_(plot_metrics.py did not produce report.md)_"
    text = metrics_report.read_text()
    lines = text.splitlines()
    # Drop the leading "# Metrics report" + window/step lines emitted by plot_metrics.py;
    # we already surfaced that context above.
    keep_from = 0
    for i, line in enumerate(lines):
        if line.startswith("## "):
            keep_from = i
            break
    return "\n".join(lines[keep_from:])


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("dart", type=Path, nargs="?", default=None, help="dart YAML file (default: <metrics_dir>/dart.yaml)")
    parser.add_argument("metrics_dir", type=Path, help="directory produced by `dartboard collect-metrics`")
    parser.add_argument("--out", type=Path, default=None, help="output path (default: <metrics_dir>/dart_report.md)")
    parser.add_argument("--skip-plot", action="store_true", help="skip running plot_metrics.py (assume report.md + charts already exist)")
    args = parser.parse_args()

    metrics_dir = args.metrics_dir.resolve()
    dart_path = (args.dart or (metrics_dir / "dart.yaml")).resolve()
    if not dart_path.exists():
        parser.error(
            f"dart not found at {dart_path}. Pass an explicit dart path, or "
            f"re-run `dartboard collect-metrics` (it now snapshots the dart "
            f"into the metrics dir as dart.yaml)."
        )
    out_path = (args.out or (metrics_dir / "dart_report.md")).resolve()

    dart = yaml.safe_load(dart_path.read_text()) or {}
    chart_vars = dart.get("chart_variables") or {}
    test_vars = dart.get("test_variables") or {}
    tofu_vars = dart.get("tofu_variables") or {}
    upstream = dart.get("upstream_cluster") or {}
    templates = tofu_vars.get("downstream_cluster_templates") or []

    if not args.skip_plot:
        run_plot_script(metrics_dir)

    md = [
        f"# Dart report: `{dart_path.name}`",
        "",
        "## Dart configuration",
        "",
        f"- **dart_file:** `{dart_path}`",
        f"- **tofu_main_directory:** `{dart.get('tofu_main_directory', '?')}`",
        f"- **tofu_workspace:** `{dart.get('tofu_workspace') or '(default)'}`",
        f"- **cluster_batch_size:** {dart.get('cluster_batch_size', '?')}",
        "",
        "### Rancher / charts",
        "",
        chart_section(chart_vars),
        "",
        "### Upstream cluster",
        "",
        upstream_section(upstream, tofu_vars),
        "",
        "### Downstream clusters",
        "",
        downstream_section(templates),
        "",
        "### Test workload",
        "",
        test_section(test_vars),
        "",
        "## Metrics collection window",
        "",
        manifest_section(metrics_dir),
        "",
        "## Metrics",
        "",
        metrics_section(metrics_dir / "report.md"),
        "",
    ]

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(md))
    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
