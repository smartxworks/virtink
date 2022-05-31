#!/bin/sh

# Copyright (C) 2022 SmartX, Inc. <info@smartx.com>
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

set -o errexit
set -o nounset
set -o pipefail

case $1 in
  "cloud-init")
    temp=$(mktemp -d)
    echo "$2" | base64 -d > $temp/meta-data

    if [[ "$3" =~ ^/.* ]]; then
      cp $3 $temp/user-data
    else
      echo "$3" | base64 -d > $temp/user-data
    fi

    if [[ "$4" =~ ^/.* ]]; then
      cp $4 $temp/network-data
    else
      echo "$4" | base64 -d > $temp/network-data
    fi

    genisoimage -volid cidata -joliet -rock -output $5 $temp
    ;;
esac
