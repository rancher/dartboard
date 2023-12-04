#!/bin/sh
#

RES="management.cattle.io.setting \
     management.cattle.io.clusters \
     rbac.authorization.k8s.io.rolebindings"

if [ -z "$BASE_URL" ]; then
	echo "BASE_URL env is missing"
	exit 1
fi


urlShort="${BASE_URL/https:\/\/}"
urlShort="${urlShort/:*}"
fileBaseName="benchmark_${urlShort}"

get_values() {
	echo "Retrieve files from $BASE_URL"
	for res in $RES; do
		echo "* Download ${fileBaseName}_${res}.txt"
		kubectl cp k6-manual-run:/home/k6/${fileBaseName}_${res}.txt ./${fileBaseName}_${res}.txt
	done
}

if [ "$1" = "get" ]; then
	get_values
	exit 0
fi


[ ! -f ./steve_paginated_api_benchmark.js ] && wget https://raw.githubusercontent.com/moio/scalability-tests/20231201_aks_rke_comparison/k6/steve_paginated_api_benchmark.js

echo "Running benchmark against $BASE_URL"
for resource in $RES ; do
	fileName="${fileBaseName}_$resource.txt"
	echo "* Testing resource ${resource} - output to: $fileName"
	k6 run  -e BASE_URL=${BASE_URL} \
		-e USERNAME=admin \
		-e PASSWORD=adminadminadmin \
		-e RESOURCE=${resource} \
		./steve_paginated_api_benchmark.js > "$fileName"
done
