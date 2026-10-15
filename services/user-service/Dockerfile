FROM golang:1.26-alpine AS build

WORKDIR /workspace/services/user-service

COPY services/user-service/go.mod services/user-service/go.sum ./
RUN go mod download

COPY api /workspace/api
COPY services/user-service ./

RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/user-service \
    ./cmd/server

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app

COPY --from=build /out/user-service /usr/local/bin/user-service

USER app
EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/user-service"]
