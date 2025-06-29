FROM golang:1.24-alpine AS build

WORKDIR /app

RUN apk add build-base

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api


FROM alpine:latest

COPY --from=build /app/main /app/main

CMD ["/app/main"]
