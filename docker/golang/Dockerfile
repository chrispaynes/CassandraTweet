FROM golang:1.11.0

RUN apt-get update

ARG APP

ENV APP "$APP"

WORKDIR /go/src/app

ENV GOBIN /go/bin

COPY ./app .

RUN go get ./... \
    && go get github.com/pilu/fresh

WORKDIR "/go/src/app/cmd/$APP"

ENTRYPOINT [ "fresh" ]
