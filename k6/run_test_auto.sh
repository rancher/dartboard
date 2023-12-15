#!/bin/sh
#

RES="management.cattle.io.setting \
     management.cattle.io.clusters \
     configmaps"


#urlShort="${BASE_URL/https:\/\/}"
#urlShort="${urlShort/:*}"
fileBaseName="benchmark"

get_values() {
	local outputSuffix="$1"
	if [ -z "$outputSuffix" ]; then
		echo "instance name is missing"
		exit 1
	fi

	echo "Retrieve test files"
	for res in $RES; do
		local fileName="${fileBaseName}_${res}.txt"
		local outputFileName="${fileBaseName}_${outputSuffix}_${res}.txt"
		if [ -f "$outputFileName" ]; then
			echo "file $outputFileName already present, would overwrite: quitting"
			exit 1
		fi
		echo "* Download ${fileName}"
		kubectl cp k6-manual-run:/home/k6/${fileName} ./${outputFileName}

	done

}

if [ "$1" = "get" ]; then
	get_values "$2"
	exit 0
fi

if [ -z "$BASE_URL" ]; then
	echo "BASE_URL env is missing"
	exit 1
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
		./steve_paginated_api_benchmark.js | tee "$fileName"
done
