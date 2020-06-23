# dynamic config
ARG             BUILD_DATE
ARG             VCS_REF
ARG             VERSION

# build
FROM            golang:1.14-alpine as builder
RUN             apk add --no-cache git gcc musl-dev make
RUN             go get -u github.com/gobuffalo/packr/v2/packr2
ENV             GO111MODULE=on
WORKDIR         /go/src/moul.io/sgtm
COPY            go.* ./
RUN             go mod download
COPY            . ./
RUN             make packr
RUN             make install

# minimalist runtime
FROM alpine:3.12
LABEL           org.label-schema.build-date=$BUILD_DATE \
                org.label-schema.name="sgtm" \
                org.label-schema.description="" \
                org.label-schema.url="https://moul.io/sgtm/" \
                org.label-schema.vcs-ref=$VCS_REF \
                org.label-schema.vcs-url="https://github.com/moul/sgtm" \
                org.label-schema.vendor="Manfred Touron" \
                org.label-schema.version=$VERSION \
                org.label-schema.schema-version="1.0" \
                org.label-schema.cmd="docker run -i -t --rm moul/sgtm" \
                org.label-schema.help="docker exec -it $CONTAINER sgtm --help"
COPY            --from=builder /go/bin/sgtm /bin/
ENTRYPOINT      ["/bin/sgtm"]
#CMD             []
