#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

bash $GOPATH/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/smartxworks/virtink/pkg/generated github.com/smartxworks/virtink/pkg/apis \
  virt:v1alpha1 \
  --go-header-file ./hack/boilerplate.go.txt

controller-gen paths=./pkg/apis/... crd output:crd:artifacts:config=deploy/crd
controller-gen paths=./cmd/virt-controller/... paths=./pkg/controller/... rbac:roleName=virt-controller output:rbac:artifacts:config=deploy/virt-controller webhook output:webhook:artifacts:config=deploy/virt-controller
controller-gen paths=./cmd/virt-daemon/... paths=./pkg/daemon/... rbac:roleName=virt-daemon output:rbac:artifacts:config=deploy/virt-daemon

go generate ./...
