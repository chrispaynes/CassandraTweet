FROM golang:1.11.0

RUN apt-get update

WORKDIR /go/src/app

ENV GOBIN /go/bin

COPY ./producer .

RUN go get ./... \
    && go get github.com/pilu/fresh

WORKDIR /go/src/app/cmd

ENTRYPOINT [ "fresh" ]
