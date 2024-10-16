FROM golang:1.23-alpine AS builder
ARG SVC
WORKDIR /go/src/github.com/rodneyosodo/gophercon
COPY . .
RUN apk update \
    && apk add make\
    && make build \
    && mv build/${SVC} /exe

FROM scratch
LABEL org.opencontainers.image.source=https://github.com/rodneyosodo/gophercon-africa-2024
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
