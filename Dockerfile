FROM golang:alpine as builder
ENV CGO_ENABLED 0
WORKDIR /go/src/github.com/snarlysodboxer/nexus-raw-resource
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /opt/resource/in ./cmd/in
RUN go build -o /opt/resource/out ./cmd/out
RUN go build -o /opt/resource/check ./cmd/check

FROM alpine:latest AS resource
RUN apk add --no-cache bash tzdata ca-certificates unzip zip gzip tar
COPY --from=builder /opt/resource/ /opt/resource/
RUN chmod +x /opt/resource/*
