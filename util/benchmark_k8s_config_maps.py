#!/usr/bin/python3

import base64
import json
import re
import statistics
import sys
import time

from kubernetes import client, config

NAMESPACE = "default"
CONFIG_MAP_NAME_PREFIX = "test-config-map-"
HOST = "upstream.local.gd"
PORT = 443
USERNAME = "admin"
PASSWORD = "adminpassword"
REPETITIONS = 5
DATA_SIZE = 10*1024 # 10 kiB


def delete_config_maps_exceeding(v1, start_index):
    raw_config_maps = v1.list_namespaced_config_map(namespace=NAMESPACE, watch=False, limit=5000)

    while raw_config_maps is not None:
        config_map_names = [raw_config_map.metadata.name for raw_config_map in raw_config_maps.items]
        print(f"Checking config_maps bunch starting with: {config_map_names[0]} for deletion", file=sys.stderr)

        deleted = 0
        errored = 0
        for name in config_map_names:
            m = re.match("test-config-map-([0-9]+)", name)
            if m is not None:
                index = int(m.group(1))
                if index > start_index:
                    try:
                        v1.delete_namespaced_config_map(name, NAMESPACE)
                        deleted += 1
                    except client.exceptions.ApiException as e:
                        if e.status == 404:
                            pass
                        else:
                            print(e, file=sys.stderr)
                            errored += 1

        if raw_config_maps.metadata._continue is not None:
            try:
                raw_config_maps = v1.list_namespaced_config_map(namespace=NAMESPACE, watch=False, limit=5000, _continue=raw_config_maps.metadata._continue)
            except client.exceptions.ApiException as e:
                if e.status == 410:
                    raw_config_maps = v1.list_namespaced_config_map(namespace=NAMESPACE, watch=False, limit=5000)
                else:
                    raise
        else:
            raw_config_maps = None
        print(f"Waiting {5} seconds...", file=sys.stderr)
        time.sleep(5)

    return deleted, errored


def create_config_maps(v1, start_index, end_index):
    data = base64.b64encode(b"a" * DATA_SIZE).decode("ascii")

    created = 0
    errored = 0
    for i in range(start_index, end_index):
        name = f"{CONFIG_MAP_NAME_PREFIX}{i}"
        try:
            v1.create_namespaced_config_map(namespace=NAMESPACE, body = {
                "metadata": {
                    "name": name,
                    "namespace": "default"
                },
                "data": {"data": data}
            })
            created += 1
        except Exception as e:
            print(e, file=sys.stderr)
            errored += 1
    return created, errored


def benchmark_k8s(v1, limit):
    # make request REPETITIONS times, saving runtimes
    runtimes = []
    bytes = []
    errors = 0
    for i in range(REPETITIONS):
        try:
            url="https://127.0.0.1:6443/api/v1/namespaces/default/configmaps"
            headers={'Accept': 'application/json', 'User-Agent': 'OpenAPI-Generator/12.0.1/python'}
            _preload_content=True
            _request_timeout=3600
            initial_query_params=[('limit', 5000), ('timeout', '3600s'), ('watch', False)]

            runtimes += [0]
            bytes += [0]
            _continue = "first"
            while _continue is not None:
                tic = time.perf_counter()
                query_params = initial_query_params
                if _continue != "first":
                    query_params = initial_query_params + [('continue', _continue)]
                response = v1.api_client.rest_client.GET(url=url, headers=headers, _preload_content=_preload_content, _request_timeout=_request_timeout, query_params=query_params)
                toc = time.perf_counter()
                runtimes[-1] += toc-tic
                bytes[-1] += len(response.data)
                if response.status != 200:
                    print(f"ERROR: status {response.status} at repetition {i}", file=sys.stderr)
                    errors += 1
                _continue = json.loads(response.data)['metadata'].get('continue')
        except Exception as e:
            print(f"ERROR: exception at repetition {i}: {e}", file=sys.stderr)
            errors += 1
            runtimes = runtimes[:-1]
            bytes = bytes[:-1]

    # print results
    mean = statistics.mean(runtimes)
    stdev = statistics.stdev(runtimes, mean)
    bytes = statistics.mode(bytes)

    return mean, stdev, bytes, errors


_, start_index_string, first_chunk_index_string, end_index_string = sys.argv
start_index = int(start_index_string)
current_chunk_index = int(first_chunk_index_string)
end_index = int(end_index_string)

config.load_kube_config()
v1 = client.CoreV1Api()

print("resources\tmean runtime (s)\tstdev (s)\tstdev (%)\tbytes\terrors")

while start_index < end_index:
    created, errored = create_config_maps(v1, start_index, current_chunk_index)
    print(f"Created: {created}, Errored: {errored}", file=sys.stderr)
    print(f"Waiting {created/500} seconds...", file=sys.stderr)
    time.sleep(created/500)
    print(f"Benchmarking {current_chunk_index} resources...", file=sys.stderr)
    mean, stdev, bytes, errors = benchmark_k8s(v1, current_chunk_index)
    percent = stdev/mean*100
    print(f"{current_chunk_index}\t{'{:.3f}'.format(mean)}\t{'{:.3f}'.format(stdev)}\t{'{:.2f}'.format(percent)}%\t{bytes}\t{errors}")
    sys.stdout.flush()
    start_index = current_chunk_index
    current_chunk_index *= 2
