#!/bin/bash
set -e

while [ ! -f /var/lib/cloud/instance/boot-finished ]; do
  echo "Waiting for cloud-init to complate..."
  sleep 5
done

until curl -s -f $1 ; do
  echo "Waiting for nomad to become available..."
  sleep 5
done