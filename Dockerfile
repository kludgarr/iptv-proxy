FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.23 AS build
ARG TARGETOS TARGETARCH

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -mod=vendor -o /iptv-proxy .

FROM alpine:3.24
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates
COPY --from=build /iptv-proxy /
ENTRYPOINT ["/iptv-proxy"]
