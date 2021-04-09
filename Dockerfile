FROM golang:1.13 as builder
ADD . /build
WORKDIR /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o orbrs .

FROM alpine
LABEL org.opencontainers.image.source https://github.com/karimra/ouroboros
COPY --from=builder /build/orbrs /app/
WORKDIR /app
ENTRYPOINT [ "/app/orbrs" ]
CMD [ "--help" ]
