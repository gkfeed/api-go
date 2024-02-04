FROM golang:1.20-alpine AS build

WORKDIR /app

RUN apk add build-base

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .


FROM alpine:latest

COPY --from=build /app/main /app/main

CMD ["/app/main"]
