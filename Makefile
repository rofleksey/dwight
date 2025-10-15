.PHONY: all
all: gen build

.PHONY: clean
clean:
	@go clean

.PHONY: gen
gen:
	@echo "Generating dependency files..."
	@go generate ./...

.PHONY: lint
lint:
	@golangci-lint run

.PHONY: build
build:
	@go build -ldflags "-X go.szostok.io/version.version=${GIT_TAG} -X 'go.szostok.io/version.buildDate=`date`' -X go.szostok.io/version.commit=${GIT_COMMIT} -X go.szostok.io/version.commitDate=${GIT_COMMIT_DATE}" -o dwight .

.PHONY: install
install:
	@go install -ldflags "-X go.szostok.io/version.version=${GIT_TAG} -X 'go.szostok.io/version.buildDate=`date`' -X go.szostok.io/version.commit=${GIT_COMMIT} -X go.szostok.io/version.commitDate=${GIT_COMMIT_DATE}" .

.PHONY: run
run:
	@./dwight