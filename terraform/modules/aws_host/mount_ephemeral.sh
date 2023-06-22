#!/usr/bin/env bash

set -xe

# if this host has a local SSD device (eg. m6i family), format the partition and mount it to /data
for device in `ls /dev/nvme?n?`; do
  if [ -b ${device} ] && [ ! -b ${device}p1 ]; then
    /usr/sbin/parted -s ${device} mklabel gpt
    /usr/sbin/parted -s ${device} mkpart primary 0% 100%

    # HACK: allow kernel time before calling mkfs
    sleep 1

    /sbin/mkfs.xfs ${device}p1
    mkdir -p /data
    mount ${device}p1 /data

    exit 0
  fi
done
