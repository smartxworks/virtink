#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

bash "$GOPATH"/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/smartxworks/virtink/pkg/generated github.com/smartxworks/virtink/pkg/apis \
  virt:v1alpha1 \
  --go-header-file ./hack/boilerplate.go.txt

dir="deploy/helm/virtink/templates"

controller-gen paths=./pkg/apis/... crd output:crd:artifacts:config=deploy/crd
controller-gen paths=./cmd/virt-controller/... paths=./pkg/controller/... rbac:roleName=virt-controller \
  output:rbac:artifacts:config="$dir"/virt-controller \
  webhook output:webhook:artifacts:config="$dir"/virt-controller
controller-gen paths=./cmd/virt-daemon/... paths=./pkg/daemon/... rbac:roleName=virt-daemon \
  output:rbac:artifacts:config="$dir"/virt-daemon

# TODO: should use a more elegant way for editing generated manifests.yaml
replace="  name: virt-controller
  annotations:
    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/virt-controller-cert"

# Replace newlines with literal \n, replace \ -> \/ for sed replace below
replace="$(echo "${replace//$'\n'/\\n}" | sed "s/\//\\\\\//g")"

sed -i "s/  name: mutating-webhook-configuration/$replace/g;
  s/  name: validating-webhook-configuration/$replace/g;
  s/name: webhook-service/name: virt-controller/g;
  s/namespace: system/namespace: {{ .Release.Namespace }}/g" "$dir"/virt-controller/manifests.yaml

go generate ./...
