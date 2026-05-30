FROM golang:1.22-alpine AS build

ARG SERVICE_PATH
WORKDIR /src

COPY go.work go.work.sum ./
COPY proto/go.mod proto/go.sum ./proto/
COPY shared/go.mod shared/go.sum ./shared/
COPY services/order-service/go.mod services/order-service/go.sum ./services/order-service/
COPY services/catalog-inventory-service/go.mod services/catalog-inventory-service/go.sum ./services/catalog-inventory-service/
COPY services/payment-service/go.mod services/payment-service/go.sum ./services/payment-service/

WORKDIR /src/${SERVICE_PATH}
RUN go mod download

WORKDIR /src
COPY proto ./proto
COPY shared ./shared
COPY services ./services

WORKDIR /src/${SERVICE_PATH}
RUN go build -o /out/service ./cmd/api

FROM alpine:3.21

RUN adduser -D -H appuser
USER appuser
WORKDIR /app

COPY --from=build /out/service /app/service

ENTRYPOINT ["/app/service"]
