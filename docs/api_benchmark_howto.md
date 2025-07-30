# Rancher API Benchmark HOWTO

During Rancher development or troubleshooting it may be useful to check whether a certain API endpoint is returning slower than expected.

This document explains how to get specific, repeatable measurements via the <https://k6.io/> tool, which can also be used standalone, outside the context of the `dartboard` project.

## Requirements

- a host that can access Rancher
- k6, either:
  - installed on that host: <https://k6.io/docs/get-started/installation/>, or
  - run via Docker. Only a Docker daemon up and running is necessary, no installation required
- an API benchmark script, eg. <https://github.com/rancher/dartboard/raw/refs/heads/main/k6/tests/api_benchmark.js>

## Running a test

Basic usage (from an installed k6):

```sh
k6 run -e BASE_URL=<BASE_URL> ./api_benchmark.js |& tee benchmark_output_`date +"%Y-%m-%d_%H-%M-%S"`.txt
```

Replace `<BASE_URL>` appropriately, for example:

```sh
k6 run -e BASE_URL=https://upstream.local.gd:8443 ./api_benchmark.js |& tee benchmark_output_`date +"%Y-%m-%d_%H-%M-%S"`.txt
```

### Advanced usage

- running in Docker: substitute `k6 run` with `docker run --rm -i grafana/k6 run`
- additional parameters: 
  - `-e USERNAME=<USERNAME>`: specify a username to log in (local authentication only) (default: no authentication)
  - `-e PASSWORD=<PASSWORD>`: specify a password to log in (local authentication only) (default: no authentication)
  - `-e TOKEN=<TOKEN>`: specify an authentication token to log in (alternative to `-e USERNAME` and `-e PASSWORD`, works for external authentication). `<TOKEN>` must be a string starting with `token-` and can be collected via the procedure below (default: no authentication)
  - `-e RESOURCE=<RESOURCE>`: benchmark fetching of a specific resource type (eg. namespaces, pods, etc.) (default: [`management.cattle.io`](http://management.cattle.io/)`.setting`)
  - `-e API=norman`: benchmark the `norman` (v3) API (default: `steve (v1)`)
  - `-e VUS=<N>`: run `N` concurrent simulated users (default: 1)
  - `-e PER_VU_ITERATIONS=<N>`: repeat API calls N times for statistical relevance (default: 30)
  - `-e CLUSTER=<CLUSTER_ID>`: benchmark the API of a downstream cluster. `<CLUSTER_ID>` must be a string starting with `c-m-` and can be collected with `kubectl get -A clusters.management.cattle.io` (default: local cluster)
  - `-e PAGINATION_STYLE=steve`: benchmark the new Steve pagination style introduced in <https://github.com/rancher/steve/pull/63>. Ignored if `API=norman` (default: `k8s` native pagination style)
  - `-e PAGE_SIZE=1000`: page results in batches of 1000 (default: `100`)
  - `-e URL_SUFFIX="&link=index"`: add a suffix to request URLs (default: no suffix)
  - `-e PAUSE_SECONDS=5`: add an average pause of 5 seconds after each request (simulated click)

### Creating a \<TOKEN\> with an API Key

- Open Rancher in a browser, log in
- Click on the user icon in the top-right corner
- Click on "Account and API Keys"
- Click on "Create API Key"
- Add a Description, click Create
- Copy the "Bearer Token" value

## Interpreting results

Outputs are saved in files like `benchmark_output_2023-09-01_12-31-01.txt`. Most important to check are the following:

- **checks should report no errors:**

Good example line:
```
✓ checks.........................: 100.00% ✓ 61        ✗ 0
```

Bad example line (see below):
```
✗ checks.....................: 0.00%   ✓ 0          ✗ 31
```

- **request duration (especially p(95)) should be less than a second in most cases:**

```
http_req_duration..............: avg=35.49ms  min=7.28ms med=31.21ms max=96.76ms  p(90)=64.98ms  p(95)=69.57ms
```

- **otherwise, check if download time plays a major role**: `http_req_receiving` is the portion of `http_req_duration` spent downloading. If it is in the seconds, then it could mean the link between browser and Rancher is slow.

## What if... checks fail (errors are reported)

That means some of the requests returned errors. To investigate, add the `--http-debug` flag to `k6 run` and inspect the response statuses. `--http-debug=full` can be used to get more information if that is not sufficient, but it is more verbose. Additionally, logs of the rancher/rancher pod may contain indications of the reason for the failures.

**Please note benchmark results are generally not valid in presence of errors.**

Root causes need to be fixed and tests should pass without errors before they can be considered valid.
