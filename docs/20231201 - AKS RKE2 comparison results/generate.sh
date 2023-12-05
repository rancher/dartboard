#!/bin/sh

for res in clusters configmaps rolebindings setting; do
	outputFile=${res}.csv
	if [ -f "$outputFile" ]; then
		mv $outputFile $outputFile.bkup
	fi
	fileList=$(ls benchmark_*${res}*)
	for file in $fileList; do
		./parser.js $file $outputFile
	done
done
