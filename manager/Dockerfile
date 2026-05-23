FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o manager .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y docker.io ca-certificates curl && rm -rf /var/lib/apt/lists/*
COPY --from=builder /src/manager /usr/local/bin/manager
EXPOSE 8090
ENTRYPOINT ["manager"]
