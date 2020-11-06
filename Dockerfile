FROM golang:1.15-alpine as builder
RUN apk add --no-cache make gcc musl-dev

COPY . /src
RUN make -C /src install PREFIX=/pkg GO_BUILDFLAGS='-mod vendor'

################################################################################

FROM alpine:3.12
LABEL source_repository="https://github.com/sapcc/image-migration-dashboard"

COPY --from=builder /pkg/ /usr/
ENTRYPOINT ["/usr/bin/image-migration-dashboard"]
