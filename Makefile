LOCALBIN ?= $(shell pwd)/bin
ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_K8S_VERSION = 1.23

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
