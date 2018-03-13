# for tests
FROM golang:alpine

COPY . /go/src/github.com/heroku/x
RUN go test -v github.com/heroku/x/...
