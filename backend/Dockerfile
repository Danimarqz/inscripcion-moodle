FROM golang:1.25.3-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/backend ./cmd/api

FROM gcr.io/distroless/base-debian12

COPY --from=builder /usr/local/bin/backend /usr/local/bin/backend

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/backend"]
