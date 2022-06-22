#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

cp /vmlinux $1
