#!/usr/bin/env python3
"""
Plot CSVs collected by `dartboard collect-metrics` and render a markdown
report suitable for committing alongside test results in docs/.

    pip install pandas matplotlib
    scripts/plot_metrics.py <metrics_dir> [--out <output_dir>]

By default writes <metrics_dir>/charts/<group>/*.png and <metrics_dir>/report.md.
"""

import argparse
import json
import math
from pathlib import Path

import matplotlib.dates as mdates
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import pandas as pd

# Display unit per query (matches internal/metrics/catalog.go).
UNITS = {
    "node_cpu_utilization":        "ratio",
    "rancher_pod_cpu":             "cores",
    "node_mem_used_bytes":         "bytes",
    "rancher_pod_workingset":      "bytes",
    "rancher_pod_rss":             "bytes",
    "node_disk_throughput":        "bytes/sec",
    "node_disk_iops":              "iops",
    "node_filesystem_utilization": "ratio",
    "node_net_throughput":         "bytes/sec",
    "pod_cpu_requests":            "cores",
    "pod_cpu_limits":              "cores",
    "pod_memory_requests":         "bytes",
    "pod_memory_limits":           "bytes",
    "k3k_server_cpu_per_active_pod_millicores": "millicores",
    "k3k_server_memory_per_active_pod_mb":       "MB",
    "k3k_kubelet_cpu_per_active_pod_millicores": "millicores",
    "k3k_kubelet_memory_per_active_pod_mb":       "MB",
    "k3k_agent_cpu_per_active_pod_millicores": "millicores",
    "k3k_agent_memory_per_active_pod_mb":       "MB",
    "k3k_controller_cpu_per_active_pod_millicores": "millicores",
    "k3k_controller_memory_per_active_pod_mb":       "MB",
}

# Which label columns identify a "series" for legend purposes. Anything not
# listed falls back to all non-(timestamp,value) columns.
SERIES_LABEL = {
    "node_cpu_utilization":        ["instance"],
    "rancher_pod_cpu":             ["namespace", "pod"],
    "node_mem_used_bytes":         ["instance"],
    "rancher_pod_workingset":      ["namespace", "pod"],
    "rancher_pod_rss":             ["namespace", "pod"],
    "node_disk_throughput":        ["instance", "device"],
    "node_disk_iops":              ["instance", "device"],
    "node_filesystem_utilization": ["instance", "mountpoint"],
    "node_net_throughput":         ["instance", "device"],
    "pod_cpu_requests":            ["namespace", "pod"],
    "pod_cpu_limits":              ["namespace", "pod"],
    "pod_memory_requests":         ["namespace", "pod"],
    "pod_memory_limits":           ["namespace", "pod"],
}

MAX_LEGEND_LINES = 12


