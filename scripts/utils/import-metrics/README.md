# Import Metrics

A script to collect and aggregate TSDB of cluster metrics via Grafana mimirtool's `remote-read` using only a kubeconfig from the target cluster.

#### Target Requirements

 - Rancher 2.9+ with Monitoring installed

## Usage

`./import-metrics /path/to/kubeconfig.yaml selector from to`

- Example 1: `./import-metrics.sh path/to/kubeconfig.yaml '{__name__!=""}' 2025-03-03T23:12:52Z 2025-03-04T02:12:30Z`

- Example 2: `./import-metrics.sh /path/to/kubeconfig.yaml '{__name__!=""}' $(date -u -v-4H +"%Y-%m-%dT%H:%M:%SZ") $(date -u -v-2H +"%Y-%m-%dT%H:%M:%SZ")`

   - Time format for date ranges `YYYY-MM-DDThh:mm:ssZ`
     - Example: `2025-03-03T23:12:52Z`
     - See below for date command helpers
   - Selector is a valid metric for prometheus query in single quotes `'{__name__!=""}'`
     - Example: `'go_memstats_heap_inuse_bytes{service=~"rancher-monitoring-operator"}'`
     - Example: `'{__name__="go_memstats_heap_inuse_bytes"}'`
     - Use `'{__name__!=""}'` to query all metrics, avoid any operations like `sum`

Query selector and date range can be set via cli arguments

  - Usage: `./import-metrics.sh kubeconfig selector from to`
  - Example: `./import-metrics.sh /path/to/kubeconfig.yaml '{__name__!=""}' 2025-03-03T23:12:52Z 2025-03-04T02:12:30Z`
 
 Queries containing only one "from" date will utilize the current time at execution as the "to" date

  - Usage: `./import-metrics.sh kubeconfig selector from`
  - Example: `./import-metrics.sh /path/to/kubeconfig.yaml '{__name__!=""}' 2025-03-03T23:12:52Z `
 
 Set date command as argument to help query specific range of hours (i.e. 36 hours)

  - Usage: `./import-metrics.sh kubeconfig selector $(date command)`
  - Example: `./import-metrics.sh /path/to/kubeconfig.yaml '{__name__!=""}' $(date -u -v-36H +"%Y-%m-%dT%H:%M:%SZ")`

 Queries with no date range will utilize default range of one hour from current time

   - Usage: `./import-metrics.sh kubeconfig selector`
   - Example: `./import-metrics.sh /path/to/kubeconfig.yaml '{__name__!=""}'`

 Queries with no selector will target ALL metrics

   - Usage: `./import-metrics.sh kubeconfig`
   - Example: `./import-metrics.sh /path/to/kubeconfig.yaml`

#### Helpers

 - Bash helper for obtaining properly formatted date

   - `date -u +"%Y-%m-%dT%H:%M:%SZ`

 - Flag usage can help define a time range `[-v[+|-]val[y|m|w|d|H|M|S]]`
 
   - `date -u -v-36H +"%Y-%m-%dT%H:%M:%SZ"`

#### Experimental

- To increase execution speed and reduce quantity of raw data files on large imports, the time range of each individual query can be increased by including an OFFSET(in seconds) argument at execution.
 
  - Usage: `./import-metrics /path/to/kubeconfig.yaml selector from to OFFSET`

- Warning! When using Rancher Monitoring's default Prometheus memory allocation of 3000Mi, OOM errors are likely to occur with an OFFSET larger than one hour (3600 seconds) while querying for ALL METRICS. Increasing the Prometheus installation memory to 10000Mi allows for more consistent pulls of ALL METRICS using a two hour increment (7200 second) for the OFFSET however, further increases to memory DO NOT reliably allow larger query time ranges increments and some failures have been observed during testing. To increase stability, OFFSET has been set with a default of one hour and limited to two hours.

### Notes

 - Data from queries with a time range that reaches into the previous day(s) will be aggregated into the `metrics-(YYYY-MM-DD)` directory corresponding to the date of execution, raw data files are timestamped according the query start time

