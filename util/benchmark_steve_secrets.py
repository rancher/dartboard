#!/usr/bin/python3

import base64
import json
import statistics
import sys
import time

from kubernetes import client, config
import requests

NAMESPACE = "default"
SECRET_NAME_PREFIX = "test-secret-"
HOST = "upstream.local.gd"
PORT = 443
USERNAME = "admin"
PASSWORD = "adminpassword"
REPETITIONS = 5


def create_secrets(v1, start_index, end_index, data_size):
    data = base64.b64encode(b"a" * data_size).decode("ascii")
    created = 0
    errored = 0
    for i in range(start_index, end_index):
        name = f"{SECRET_NAME_PREFIX}{i}"
        try:
            v1.create_namespaced_secret(namespace=NAMESPACE, body={
                "type": "Opaque",
                "metadata": {
                    "name": name,
                    "namespace": "default"
                },
                "data": {"data": data}
            })
            created += 1
        except Exception as e:
            errored += 1
    return created, errored


def get_steve_login_cookies():
    requests.packages.urllib3.disable_warnings()
    # curl --insecure --cookie-jar cookie.jar 'https://upstream.local.gd:443/v3-public/localProviders/local?action=login' -X POST --data-raw '{"description":"UI session","responseType":"cookie","username":"admin","password":"adminpassword"}'
    response = requests.post(f"https://{HOST}:{PORT}/v3-public/localProviders/local?action=login", data=json.dumps({
        "description": "UI session",
        "responseType": "cookie",
        "username": USERNAME,
        "password": PASSWORD
    }), verify=False)
    return response.cookies


def benchmark_steve(cookies, limit):
    # make request REPETITIONS times, saving runtimes
    runtimes = []
    bytes = []
    errors = 0
    for i in range(REPETITIONS):
        try:
            tic = time.perf_counter()
            # curl --insecure --cookie cookie.jar 'https://upstream.local.gd:443/v1/secrets?limit=300000'
            # limit must be set or steve will cap the number of returned objects
            # see https://github.com/rancher/steve/blob/a10fe811f58ff7d92ffee51dd3f11010afbaf115/pkg/stores/partition/store.go#L179-L191
            response = requests.get(f"https://{HOST}:{PORT}/v1/secrets?limit={limit}", cookies=cookies, verify=False)
            toc = time.perf_counter()
            if response.status_code == 200:
                bytes = bytes + [len(response.content)]
                runtimes = runtimes + [toc-tic]
            else:
                print(f"ERROR: status {response.status_code} at repetition {i}, {response.content}", file=sys.stderr)
                errors += 1
        except Exception as e:
            print(f"ERROR: exception at repetition {i}", file=sys.stderr)
            errors += 1

    # print results
    mean = statistics.mean(runtimes)
    stdev = statistics.stdev(runtimes, mean)
    bytes = statistics.mode(bytes)

    return mean, stdev, bytes, errors


_, start_index_string, first_chunk_index_string, end_index_string, data_size_string = sys.argv
start_index = int(start_index_string)
current_chunk_index = int(first_chunk_index_string)
end_index = int(end_index_string)
data_size = int(data_size_string)

config.load_kube_config()
v1 = client.CoreV1Api()

cookies = get_steve_login_cookies()

print("resources\tmean runtime (s)\tstdev (s)\tstdev (%)\tbytes\terrors")

while start_index < end_index:
    created, errored = create_secrets(v1, start_index, current_chunk_index, data_size)
    print(f"Created: {created}, Errored: {errored}", file=sys.stderr)
    print(f"Waiting {created/500} seconds...", file=sys.stderr)
    time.sleep(created/500)
    print(f"Benchmarking {current_chunk_index} resources...", file=sys.stderr)
    mean, stdev, bytes, errors = benchmark_steve(cookies, current_chunk_index)
    percent = stdev/mean*100
    print(f"{current_chunk_index}\t{'{:.3f}'.format(mean)}\t{'{:.3f}'.format(stdev)}\t{'{:.2f}'.format(percent)}%\t{bytes}\t{errors}")
    sys.stdout.flush()
    start_index = current_chunk_index
    current_chunk_index *= 2
