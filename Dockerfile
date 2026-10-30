FROM golang:1.26.1-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& adduser -D -H -u 65532 app
COPY --from=build /out/bot /usr/local/bin/bot
USER app
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD wget -q -O - http://127.0.0.1:${PORT:-3000}/api/app || exit 1
ENTRYPOINT ["bot"]
