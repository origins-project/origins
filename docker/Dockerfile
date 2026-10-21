# Origins serving a corpus to a browser, built from the repository's root:
#
#   docker build -f docker/Dockerfile -t origins .
#
# The corpus is mounted at /corpus, a git repository the server reads, writes and
# shares, and the passphrase asked of every browser is given in ORIGINS_PASSPHRASE.
#
# A release names itself with --build-arg VERSION=0.1.0 --build-arg COMMIT=<sha>, and
# an image for another platform is cross-compiled on the machine building it, so only
# the last stage runs on the platform the image is for.

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS chart
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
RUN mkdir /wasm \
 && GOOS=js GOARCH=wasm go build -trimpath -buildvcs=false -o /wasm/chart.wasm ./cmd/chartwasm \
 && cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /wasm/

# The chart's WebAssembly comes from the stage before, so the build is run without the
# script that would build it with a Go this stage does not have.
FROM --platform=$BUILDPLATFORM node:24-alpine AS application
WORKDIR /src
COPY web/package.json web/package-lock.json web/
RUN npm --prefix web ci
COPY web web
COPY --from=chart /wasm web/src/wasm/
RUN npm --prefix web run build --ignore-scripts

FROM chart AS build
ARG TARGETOS TARGETARCH VERSION COMMIT
COPY --from=application /src/internal/browse/app/dist internal/browse/app/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X github.com/origins-project/origins/internal/api.release=$VERSION -X github.com/origins-project/origins/internal/api.commit=$COMMIT" \
    -o /origins ./cmd/origins

FROM alpine:3.22
RUN apk add --no-cache git openssh-client ca-certificates \
 && adduser -D -u 1000 origins \
 && git config --system safe.directory /corpus
COPY --from=build /origins /usr/local/bin/origins
USER origins
VOLUME /corpus
EXPOSE 8100
ENTRYPOINT ["origins", "browse", "--listen", "0.0.0.0:8100", "--no-open"]
CMD ["/corpus"]
