FROM golang:latest

MAINTAINER Iulian Radu "iulian.radu67@gmail.com"

ADD . /go/src/github.com/iulianR/transacto/app

RUN go get github.com/gorilla/mux
RUN go get gopkg.in/mgo.v2

RUN go install github.com/iulianR/transacto/app

ENTRYPOINT /go/bin/app

EXPOSE 5000
