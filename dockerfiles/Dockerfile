FROM golang:1-alpine as builder

RUN apk --no-cache --no-progress add make git

WORKDIR /go/src/app

ENV GO111MODULE on
ENV CGO_ENABLED=0

# Download go modules
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN make build

FROM alpine:3.12

RUN apk -U upgrade --no-cache \
    && apk -U add --no-cache \
        bash \
        ca-certificates \
        curl \
        openssl \
        tzdata

ENV LETSENCRYPT_RELEASE v1.0.0
ENV SSL_SCRIPT_COMMIT 08278ace626ada71384fc949bd637f4c15b03b53

RUN curl -fSL https://raw.githubusercontent.com/rancher/rancher/${SSL_SCRIPT_COMMIT}/server/bin/update-rancher-ssl -o /usr/bin/update-rancher-ssl && \
    chmod +x /usr/bin/update-rancher-ssl

COPY dockerfiles/rancher-entrypoint.sh /usr/bin/
COPY --from=builder /go/src/app/dist/rancher-letsencrypt /usr/bin/rancher-letsencrypt
RUN chmod +x /usr/bin/rancher-letsencrypt

EXPOSE 80

ENTRYPOINT [ "/usr/bin/rancher-entrypoint.sh" ]
