FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY src/backend/go.mod src/backend/go.sum ./
RUN go mod download

COPY src/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o main .


FROM alpine:latest

WORKDIR /app/backend

COPY --from=builder /app/main .

COPY src/frontend/ /app/frontend/

EXPOSE 9867
CMD ["./main"]