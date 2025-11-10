#
# Makefile
#

# enable sdjournal
#GOTAGS=
GOTAGS=-tags 'sdjournal'

#
VERSION=`git describe --always`
REVISION=`git rev-parse HEAD`
DATE=`date +%Y%m%d%H%M%S`
USER=`whoami`
# BRANCH=`git branch | grep '^\*' | cut -d ' ' -f 2`
BRANCH=`git rev-parse --abbrev-ref HEAD`
LDFLAGS="\
	-X main.Version=$(VERSION) \
	-X main.Revision='$(REVISION) \
	-X main.BuildDate=$(DATE) \
	-X main.BuildUser=$(USER) \
	-X main.Branch=$(BRANCH)"
LDFLAGSR="-w -s\
	-X main.Version=$(VERSION) \
	-X main.Revision='$(REVISION) \
	-X main.BuildDate=$(DATE) \
	-X main.BuildUser=$(USER) \
	-X main.Branch=$(BRANCH)"

.PHONY: build
build:
	go build $(GOTAGS) -v -o go-gpo -ldflags $(LDFLAGS) \
		./cli

.PHONY: build_arm64
build_arm64:
	CGO_ENABLED=1 \
	GOGCCFLAGS="-fPIC -O4 -Ofast -pipe -march=native -s" \
		GOARCH=arm64 GOOS=linux \
		go build -v -o go-gpo-arm64 --ldflags $(LDFLAGS) \
		./cli

.PHONY: build_arm64_release
build_arm64_release:
	CGO_ENABLED=1 \
	GOGCCFLAGS="-fPIC -O4 -Ofast -pipe -march=native -s" \
		GOARCH=arm64 GOOS=linux \
		go build -trimpath -v -o go-gpo-arm64 --ldflags $(LDFLAGSR) \
		./cli


.PHONY: run
run:
	air

.PHONY: clean
clean:
	rm -f go-gpo

.PHONY: test
test:
	go test -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html


.PHONY: lint
lint: 
	golangci-lint run
	# go install fillmore-labs.com/errortype@latest
	errortype ./... || true
	# go install go.uber.org/nilaway/cmd/nilaway@latest
	nilaway ./... || true
	typos

.PHONY: format
format:
	golangci-lint fmt


database.sqlite: schema.sql
	sqlite3 database.sqlite ".read schema.sql"

migrate:
	goose -dir ./internal/cmd/migrations/ sqlite3 ./database.sqlite up


# vim:ft=make
