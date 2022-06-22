#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

qemu-img convert /disk $1
