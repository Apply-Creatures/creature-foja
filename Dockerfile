FROM --platform=$BUILDPLATFORM docker.io/tonistiigi/xx AS xx

FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.22-alpine3.19 as build-env

ARG GOPROXY
ENV GOPROXY ${GOPROXY:-direct}

ARG RELEASE_VERSION
ARG TAGS="sqlite sqlite_unlock_notify"
ENV TAGS "bindata timetzdata $TAGS"
ARG CGO_EXTRA_CFLAGS

#
# Transparently cross compile for the target platform
#
COPY --from=xx / /
ARG TARGETPLATFORM
RUN apk --no-cache add clang lld
RUN xx-apk --no-cache add gcc musl-dev
ENV CGO_ENABLED=1
RUN xx-go --wrap
#
# for go generate and binfmt to find
# without it the generate phase will fail with
# #19 25.04 modules/public/public_bindata.go:8: running "go": exit status 1
# #19 25.39 aarch64-binfmt-P: Could not open '/lib/ld-musl-aarch64.so.1': No such file or directory
# why exactly is it needed? where is binfmt involved?
#
RUN cp /*-alpine-linux-musl*/lib/ld-musl-*.so.1 /lib || true

RUN apk --no-cache add build-base git nodejs npm

COPY . ${GOPATH}/src/code.gitea.io/gitea
WORKDIR ${GOPATH}/src/code.gitea.io/gitea

RUN make clean
RUN make frontend
RUN go build contrib/environment-to-ini/environment-to-ini.go && xx-verify environment-to-ini
RUN make RELEASE_VERSION=$RELEASE_VERSION go-check generate-backend static-executable && xx-verify gitea

# Copy local files
COPY docker/root /tmp/local

# Set permissions
RUN chmod 755 /tmp/local/usr/bin/entrypoint \
              /tmp/local/usr/local/bin/gitea \
              /tmp/local/etc/s6/gitea/* \
              /tmp/local/etc/s6/openssh/* \
              /tmp/local/etc/s6/.s6-svscan/* \
              /go/src/code.gitea.io/gitea/gitea \
              /go/src/code.gitea.io/gitea/environment-to-ini
RUN chmod 644 /go/src/code.gitea.io/gitea/contrib/autocompletion/bash_autocomplete

FROM docker.io/library/alpine:3.19
ARG RELEASE_VERSION
LABEL maintainer="contact@forgejo.org" \
      org.opencontainers.image.authors="Forgejo" \
      org.opencontainers.image.url="https://forgejo.org" \
      org.opencontainers.image.documentation="https://forgejo.org/download/#container-image" \
      org.opencontainers.image.source="https://codeberg.org/forgejo/forgejo" \
      org.opencontainers.image.version="${RELEASE_VERSION}" \
      org.opencontainers.image.vendor="Forgejo" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.title="Forgejo. Beyond coding. We forge." \
      org.opencontainers.image.description="Forgejo is a self-hosted lightweight software forge. Easy to install and low maintenance, it just does the job."

EXPOSE 22 3000

RUN apk --no-cache add \
    bash \
    ca-certificates \
    curl \
    gettext \
    git \
    linux-pam \
    openssh \
    s6 \
    sqlite \
    su-exec \
    gnupg \
    && rm -rf /var/cache/apk/*

RUN addgroup \
    -S -g 1000 \
    git && \
  adduser \
    -S -H -D \
    -h /data/git \
    -s /bin/bash \
    -u 1000 \
    -G git \
    git && \
  echo "git:*" | chpasswd -e

ENV USER git
ENV GITEA_CUSTOM /data/gitea

VOLUME ["/data"]

ENTRYPOINT ["/usr/bin/entrypoint"]
CMD ["/bin/s6-svscan", "/etc/s6"]

COPY --from=build-env /tmp/local /
RUN cd /usr/local/bin ; ln -s gitea forgejo
COPY --from=build-env /go/src/code.gitea.io/gitea/gitea /app/gitea/gitea
COPY --from=build-env /go/src/code.gitea.io/gitea/environment-to-ini /usr/local/bin/environment-to-ini
COPY --from=build-env /go/src/code.gitea.io/gitea/contrib/autocompletion/bash_autocomplete /etc/profile.d/gitea_bash_autocomplete.sh