def recover_collection_metadata(metrics_dir: Path):
    """Recover metadata for older/partial collections that have CSVs only."""
    summary_path = metrics_dir / "summary.json"
    manifest_path = metrics_dir / "manifest.json"
    if summary_path.exists() and manifest_path.exists():
        return json.loads(summary_path.read_text()), json.loads(manifest_path.read_text())

    summary = {}
    all_timestamps = []
    for csv_path in metrics_dir.glob("*/*.csv"):
        df = pd.read_csv(csv_path)
        if df.empty:
            continue
        df["timestamp"] = pd.to_datetime(df["timestamp"], utc=True)
        all_timestamps.extend(df["timestamp"].tolist())
        label_cols = sorted(c for c in df.columns if c not in ("timestamp", "value"))
        grouped = [("aggregate", df)]
        if label_cols:
            grouped = list(df.groupby(label_cols, dropna=False, sort=True))
        query_summary = {}
        for labels, rows in grouped:
            values = rows[["timestamp", "value"]].dropna().sort_values("timestamp")
            if values.empty:
                continue
            numeric = values["value"].astype(float).sort_values().tolist()
            first = float(values.iloc[0]["value"])
            last = float(values.iloc[-1]["value"])
            change = last - first
            x = pd.Series(
                ((values["timestamp"] - values.iloc[0]["timestamp"]).dt.total_seconds() / 3600).to_numpy(),
                dtype=float,
            )
            y = pd.Series(values["value"].to_numpy(), dtype=float)
            variance = x.var()
            slope = None if pd.isna(variance) or variance <= 0 else float(y.cov(x) / variance)
            percentile = lambda p: numeric[max(0, math.ceil(p * len(numeric)) - 1)]
            if label_cols:
                label_values = labels if isinstance(labels, tuple) else (labels,)
                key = ",".join(f"{column}={value}" for column, value in zip(label_cols, label_values))
            else:
                key = "aggregate"
            query_summary[key] = {
                "count": len(numeric), "min": numeric[0], "p50": percentile(.5),
                "p95": percentile(.95), "p99": percentile(.99), "max": numeric[-1],
                "mean": sum(numeric) / len(numeric), "first": first, "last": last,
                "change": change, "slope_per_hour": slope,
            }
            if first != 0:
                query_summary[key]["change_percent"] = change / abs(first) * 100
        if query_summary:
            summary[csv_path.stem] = query_summary

    if not all_timestamps:
        raise ValueError(f"no metric samples found under {metrics_dir}")
    timestamps = sorted(set(all_timestamps))
    steps = [(b - a).total_seconds() for a, b in zip(timestamps, timestamps[1:]) if b > a]
    queries_path = metrics_dir / "queries.json"
    query_count = len(json.loads(queries_path.read_text())) if queries_path.exists() else len(summary)
    manifest = {
        "generated_at": pd.Timestamp.now(tz="UTC").isoformat(),
        "start": timestamps[0].isoformat().replace("+00:00", "Z"),
        "end": timestamps[-1].isoformat().replace("+00:00", "Z"),
        "step_seconds": int((min(steps) if steps else 0) * 1_000_000_000),
        "preset": "k3k", "queries": query_count, "recovered_from_csv": True,
    }
    summary_path.write_text(json.dumps(summary, indent=2) + "\n")
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n")
    return summary, manifest


def human_bytes(n):
    n = float(n)
    for unit in ("B", "KiB", "MiB", "GiB", "TiB"):
        if abs(n) < 1024:
            return f"{n:.1f}{unit}"
        n /= 1024
    return f"{n:.1f}PiB"


def human_number(n):
    """Format a non-byte numeric value without scientific notation."""
    n = float(n)
    if n == 0:
        return "0"
    if abs(n) >= 1000:
        return f"{n:,.0f}"
    return f"{n:.4g}"


def human_value(value, unit, series_label=""):
    """Format a metric value as a human-readable string for the given unit."""
    if value is None:
        return "—"
    v = float(value)
    if unit == "bytes":
        return human_bytes(v)
    if unit == "bytes/sec":
        return human_bytes(v) + "/s"
    if unit == "ratio":
        return f"{v * 100:.1f}%"
    if unit == "mixed":
        # pod_resource_{requests,limits} mixes memory (bytes) and cpu (cores)
        # rows in one query — pick the formatter from the series labels.
        if "unit=byte" in series_label or "resource=memory" in series_label:
            return human_bytes(v)
        return human_number(v)
    return human_number(v)


def y_formatter(unit):
    return lambda x, _: human_value(x, unit)


def query_unit(query_name):
    if query_name in UNITS:
        return UNITS[query_name]
    if query_name.endswith(("_pod_cpu", "_cpu_total", "_cpu_max", "_focus_server_cpu", "_focus_kubelet_cpu")):
        return "cores"
    if "workingset" in query_name or query_name.endswith("_used_bytes"):
        return "bytes"
    return ""


