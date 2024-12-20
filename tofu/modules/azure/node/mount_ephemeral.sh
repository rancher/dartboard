#!/usr/bin/env bash

set -xe

# Azure ephemeral local disk are mounted by default on /mnt and formatted as ext4
# let's re-format as xfs and mount on /data
partition=$(findmnt -nM /mnt | awk '{print $2}')
case "$partition" in
  "/dev/sd"*)
    # ensure the ephemeral disk is big enough, i.e., 8 GB, or skip
    size=$(findmnt -nM /mnt -bo size)
    if [ $size -gt $(( 8 * 1024*1024*1024)) ]; then
      sudo umount /mnt
      sudo /sbin/mkfs.xfs -f ${partition}
      sudo mkdir -p /data
      sudo mount ${partition} /data
    fi
    ;;
esac
