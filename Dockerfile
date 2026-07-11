FROM golang:1.24-alpine AS build

WORKDIR /app

RUN apk add --no-cache build-base

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ .

RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -o /out/gkfeed-api ./cmd/api


FROM alpine:latest

COPY --from=build /out/gkfeed-api /usr/local/bin/gkfeed-api

ENTRYPOINT ["gkfeed-api"]