def plot_csv(csv_path: Path, out_path: Path, query_name: str) -> bool:
    df = pd.read_csv(csv_path)
    if df.empty:
        return False

    df["timestamp"] = pd.to_datetime(df["timestamp"], utc=True)
    series_cols = [c for c in SERIES_LABEL.get(query_name, []) if c in df.columns]
    if not series_cols:
        series_cols = [c for c in df.columns if c not in ("timestamp", "value")]

    if series_cols:
        df["__series__"] = df[series_cols].astype(str).agg("/".join, axis=1)
    else:
        df["__series__"] = "value"

    pivot = (
        df.pivot_table(index="timestamp", columns="__series__", values="value", aggfunc="last")
          .sort_index()
    )

    series_max = pivot.max().sort_values(ascending=False)
    top = list(series_max.index[:MAX_LEGEND_LINES])
    others = [s for s in pivot.columns if s not in top]

    fig, ax = plt.subplots(figsize=(10, 4.5))
    for s in others:
        ax.plot(pivot.index, pivot[s], color="lightgray", linewidth=0.6, alpha=0.6)
    for s in top:
        ax.plot(pivot.index, pivot[s], linewidth=1.2, label=s)
        values = pivot[s].dropna()
        if len(values) >= 2:
            # Convert the DatetimeIndex-derived values to Series explicitly.
            # Older pandas releases do not implement Index.mean(), and using
            # Series also guarantees aligned RangeIndexes for cov/var.
            x = pd.Series(
                ((values.index - values.index[0]).total_seconds() / 3600).to_numpy(),
                dtype=float,
            )
            y = pd.Series(values.to_numpy(), dtype=float)
            variance = x.var()
            if pd.notna(variance) and variance > 0:
                slope = y.cov(x) / variance
                intercept = y.mean() - slope * x.mean()
                ax.plot(
                    values.index,
                    intercept + slope * x.to_numpy(),
                    linestyle="--",
                    linewidth=1,
                    alpha=0.7,
                )

    unit = query_unit(query_name)
    ax.set_title(query_name)
    ax.set_xlabel("time (UTC)")
    ax.set_ylabel(unit)
    ax.yaxis.set_major_formatter(mticker.FuncFormatter(y_formatter(unit)))
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%H:%M"))
    ax.grid(True, alpha=0.3)
    fig.autofmt_xdate()

    if top:
        title = f"top {len(top)}/{len(pivot.columns)} series" if others else None
        ax.legend(
            fontsize="x-small",
            loc="upper left",
            bbox_to_anchor=(1.02, 1.0),
            borderaxespad=0,
            title=title,
        )

    out_path.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(out_path, dpi=130, bbox_inches="tight")
    plt.close(fig)
    return True


