#!/usr/bin/env bash

set -xe

# Azure ephemeral local disk are mounted by default on /mnt and formatted as ext4
# let's re-format as xfs and mount on /data
partition=$(findmnt -nM /mnt | awk '{print $2}')
case "$partition" in
  "/dev/sd"*)
    sudo umount /mnt
    sudo /sbin/mkfs.xfs -f ${partition}
    sudo mkdir -p /data
    sudo mount ${partition} /data
    ;;
esac
