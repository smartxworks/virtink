#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

truncate -s $2 $1
mkfs.ext4 -d /rootfs $1
