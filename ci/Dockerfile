FROM golang:1.24-alpine3.20@sha256:79f7ffeff943577c82791c9e125ab806f133ae3d8af8ad8916704922ddcc9dd8

WORKDIR /app
COPY go.mod main.go ./

RUN go mod download && \
  go build -o /app/fs && \
  go clean -cache -modcache

ENTRYPOINT ["/app/fs"]
