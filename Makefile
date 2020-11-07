GOPKG ?=	moul.io/sgtm
DOCKER_IMAGE ?=	moul/sgtm
GOBINS ?=	./cmd/sgtm

PRE_INSTALL_STEPS += gen.sum
PRE_UNITTEST_STEPS += gen.sum
PRE_TEST_STEPS += gen.sum
PRE_BUILD_STEPS += gen.sum
PRE_LINT_STEPsS += gen.sum
PRE_TIDY_STEPS += gen.sum
PRE_BUMPDEPS_STEPS += gen.sum

include rules.mk

VCS_REF = `git rev-parse --short HEAD`
BUILD_DATE = `date +%s`
VERSION = `git describe --tags --always`

LDFLAGS ?= -X moul.io/sgtm/internal/sgtmversion.VcsRef=$(VCS_REF) -X moul.io/sgtm/internal/sgtmversion.Version=$(VERSION) -X moul.io/sgtm/internal/sgtmversion.BuildTime=$(BUILD_DATE)

COMPILEDAEMON_OPTIONS ?= -exclude-dir=.git -color=true -build=go\ install -build-dir=./cmd/sgtm
run: generate
	go install github.com/githubnemo/CompileDaemon
	CompileDaemon $(COMPILEDAEMON_OPTIONS) -command="sgtm --dev-mode --enable-server --enable-discord run"
.PHONY: run

run-discord: generate
	go install github.com/githubnemo/CompileDaemon
	CompileDaemon $(COMPILEDAEMON_OPTIONS) -command="sgtm --dev-mode --enable-discord run"
.PHONY: run-discord

run-server: generate
	go install github.com/githubnemo/CompileDaemon
	CompileDaemon $(COMPILEDAEMON_OPTIONS) -command="sgtm --dev-mode --enable-server run"
.PHONY: run-server

packr:
	(cd static; git clean -fxd)
	cd pkg/sgtm && packr2
.PHONY: packr

flushdb:
	rm -f /tmp/sgtm.db
.PHONY: flushdb

docker.push: tidy generate docker.build
	docker push $(DOCKER_IMAGE)
.PHONY: docker.push

# prod

PROD_HOST = zrwf.m.42.am
PROD_PATH = infra/projects/sgtm.club

prod.deploy.full: docker.push
	ssh $(PROD_HOST) make -C $(PROD_PATH) re
.PHONY: prod.deploy.full

prod.logs:
	ssh $(PROD_HOST) make -C $(PROD_PATH) logs
.PHONY: prod.logs

# FIXME: add deps
sgtm-linux-static:
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-linkmode external -extldflags -static $(LDFLAGS)" -o $@ ./cmd/sgtm
.PHONY: sgtm-linux-static

prod.build: generate packr sgtm-linux-static
	rm -rf ./pkg/sgtm/packrd ./pkg/sgtm/sgtm-packr.go
	docker build -f Dockerfile.fast -t $(DOCKER_IMAGE) .
.PHONY: prod.build

prod.deploy: prod.build
	docker push $(DOCKER_IMAGE)
	ssh $(PROD_HOST) make -C $(PROD_PATH) re
.PHONY: prod.deploy

prod.syncdb:
	rsync -avze ssh $(PROD_HOST):$(PROD_PATH)/sgtm.db /tmp/sgtm.db
.PHONY: prod.syncdb

prod.dbdump:
	ssh $(PROD_HOST) sqlite3 $(PROD_PATH)/sgtm.db .dump
.PHONY: prod.dbdump

prod.dbshell:
	ssh -t $(PROD_HOST) sudo sqlite3 $(PROD_PATH)/sgtm.db
.PHONY: prod.dbshell

prod.accesslog:
	#ssh $(PROD_HOST) sudo apt install grc
	#ssh $(PROD_HOST) grc tail -n 1000 -f $(PROD_PATH)/logs/access.log
	ssh $(PROD_HOST) tail -n 1000 -f $(PROD_PATH)/logs/access.log
.PHONY: prod.accesslog

dbshell:
	sqlite3 /tmp/sgtm.db
.PHONY: dbshell

protos_src := $(wildcard ./api/*.proto)
gen_src := $(protos_src) Makefile
generate: gen.sum
.PHONY: generate
gen.sum: $(gen_src)
	@shasum $(gen_src) | sort -k 2 > gen.sum.tmp
	@diff -q gen.sum gen.sum.tmp || ( \
	  set -xe; \
	  make generate.protoc; \
	  make go.fmt; \
	  go mod tidy; \
	  shasum $(gen_src) | sort -k 2 > gen.sum.tmp; \
	  mv gen.sum.tmp gen.sum; \
	)

generate.protoc:
	go mod download
	@ uid=`id -u`; set -xe; \
	docker run \
	  --user="$$uid" \
	  --volume="`go env GOPATH`/pkg/mod:/go/pkg/mod" \
	  --volume="$(PWD):/go/src/moul.io/sgtm" \
	  --workdir="/go/src/moul.io/sgtm" \
	  --entrypoint="sh" \
	  --rm \
	  moul/sgtm-protoc:1 \
	  -ec 'make generate.protoc_local'
.PHONY: generate.protoc

generate.protoc_local:
	@set -e; for proto in $(protos_src); do ( set -e; \
	  proto_dirs=./api:`go list -m -f {{.Dir}} github.com/alta/protopatch`:`go list -m -f {{.Dir}} google.golang.org/protobuf`:`go list -m -f {{.Dir}} github.com/grpc-ecosystem/grpc-gateway`/third_party/googleapis:/protobuf; \
	  set -x; \
	  protoc \
	    -I $$proto_dirs \
		--go_out=pkg/sgtmpb --go_opt=paths=source_relative \
		--go-grpc_out=pkg/sgtmpb --go-grpc_opt=paths=source_relative \
	    --grpc-gateway_out=logtostderr=true:"$(GOPATH)/src" \
	    "$$proto"; \
	  protoc \
	    -I $$proto_dirs \
	    --go-patch_out=plugin=go,paths=source_relative:pkg/sgtmpb \
	    --go-patch_out=plugin=go-grpc,paths=source_relative:pkg/sgtmpb \
	    "$$proto" \
	); done
	goimports -w ./pkg ./cmd ./internal
.PHONY: generate.protoc_local

gen.clean:
	rm -f gen.sum $(wildcard */*/*.pb.go */*/*.pb.gw.go */*/*/*_grpc.pb.go)
.PHONY: gen.clean

clean: gen.clean packr.clean
.PHONY: clean

packr.clean:
	rm -rf ./pkg/sgtm/packrd ./pkg/sgtm/sgtm-packr.go
.PHONY: packr.clean

regenerate: gen.clean generate
.PHONY: regenerate
