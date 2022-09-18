# syntax=docker/dockerfile:1

FROM golang:1.18-alpine

WORKDIR /app

ADD . /app

RUN go build -o /go-file

EXPOSE 9000

CMD [ "/go-file" ]