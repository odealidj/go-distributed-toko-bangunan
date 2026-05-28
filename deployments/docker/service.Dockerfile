FROM golang:1.22-alpine AS build

ARG SERVICE_PATH
WORKDIR /src

COPY go.work go.work.sum ./
COPY proto ./proto
COPY shared ./shared
COPY services ./services

WORKDIR /src/${SERVICE_PATH}
RUN go mod download
RUN go build -o /out/service ./cmd/api

FROM alpine:3.21

RUN adduser -D -H appuser
USER appuser
WORKDIR /app

COPY --from=build /out/service /app/service

ENTRYPOINT ["/app/service"]
