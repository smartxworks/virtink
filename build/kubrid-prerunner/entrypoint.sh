#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

ch_cmd=$(kubrid-prerunner $@)
sh -c "$ch_cmd"
