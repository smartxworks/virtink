LOCALBIN ?= $(shell pwd)/bin
ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_K8S_VERSION = 1.23
KIND ?= $(LOCALBIN)/kind
CMCTL ?= $(LOCALBIN)/cmctl
SKAFFOLD ?= $(LOCALBIN)/skaffold
KUTTL ?= $(LOCALBIN)/kuttl
KUBECTL ?= $(LOCALBIN)/kubectl
GOARCH ?= $(shell go env GOARCH)
GOOS ?= $(shell go env GOOS)

all: test

generate:
	iidfile=$$(mktemp /tmp/iid-XXXXXX) && \
	docker build -f hack/Dockerfile --iidfile $$iidfile . && \
	docker run --rm -v $$PWD:/go/src/github.com/smartxworks/virtink -w /go/src/github.com/smartxworks/virtink $$(cat $$iidfile) ./hack/generate.sh && \
	rm -rf $$iidfile

fmt:
	go fmt ./...

test: envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: envtest
envtest: $(ENVTEST)
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: kind
kind: $(KIND)
$(KIND): $(LOCALBIN)
	curl -sLo $(KIND) https://kind.sigs.k8s.io/dl/v0.14.0/kind-$(GOOS)-$(GOARCH) && chmod +x $(KIND)

.PHONY: kubectl
kubectl: $(KUBECTL)
$(KUBECTL): $(LOCALBIN)
	curl -sLo $(KUBECTL) https://dl.k8s.io/release/v1.24.0/bin/$(GOOS)/$(GOARCH)/kubectl && chmod +x $(KUBECTL)

.PHONY: cmctl
cmctl: $(CMCTL)
$(CMCTL): $(LOCALBIN)
	curl -sLo cmctl.tar.gz https://github.com/cert-manager/cert-manager/releases/download/v1.8.2/cmctl-$(GOOS)-$(GOARCH).tar.gz
	tar xzf cmctl.tar.gz -C $(LOCALBIN)
	rm -rf cmctl.tar.gz

.PHONY: skaffold
skaffold: $(SKAFFOLD)
$(SKAFFOLD): $(LOCALBIN)
	curl -sLo $(SKAFFOLD) https://storage.googleapis.com/skaffold/releases/latest/skaffold-$(GOOS)-$(GOARCH) && chmod +x $(SKAFFOLD)

.PHONY: kuttl
kuttl: $(KUTTL)
$(KUTTL): $(LOCALBIN)
	curl -sLo $(KUTTL) https://github.com/kudobuilder/kuttl/releases/download/v0.12.1/kubectl-kuttl_0.12.1_$(GOOS)_$(shell uname -m) && chmod +x $(KUTTL)

E2E_KIND_CLUSTER_NAME := virtink-e2e-$(shell date "+%Y-%m-%d-%H-%M-%S")
E2E_KIND_CLUSTER_KUBECONFIG := /tmp/$(E2E_KIND_CLUSTER_NAME).kubeconfig

.PHONY: e2e-image
e2e-image:
	docker buildx build -t virt-controller:e2e -f build/virt-controller/Dockerfile --build-arg PRERUNNER_IMAGE=virt-prerunner:e2e --load .
	docker buildx build -t virt-daemon:e2e -f build/virt-daemon/Dockerfile --load .
	docker buildx build -t virt-prerunner:e2e -f build/virt-prerunner/Dockerfile  --load .

e2e: kind kubectl cmctl skaffold kuttl e2e-image
	echo "e2e kind cluster: $(E2E_KIND_CLUSTER_NAME)"

	$(KIND) create cluster --config test/e2e/config/kind/config.yaml --name $(E2E_KIND_CLUSTER_NAME) --kubeconfig $(E2E_KIND_CLUSTER_KUBECONFIG)
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) virt-controller:e2e
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) virt-daemon:e2e
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) virt-prerunner:e2e

	docker pull docker.io/calico/cni:v3.23.5
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) docker.io/calico/cni:v3.23.5
	docker pull docker.io/calico/node:v3.23.5
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) docker.io/calico/node:v3.23.5
	docker pull docker.io/calico/kube-controllers:v3.23.5
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) docker.io/calico/kube-controllers:v3.23.5
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) apply -f https://projectcalico.docs.tigera.io/archive/v3.23/manifests/calico.yaml
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) wait -n kube-system deployment calico-kube-controllers --for condition=Available --timeout -1s

	docker pull quay.io/jetstack/cert-manager-controller:v1.8.2
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/jetstack/cert-manager-controller:v1.8.2
	docker pull quay.io/jetstack/cert-manager-cainjector:v1.8.2
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/jetstack/cert-manager-cainjector:v1.8.2
	docker pull quay.io/jetstack/cert-manager-webhook:v1.8.2
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/jetstack/cert-manager-webhook:v1.8.2
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.2/cert-manager.yaml
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(CMCTL) check api --wait=10m

	docker pull quay.io/kubevirt/cdi-operator:v1.53.0
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/kubevirt/cdi-operator:v1.53.0
	docker pull quay.io/kubevirt/cdi-apiserver:v1.53.0
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/kubevirt/cdi-apiserver:v1.53.0
	docker pull  quay.io/kubevirt/cdi-controller:v1.53.0
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/kubevirt/cdi-controller:v1.53.0
	docker pull quay.io/kubevirt/cdi-uploadproxy:v1.53.0
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/kubevirt/cdi-uploadproxy:v1.53.0
	docker pull quay.io/kubevirt/cdi-importer:v1.53.0
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) quay.io/kubevirt/cdi-importer:v1.53.0
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) apply -f https://github.com/kubevirt/containerized-data-importer/releases/download/v1.53.0/cdi-operator.yaml
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) wait -n cdi deployment cdi-operator --for condition=Available --timeout -1s
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) apply -f https://github.com/kubevirt/containerized-data-importer/releases/download/v1.53.0/cdi-cr.yaml
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) wait cdi.cdi.kubevirt.io cdi --for condition=Available --timeout -1s

	docker pull rook/nfs:master
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) rook/nfs:master
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) apply -f test/e2e/config/rook-nfs/crds.yaml
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) wait crd nfsservers.nfs.rook.io --for condition=Established
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) apply -f test/e2e/config/rook-nfs/

	PATH=$(LOCALBIN):$(PATH) $(SKAFFOLD) render --offline=true --default-repo="" --digest-source=tag --images virt-controller:e2e,virt-daemon:e2e | KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) apply -f -
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUBECTL) wait -n virtink-system deployment virt-controller --for condition=Available --timeout -1s

	docker pull smartxworks/virtink-kernel-5.15.12
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) smartxworks/virtink-kernel-5.15.12
	docker pull smartxworks/virtink-container-disk-ubuntu
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) smartxworks/virtink-container-disk-ubuntu
	docker pull smartxworks/virtink-container-rootfs-ubuntu
	$(KIND) load docker-image --name $(E2E_KIND_CLUSTER_NAME) smartxworks/virtink-container-rootfs-ubuntu
	KUBECONFIG=$(E2E_KIND_CLUSTER_KUBECONFIG) $(KUTTL) test --config test/e2e/kuttl-test.yaml

	$(KIND) delete cluster --name $(E2E_KIND_CLUSTER_NAME)