def stats_table(stats_for_query: dict, query_name: str, top_n: int = 10) -> str:
    if not stats_for_query:
        return "_(no series)_"
    rows = sorted(stats_for_query.items(), key=lambda kv: kv[1].get("p95", 0), reverse=True)[:top_n]
    unit = query_unit(query_name)
    out = [
        "| series | samples | first | last | change | change % | slope/hour | mean | p95 | max |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for series, s in rows:
        display_series = "aggregate" if series in ("(no_labels)", "aggregate") else series
        percent = "—" if s.get("change_percent") is None else f"{s['change_percent']:.1f}%"
        slope = "—" if s.get("slope_per_hour") is None else human_value(s["slope_per_hour"], unit, series)
        cols = [display_series, str(s.get("count", 0))]
        cols += [human_value(s.get(k, 0), unit, series) for k in ("first", "last", "change")]
        cols += [percent, slope]
        cols += [human_value(s.get(k, 0), unit, series) for k in ("mean", "p95", "max")]
        out.append("| " + " | ".join(cols) + " |")
    if len(stats_for_query) > top_n:
        out.append(f"\n_(showing top {top_n} of {len(stats_for_query)} series by p95)_")
    return "\n".join(out)


def k3k_role_summary(summary: dict) -> str:
    queries = (
        ("server", "CPU", "k3k_server_cpu_per_active_pod_millicores", "mCPU"),
        ("server", "Memory", "k3k_server_memory_per_active_pod_mb", "MB"),
        ("shared kubelet", "CPU", "k3k_kubelet_cpu_per_active_pod_millicores", "mCPU"),
        ("shared kubelet", "Memory", "k3k_kubelet_memory_per_active_pod_mb", "MB"),
        ("virtual agent", "CPU", "k3k_agent_cpu_per_active_pod_millicores", "mCPU"),
        ("virtual agent", "Memory", "k3k_agent_memory_per_active_pod_mb", "MB"),
        ("controller", "CPU", "k3k_controller_cpu_per_active_pod_millicores", "mCPU"),
        ("controller", "Memory", "k3k_controller_memory_per_active_pod_mb", "MB"),
    )
    rows = []
    for role, resource, query, unit in queries:
        series = summary.get(query, {})
        if not series:
            continue
        stats = next(iter(series.values()))
        change_percent = "—" if stats.get("change_percent") is None else f"{stats['change_percent']:.1f}%"
        slope = "—" if stats.get("slope_per_hour") is None else f"{stats['slope_per_hour']:.3f}"
        rows.append(
            f"| {role} | {resource} | {stats.get('count', 0)} | {stats.get('first', 0):.3f} | "
            f"{stats.get('last', 0):.3f} | {stats.get('change', 0):.3f} | {change_percent} | {slope} | "
            f"{stats.get('mean', 0):.3f} | {stats.get('p95', 0):.3f} | {stats.get('max', 0):.3f} | {unit} |"
        )
    if not rows:
        return "_(no normalized k3k role metrics)_"
    return "\n".join([
        "| role | resource | samples | first | last | change | change % | slope/hour | mean | p95 | max | unit |",
        "| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |",
        *rows,
    ])


def load_query_catalog(metrics_dir: Path) -> dict:
    path = metrics_dir / "queries.json"
    if not path.exists():
        return {}
    return {query["name"]: query for query in json.loads(path.read_text())}


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("metrics_dir", type=Path, help="directory produced by `dartboard collect-metrics`")
    parser.add_argument("--out", type=Path, default=None, help="output dir (default: <metrics_dir>)")
    args = parser.parse_args()

    metrics_dir = args.metrics_dir.resolve()
    out_dir = (args.out or metrics_dir).resolve()

    summary, manifest = recover_collection_metadata(metrics_dir)
    query_catalog = load_query_catalog(metrics_dir)

    md = [
        "# Metrics report",
        "",
        f"- **window:** {manifest['start']} → {manifest['end']}",
        f"- **step:** {manifest['step_seconds'] / 1e9:g}s",
        f"- **dart:** `{manifest.get('dart_file', '?')}`",
        f"- **workspace:** `{manifest.get('tofu_workspace', '?')}`",
        f"- **operational metrics:** `{manifest.get('operational', False)}`",
        "- **trend lines:** dashed lines are least-squares trends; table slopes are per hour",
        "- **Prometheus queries:** [queries.md](queries.md)" if (metrics_dir / "queries.md").exists() else "",
        "",
    ]

    if any(name.startswith("k3k_") for name in summary):
        md.extend([
            "## k3k role usage per active pod",
            "",
            k3k_role_summary(summary),
            "",
        ])

    for group in ("cpu", "memory", "disk", "network", "k3k", "operational"):
        group_dir = metrics_dir / group
        if not group_dir.is_dir():
            continue
        md.append(f"## {group}")
        md.append("")
        for csv_path in sorted(group_dir.glob("*.csv")):
            query_name = csv_path.stem
            png_rel = Path("charts") / group / f"{query_name}.png"
            ok = plot_csv(csv_path, out_dir / png_rel, query_name)
            if not ok:
                continue
            md.append(f"### {query_name}")
            md.append("")
            metadata = query_catalog.get(query_name, {})
            if metadata:
                md.append(metadata.get("description", ""))
                md.append("")
                md.append(f"- **Unit:** `{metadata.get('unit', '')}`")
                md.append(f"- **Query:** [view PromQL](queries.md#{query_name.replace('_', '-')})")
                md.append("")
            md.append(f"![{query_name}]({png_rel.as_posix()})")
            md.append("")
            md.append(stats_table(summary.get(query_name, {}), query_name))
            md.append("")

    report = out_dir / "report.md"
    report.write_text("\n".join(md))
    print(f"wrote {report}")
    print(f"charts under {out_dir / 'charts'}")


if __name__ == "__main__":
    main()
