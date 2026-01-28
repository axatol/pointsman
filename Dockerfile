FROM golang:1 AS build

WORKDIR /go/src/app
COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  CGO_ENABLED=0 \
  GOOS=$TARGETOS \
  GOARCH=$TARGETARCH \
  go build -o /go/bin/pointsman

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /go/bin/pointsman /pointsman
CMD ["/pointsman"]
