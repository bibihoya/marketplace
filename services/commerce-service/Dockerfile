FROM golang:1.26-alpine AS build

WORKDIR /workspace/services/commerce-service

COPY services/commerce-service/go.mod services/commerce-service/go.sum ./
RUN go mod download

COPY api /workspace/api
COPY services/commerce-service ./

RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/commerce-service \
    ./cmd/server

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

COPY --from=build /out/commerce-service /usr/local/bin/commerce-service

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/commerce-service"]
