all: test

generate:
	iidfile=$$(mktemp /tmp/iid-XXXXXX) && \
	docker build -f hack/Dockerfile --iidfile $$iidfile . && \
	docker run --rm -v $$PWD:/go/src/github.com/smartxworks/virtink -w /go/src/github.com/smartxworks/virtink $$(cat $$iidfile) ./hack/generate.sh && \
	rm -rf $$iidfile

fmt:
	go fmt ./...

test:
	go test ./... -coverprofile cover.out
