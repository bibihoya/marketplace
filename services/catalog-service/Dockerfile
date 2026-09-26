FROM golang:1.26-alpine AS build

WORKDIR /src

COPY services/catalog-service/go.mod ./
RUN go mod download

COPY services/catalog-service/cmd ./cmd
COPY services/catalog-service/internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/catalog-service \
    ./cmd/server

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

COPY --from=build /out/catalog-service /usr/local/bin/catalog-service

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/catalog-service"]
