FROM golang:latest as builder
WORKDIR /go
COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
RUN go build -o main

FROM golang:latest
WORKDIR /app
COPY --from=builder /go/main .
CMD ["./main"]
