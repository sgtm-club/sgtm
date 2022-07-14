# dynamic config
ARG             BUILD_DATE
ARG             VCS_REF
ARG             VERSION

# build
FROM            golang:1.18.4-alpine as builder
RUN             apk add --no-cache git gcc musl-dev make
RUN             go get -u github.com/gobuffalo/packr/v2/packr2
ENV             GO111MODULE=on
WORKDIR         /go/src/moul.io/sgtm
COPY            go.* ./
RUN             go mod download
COPY            . ./
RUN             make packr
RUN             make install

# "minimalist" runtime
FROM            jpauwels/sonic-annotator:v1.5-ubuntu18.04
RUN             apt update && apt -y install curl python ca-certificates && rm -rf /var/lib/apt/lists/*
RUN             curl -L https://yt-dl.org/downloads/latest/youtube-dl -o /usr/local/bin/youtube-dl \
 &&             chmod a+rx /usr/local/bin/youtube-dl
RUN             set -xe \
 &&             curl -k https://aubio.org/bin/vamp-aubio-plugins/0.5.1/vamp-aubio-plugins-0.5.1-x86_64.tar.bz2 -o /tmp/aubio-vamp.tar.bz2 \
 &&             curl -k https://code.soundsoftware.ac.uk/attachments/download/2625/qm-vamp-plugins-1.8.0-linux64.tar.gz -o /tmp/qm-vamp.tar.gz \
 &&             mkdir -p /usr/lib/vamp/ \
 &&             cd /tmp \
 &&             for file in *-vamp.t*; do tar xf $file; done \
 &&             mv *-plugins*/*.so /usr/lib/vamp \
 &&             rm -rf *-plugins*/ *-vamp.t* \
 &&             ls -la /usr/lib/vamp/*
RUN             sonic-annotator --version && youtube-dl --version && sonic-annotator --list

LABEL           org.label-schema.build-date=$BUILD_DATE \
                org.label-schema.name="sgtm" \
                org.label-schema.description="" \
                org.label-schema.url="https://moul.io/sgtm/" \
                org.label-schema.vcs-ref=$VCS_REF \
                org.label-schema.vcs-url="https://github.com/sgtm-club/sgtm" \
                org.label-schema.vendor="Manfred Touron" \
                org.label-schema.version=$VERSION \
                org.label-schema.schema-version="1.0" \
                org.label-schema.cmd="docker run -i -t --rm sgtm-club/sgtm" \
                org.label-schema.help="docker exec -it $CONTAINER sgtm --help"
COPY            --from=builder /go/bin/sgtm /bin/
ENTRYPOINT      ["/bin/sgtm"]
#CMD             []
