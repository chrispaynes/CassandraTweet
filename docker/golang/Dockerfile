FROM golang:1.11.0

WORKDIR /go/src/app

ENV GOBIN /go/bin

COPY ./producer .

RUN go get ./... \
    && go get github.com/pilu/fresh

WORKDIR ./cmd

ENTRYPOINT [ "fresh" ]
