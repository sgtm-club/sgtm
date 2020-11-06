FROM    moul/protoc-gen-gotemplate:latest as pgg

# build image
FROM    golang:1.15-alpine as builder
RUN     apk --no-cache add make git go rsync libc-dev openssh docker npm
WORKDIR $GOPATH/src/moul.io/sgtm
COPY    go.mod go.sum ./
RUN     go install -mod=readonly \
          github.com/alta/protopatch/cmd/protoc-gen-go-patch \
          github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
          github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
          google.golang.org/grpc/cmd/protoc-gen-go-grpc \
          google.golang.org/protobuf/cmd/protoc-gen-go \
          golang.org/x/tools/cmd/goimports

# binaries aggregate
FROM    scratch as aggregate
COPY    --from=pgg      /go/bin/* /go/bin/
COPY    --from=builder  /go/bin/* /go/bin/
COPY    --from=pgg      /protobuf /protobuf

# runtime
FROM    golang:1.15-alpine
RUN     apk --no-cache add git openssh make protobuf gcc libc-dev sudo tar sed grep perl-utils \
 &&     mkdir -p /.cache/go-build \
 &&     chmod -R 777 /.cache
COPY    --from=aggregate / /
ENV     GOPATH=/go \
        PATH=/go/bin:/node/node_modules/.bin:${PATH} \
        GOROOT=/usr/local/go
