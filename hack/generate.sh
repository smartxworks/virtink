#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

bash "$GOPATH"/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/smartxworks/virtink/pkg/generated github.com/smartxworks/virtink/pkg/apis \
  virt:v1alpha1 \
  --go-header-file ./hack/boilerplate.go.txt

dir="deploy/helm/virtink/templates"
webhook_dir="hack/webhook"

controller-gen paths=./pkg/apis/... crd output:crd:artifacts:config=deploy/crd
controller-gen paths=./cmd/virt-controller/... paths=./pkg/controller/... rbac:roleName=virt-controller \
  output:rbac:artifacts:config="$dir"/virt-controller \
  webhook output:webhook:artifacts:config="$webhook_dir"
controller-gen paths=./cmd/virt-daemon/... paths=./pkg/daemon/... rbac:roleName=virt-daemon \
  output:rbac:artifacts:config="$dir"/virt-daemon

kustomize build "$webhook_dir" > "$dir"/virt-controller/manifests.yaml

go generate ./...
