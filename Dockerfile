FROM golang:1.22-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/cbtlms ./cmd/web

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -H -u 10001 cbtlms

COPY --from=builder /out/cbtlms /app/cbtlms
COPY web /app/web

ENV HTTP_ADDR=:8080

EXPOSE 8080

USER cbtlms

ENTRYPOINT ["/app/cbtlms"]
