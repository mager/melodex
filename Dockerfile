FROM golang:alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -mod=readonly -o main .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

CMD ["./main"]