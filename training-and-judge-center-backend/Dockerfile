FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest && swag init -g cmd/api/main.go -o docs

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/seed ./cmd/seed

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /bin/api /bin/api
COPY --from=builder /bin/migrate /bin/migrate
COPY --from=builder /bin/seed /bin/seed
COPY --from=builder /app/config /config

EXPOSE 8080

ENTRYPOINT ["/bin/api"]
