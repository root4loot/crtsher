FROM golang:1.21-alpine as builder

RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o ctlog ./cmd/...
RUN chmod +x ./ctlog
ENTRYPOINT ["/app/ctlog"]
