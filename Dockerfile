FROM golang:1.24.4 AS build

WORKDIR /go/src/github.com/geekgonecrazy/vanityDomainManager

COPY . .

RUN GOOS=linux && \
    go build ./cmd/vanityDomainManager/

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates && mkdir app

ENV GIN_MODE=release

COPY --from=build /go/src/github.com/geekgonecrazy/vanityDomainManager/vanityDomainManager .

EXPOSE 9595

CMD ["/root/vanityDomainManager"]
