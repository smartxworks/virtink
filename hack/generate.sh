#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

bash $GOPATH/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/smartxworks/kubrid/pkg/generated github.com/smartxworks/kubrid/pkg/apis \
  kubrid:v1alpha1 \
  --go-header-file ./hack/boilerplate.go.txt

controller-gen paths=./pkg/apis/... crd output:crd:artifacts:config=deploy/crd
controller-gen paths=./cmd/kubrid-controller/... paths=./pkg/controller/... rbac:roleName=kubrid-controller output:rbac:artifacts:config=deploy/kubrid-controller webhook output:webhook:artifacts:config=deploy/kubrid-controller
controller-gen paths=./cmd/kubrid-daemon/... paths=./pkg/daemon/... rbac:roleName=kubrid-daemon output:rbac:artifacts:config=deploy/kubrid-daemon

go generate ./...
