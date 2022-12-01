#!/usr/bin/python3

import statistics
import sys
import time

from kubernetes import client, config

NAMESPACE = "default"
HOST="upstream.local.gd"
PORT=443
USERNAME="admin"
PASSWORD="adminpassword"
REPETITIONS=1000


config.load_kube_config()
v1 = client.AuthorizationV1Api()

runtimes = []
for i in range(REPETITIONS):
    tic = time.perf_counter()
    response = v1.create_namespaced_local_subject_access_review(namespace=NAMESPACE, body=
        {"spec": {
            "user": USERNAME,
            "groups": ["system:authenticated", "system:masters"],
            "resourceAttributes": {
                "verb": "list",
                "namespace": NAMESPACE,
                "group": "apps",
                "version": "v1",
                "resource": "deployments",
            }
        },
    })
    toc = time.perf_counter()
    runtimes += [toc-tic]
    if not response.status.allowed:
        print(f"ERROR: status {response.status} at repetition {i}", file=sys.stderr)

mean = statistics.mean(runtimes)
stdev = statistics.stdev(runtimes, mean)
percent = stdev/mean

print(f"repetitions: {REPETITIONS}, mean runtime (s): {'{:.3f}'.format(mean)}, stdev (s): {'{:.3f}'.format(stdev)}, stdev (%): {'{:.2f}'.format(percent)}")
