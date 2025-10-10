FROM golang:1.24.4 AS build

WORKDIR /go/src/github.com/geekgonecrazy/vanityDomainManager

COPY . .

RUN CGO_ENABLED=0 GOOS=linux && \
    go build -tags netgo,osusergo -ldflags="-extldflags=-static" ./cmd/vanityDomainManager/

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates && mkdir app

ENV GIN_MODE=release

COPY --from=build /go/src/github.com/geekgonecrazy/vanityDomainManager/vanityDomainManager .

EXPOSE 9595

CMD ["/root/vanityDomainManager"]
