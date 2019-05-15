ifndef VERSION
	VERSION = `git describe --always --long --dirty`
endif

# https://wiki.debian.org/ReproducibleBuilds/TimestampsProposal
ifndef SOURCE_DATE_EPOCH
	SOURCE_DATE_EPOCH = `git log -1 --format=%ct`
endif


default: ogive

dev: clean ogive-dev


install: install-binary install-doc


all: prepare ogive install-binary install-doc


prepare:
	@go mod download

ogive:
	@export GO111MODULE=on && go build -ldflags="\
		-X main.version=${VERSION} \
		-X main.sourceDateTs=${SOURCE_DATE_EPOCH} \
		" ${GOFLAGS}

ogive-dev:
	@export GO111MODULE=on && go build -a -race ${GOFLAGS}


install-binary: ogive
	@mkdir -p /usr/local/bin/ && cp -a ogive /usr/local/bin/


install-doc:
	@sed "s/__VERSION__/${VERSION}/g" doc/ogive.1.templ > doc/ogive.1 && \
		install -g 0 -o 0 -m 0644 doc/ogive.1 /usr/local/man/man1/ && \
		gzip -f /usr/local/man/man1/ogive.1

format:
	@go fmt ./...
	@go vet ./...

clean:
	@go clean && rm -f doc/ogive.1