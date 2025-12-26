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
	-X gitlab.com/kabes/go-gpo/internal/config.Version=$(VERSION) \
	-X gitlab.com/kabes/go-gpo/internal/config.Revision='$(REVISION) \
	-X gitlab.com/kabes/go-gpo/internal/config.BuildDate=$(DATE) \
	-X gitlab.com/kabes/go-gpo/internal/config.BuildUser=$(USER) \
	-X gitlab.com/kabes/go-gpo/internal/config.Branch=$(BRANCH)"
LDFLAGSR="-w -s\
	-X gitlab.com/kabes/go-gpo/internal/config.Version=$(VERSION) \
	-X gitlab.com/kabes/go-gpo/internal/config.Revision='$(REVISION) \
	-X gitlab.com/kabes/go-gpo/internal/config.BuildDate=$(DATE) \
	-X gitlab.com/kabes/go-gpo/internal/config.BuildUser=$(USER) \
	-X gitlab.com/kabes/go-gpo/internal/config.Branch=$(BRANCH)"

.PHONY: build
build: generate
	go build $(GOTAGS) -v -o go-gpo -ldflags $(LDFLAGS) \
		./cli

.PHONY: build_arm64
build_arm64: generate
	CGO_ENABLED=1 \
	GOGCCFLAGS="-fPIC -O4 -Ofast -pipe -march=native -s" \
		GOARCH=arm64 GOOS=linux \
		go build -v -o go-gpo-arm64 --ldflags $(LDFLAGS) \
		./cli

.PHONY: build_arm64_release
build_arm64_release: generate
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
	find . -type f -name '*.qtpl.go' -delete


.PHONY: test
test:
	go test -coverprofile=cover.out ./...
	go tool cover -html=cover.out -o cover.html


.PHONY: lint
lint:
	golangci-lint run
	# go install fillmore-labs.com/errortype@latest
	errortype ./...
	typos
	# go install go.uber.org/nilaway/cmd/nilaway@latest
	nilaway ./...

.PHONY: format
format:
	golangci-lint fmt


database.sqlite: schema.sql
	sqlite3 database.sqlite ".read schema.sql"

migrate:
	goose -dir ./internal/infra/sqlite/migrations sqlite3 ./database.sqlite up

migrate_pg:
	goose -dir ./internal/infra/pg/migrations postgres "user=gogpo dbname=gogpo password=gogpo123 host=127.0.0.1" up

.PHONY: deps
deps:
	go get -u ./...
	go mod tidy
	$(MAKE) test
	$(MAKE) build


QTPLS := $(shell find . -type f -name '*.qtpl')
QTPLSC := $(QTPLS:%=%.go)

generate: $(QTPLSC)

%.qtpl.go: %.qtpl
	qtc -file $<

.PHONY: clean
prepare:
	go install github.com/valyala/quicktemplate/qtc
	go mod tidy


# vim:ft=make
