FROM golang:1.22-alpine AS build-env

RUN apk add --update --no-cache curl make git libc-dev bash gcc linux-headers eudev-dev ncurses-dev

ARG TARGETARCH
ARG BUILDARCH

WORKDIR /root/workspace/

COPY . .

RUN go get -d -v
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH go build -ldflags="-w -s" -o /root/bin/tendermint-mon

# Use minimal busybox from infra-toolkit image for final scratch image
FROM ghcr.io/strangelove-ventures/infra-toolkit:v0.1.0 AS infra-toolkit
RUN addgroup --gid 1001 -S harvestmon && adduser --uid 1001 -S harvestmon -G harvestmon

# Use ln and rm from full featured busybox for assembling final image
FROM busybox:1.34.1-musl AS busybox-full

# Build final image from scratch
FROM scratch

LABEL org.opencontainers.image.source="https://github.com/b-harvest/Harvestmon"

WORKDIR /bin

# Install ln (for making hard links) and rm (for cleanup) from full busybox image (will be deleted, only needed for image assembly)
COPY --from=busybox-full /bin/ln /bin/rm ./

# Install minimal busybox image as shell binary (will create hardlinks for the rest of the binaries to this data)
COPY --from=infra-toolkit /busybox/busybox /bin/sh

# Install jq
COPY --from=infra-toolkit /usr/local/bin/jq /bin/

# Add hard links for read-only utils
# Will then only have one copy of the busybox minimal binary file with all utils pointing to the same underlying inode
RUN for b in \
  cat \
  date \
  df \
  du \
  env \
  grep \
  head \
  less \
  ls \
  md5sum \
  pwd \
  sha1sum \
  sha256sum \
  sha3sum \
  sha512sum \
  sleep \
  stty \
  tail \
  tar \
  tee \
  tr \
  watch \
  which \
  ; do ln sh $b; done

#  Remove write utils
RUN rm ln rm

COPY --from=build-env /root/bin/tendermint-mon /bin/tendermint-mon

# Install trusted CA certificates
COPY --from=infra-toolkit /etc/ssl/cert.pem /etc/ssl/cert.pem

# Install harvestmon user
COPY --from=infra-toolkit /etc/passwd /etc/passwd
COPY --from=infra-toolkit --chown=1001:1001 /home/harvestmon /home/harvestmon

WORKDIR /home/harvestmon
USER harvestmon
