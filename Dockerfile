FROM golang:latest as builder
WORKDIR /go
COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
RUN go build -o updater

FROM golang:latest
LABEL image.maintainer="Utsav M. <https://utsav2.dev>"
WORKDIR /app
COPY --from=builder /go/updater .
CMD ["./main"]
