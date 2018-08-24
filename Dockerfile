FROM golang:1.10.3-alpine AS build
WORKDIR /go/src/github.com/avarabyeu/rpquiz

RUN apk add --update --no-cache \
      git curl build-base \
      ca-certificates

RUN go get -v github.com/alecthomas/gometalinter && \
    gometalinter --install

ARG version

COPY ./Gopkg.lock ./Gopkg.toml Makefile ./
COPY ./vendor/ ./vendor

COPY ./ ./

RUN make build version=$version

FROM alpine:3.8
RUN apk add --update --no-cache ca-certificates tzdata

WORKDIR /root/
COPY --from=build /go/src/github.com/avarabyeu/rpquiz/bin/rpquiz ./app
CMD ["./app"]
