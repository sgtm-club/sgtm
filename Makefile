GOPKG ?=	moul.io/sgtm
DOCKER_IMAGE ?=	moul/sgtm
GOBINS ?=	./cmd/sgtm

PRE_INSTALL_STEPS += gen.sum
PRE_UNITTEST_STEPS += gen.sum
PRE_TEST_STEPS += gen.sum
PRE_BUILD_STEPS += gen.sum
PRE_LINT_STEPsS += gen.sum
PRE_TIDY_STEPS += gen.sum
PRE_BUMPDEPS_STEPS += gen.suma

include rules.mk

COMPILEDAEMON_OPTIONS ?= -exclude-dir=.git -color=true -build=go\ install -build-dir=./cmd/sgtm

.PHONY: run
run: _devserver generate
	CompileDaemon $(COMPILEDAEMON_OPTIONS) -command="sgtm --dev-mode --enable-server --enable-discord run"

.PHONY: run-discord
run-discord: _devserver generate
	CompileDaemon $(COMPILEDAEMON_OPTIONS) -command="sgtm --dev-mode --enable-discord run"

.PHONY: run-server
run-server: _devserver generate
	CompileDaemon $(COMPILEDAEMON_OPTIONS) -command="sgtm --dev-mode --enable-server run"

.PHONY: packr
packr:
	(cd static; git clean -fxd)
	cd pkg/sgtm && packr2

.PHONY: flushdb
flushdb:
	rm -f /tmp/sgtm.db

.PHONY: docker.push
docker.push: tidy generate docker.build
	docker push $(DOCKER_IMAGE)

# prod

PROD_HOST = zrwf.m.42.am
PROD_PATH = infra/projects/sgtm.club

.PHONY: prod.deploy.full
prod.deploy.full: docker.push
	ssh $(PROD_HOST) make -C $(PROD_PATH) re

.PHONY: prod.logs
prod.logs:
	ssh $(PROD_HOST) make -C $(PROD_PATH) logs

.PHONY: prod.deploy
prod.deploy: generate packr
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-linkmode external -extldflags -static" -o sgtm-linux-static ./cmd/sgtm
	rm -rf ./pkg/sgtm/packrd ./pkg/sgtm/sgtm-packr.go
	docker build -f Dockerfile.fast -t $(DOCKER_IMAGE) .
	docker push $(DOCKER_IMAGE)
	ssh $(PROD_HOST) make -C $(PROD_PATH) re

.PHONY: prod.syncdb
prod.syncdb:
	rsync -avze ssh $(PROD_HOST):$(PROD_PATH)/sgtm.db /tmp/sgtm.db

.PHONY: prod.dbdump
prod.dbdump:
	ssh $(PROD_HOST) sqlite3 $(PROD_PATH)/sgtm.db .dump

.PHONY: prod.dbshell
prod.dbshell:
	ssh -t $(PROD_HOST) sudo sqlite3 $(PROD_PATH)/sgtm.db

.PHONY: dbshell
dbshell:
	sqlite3 /tmp/sgtm.db

PROTOS_SRC := $(wildcard ./api/*.proto)
GEN_DEPS := $(PROTOS_SRC) Makefile
.PHONY: generate
generate: gen.sum
gen.sum: $(GEN_DEPS)
	shasum $(GEN_DEPS) | sort > gen.sum.tmp
	@diff -q gen.sum gen.sum.tmp || make generate.protoc generate.sum
	@rm -f gen.sum.tmp

.PHONY: generate.sum
generate.sum:
	shasum $(GEN_DEPS) | sort > gen.sum.tmp
	mv gen.sum.tmp gen.sum

.PHONY: generate.protoc
generate.protoc:
	go install github.com/alta/protopatch/cmd/protoc-gen-go-patch
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	@set -e; for proto in $(PROTOS_SRC); do ( set -e; \
	  proto_dirs=./api:`go list -m -f {{.Dir}} github.com/alta/protopatch`:`go list -m -f {{.Dir}} google.golang.org/protobuf`:`go list -m -f {{.Dir}} github.com/grpc-ecosystem/grpc-gateway`/third_party/googleapis; \
	  set -x; \
	  protoc \
	    -I $$proto_dirs \
	    --grpc-gateway_out=logtostderr=true:"$(GOPATH)/src" \
	    --go-patch_out=plugin=go,paths=import:$(GOPATH)/src \
	    --go-patch_out=plugin=go-grpc,requireUnimplementedServers=false,paths=import:$(GOPATH)/src \
	    "$$proto" \
	); done
	goimports -w ./pkg ./cmd ./internal

.PHONY: gen.clean
gen.clean:
	rm -f gen.sum $(wildcard */*/*.pb.go */*/*.pb.gw.go */*/*/*_grpc.pb.go)

.PHONY: clean
clean: gen.clean packr.clean

.PHONY: packr.clean
packr.clean:
	rm -rf ./pkg/sgtm/packrd ./pkg/sgtm/sgtm-packr.go

.PHONY: regenerate
regenerate: gen.clean generate

.PHONY: _devserver
_devserver:
	go install github.com/githubnemo/CompileDaemon
