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

.PHONY: build
build:
	go build $(GOTAGS) -v -o go-gpodder -ldflags $(LDFLAGS) \
		gitlab.com/kabes/go-gpodder/cli

.PHONY: run
run:
	air

.PHONY: clean
clean:
	rm -f go-gpodder

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run
	# go install go.uber.org/nilaway/cmd/nilaway@latest
	nilaway ./... || true
	typos

.PHONY: format
format:
	golangci-lint fmt


database.sqlite: schema.sql
	sqlite3 database.sqlite ".read schema.sql"




# vim:ft=make
