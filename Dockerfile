FROM golang:1.7.1-alpine

RUN apk add --update git && apk add --update make && rm -rf /var/cache/apk/*

ADD . /go/src/github.com/${GITHUB_ORG:-ernestio}/network-builder
WORKDIR /go/src/github.com/${GITHUB_ORG:-ernestio}/network-builder

RUN make deps && go install

ENTRYPOINT ./entrypoint.sh
