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
from pathlib import Path

import matplotlib.dates as mdates
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import pandas as pd

# Display unit per query (matches internal/metrics/catalog.go).
UNITS = {
    "node_cpu_utilization":        "ratio",
    "node_load1":                  "load",
    "rancher_pod_cpu":             "cores",
    "node_mem_used_bytes":         "bytes",
    "rancher_pod_workingset":      "bytes",
    "rancher_pod_rss":             "bytes",
    "node_disk_throughput":        "bytes/sec",
    "node_disk_iops":              "iops",
    "node_filesystem_utilization": "ratio",
    "node_net_throughput":         "bytes/sec",
    "node_net_errors":             "events/sec",
    "pod_resource_requests":       "mixed",
    "pod_resource_limits":         "mixed",
}

# Which label columns identify a "series" for legend purposes. Anything not
# listed falls back to all non-(timestamp,value) columns.
SERIES_LABEL = {
    "node_cpu_utilization":        ["instance"],
    "node_load1":                  ["instance"],
    "rancher_pod_cpu":             ["namespace", "pod"],
    "node_mem_used_bytes":         ["instance"],
    "rancher_pod_workingset":      ["namespace", "pod"],
    "rancher_pod_rss":             ["namespace", "pod"],
    "node_disk_throughput":        ["instance", "device"],
    "node_disk_iops":              ["instance", "device"],
    "node_filesystem_utilization": ["instance", "mountpoint"],
    "node_net_throughput":         ["instance", "device"],
    "node_net_errors":             ["instance", "device"],
    "pod_resource_requests":       ["namespace", "pod", "resource"],
    "pod_resource_limits":         ["namespace", "pod", "resource"],
}

MAX_LEGEND_LINES = 12


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

    unit = UNITS.get(query_name, "")
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
    unit = UNITS.get(query_name, "")
    out = [
        "| series | min | p50 | p95 | p99 | max | mean |",
        "| --- | --- | --- | --- | --- | --- | --- |",
    ]
    for series, s in rows:
        cols = [series] + [human_value(s.get(k, 0), unit, series) for k in ("min", "p50", "p95", "p99", "max", "mean")]
        out.append("| " + " | ".join(cols) + " |")
    if len(stats_for_query) > top_n:
        out.append(f"\n_(showing top {top_n} of {len(stats_for_query)} series by p95)_")
    return "\n".join(out)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("metrics_dir", type=Path, help="directory produced by `dartboard collect-metrics`")
    parser.add_argument("--out", type=Path, default=None, help="output dir (default: <metrics_dir>)")
    args = parser.parse_args()

    metrics_dir = args.metrics_dir.resolve()
    out_dir = (args.out or metrics_dir).resolve()

    summary = json.loads((metrics_dir / "summary.json").read_text())
    manifest = json.loads((metrics_dir / "manifest.json").read_text())

    md = [
        "# Metrics report",
        "",
        f"- **window:** {manifest['start']} → {manifest['end']}",
        f"- **step:** {manifest['step_seconds'] / 1e9:g}s",
        f"- **dart:** `{manifest.get('dart_file', '?')}`",
        f"- **workspace:** `{manifest.get('tofu_workspace', '?')}`",
        "",
    ]

    for group in ("cpu", "memory", "disk", "network", "pod"):
        group_dir = metrics_dir / group
        if not group_dir.is_dir():
            continue
        md.append(f"## {group}")
        md.append("")
        for csv_path in sorted(group_dir.glob("*.csv")):
            query_name = csv_path.stem
            png_rel = Path("charts") / group / f"{query_name}.png"
            ok = plot_csv(csv_path, out_dir / png_rel, query_name)
            md.append(f"### {query_name}")
            md.append("")
            md.append(f"![{query_name}]({png_rel.as_posix()})" if ok else "_(no data in this window)_")
            md.append("")
            md.append(stats_table(summary.get(query_name, {}), query_name))
            md.append("")

    report = out_dir / "report.md"
    report.write_text("\n".join(md))
    print(f"wrote {report}")
    print(f"charts under {out_dir / 'charts'}")


if __name__ == "__main__":
    main()
