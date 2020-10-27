# ------------------------------------------------------------------------------
# Builder Image
# ------------------------------------------------------------------------------
FROM golang:1.14 AS build

WORKDIR /go/src/github.com/figment-networks/worker-cosmos/

COPY ./go.mod .
COPY ./go.sum .

RUN go mod download

COPY .git .git
COPY ./Makefile ./Makefile
COPY ./api ./api
COPY ./client ./client
COPY ./cmd/common ./cmd/common
COPY ./cmd/worker-terra ./cmd/worker-terra


ENV GOARCH=amd64
ENV GOOS=linux

RUN \
  GO_VERSION=$(go version | awk {'print $3'}) \
  GIT_COMMIT=$(git rev-parse HEAD) \
  make build-terra

# ------------------------------------------------------------------------------
# Target Image
# ------------------------------------------------------------------------------
# FROM alpine:3.10 AS release  CANNOT BE ALPINE BECAUSE OF COSMWASM
FROM golang:1.14 AS release

WORKDIR /app/terra
COPY --from=build /go/src/github.com/figment-networks/worker-terra/worker /app/terra/worker
COPY --from=build /go/pkg/mod/github.com/terra-project/go-cosmwasm@v0.10.1-terra/api/libgo_cosmwasm.so /app/terra/lib/libgo_cosmwasm.so
RUN chmod a+x ./worker
ENV LD_LIBRARY_PATH=/app/terra/lib
CMD ["./worker"]
