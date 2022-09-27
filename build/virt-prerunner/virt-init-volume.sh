#!/bin/sh

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
      cp $4 $temp/network-config
    else
      echo "$4" | base64 -d > $temp/network-config
    fi

    genisoimage -volid cidata -joliet -rock -output $5 $temp
    ;;
esac
