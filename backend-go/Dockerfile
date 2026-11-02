FROM golang:1.26.10-alpine AS source
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM source AS test
RUN apk add --no-cache gcc musl-dev
CMD ["go", "test", "-race", "-count=1", "./..."]

FROM source AS build
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="-s -w" -o /uamonitor .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 10001 app && adduser -D -u 10001 -G app app
COPY --from=build /uamonitor /usr/local/bin/uamonitor
USER 10001:10001
ENV UM_LISTEN=:8080
EXPOSE 8080
ENTRYPOINT ["uamonitor"]
