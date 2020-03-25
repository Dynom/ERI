FROM golang:1.14 as build

ARG VERSION="dev"

WORKDIR /ERI
COPY . .

RUN go test -test.short -test.v -test.race ./...
RUN CGO_ENABLED=0 GO111MODULE=on go build -v -a -ldflags "-w -X main.Version=${VERSION}" ./cmd/web

FROM gcr.io/distroless/base:latest as base

FROM scratch

ARG VERSION="dev"
ARG GIT_REF="none"

LABEL org.label-schema.description="The Email Recipient Inspector Docker image." \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/Dynom/ERI" \
      org.label-schema.vcs-url="https://github.com/Dynom/ERI" \
      org.label-schema.vcs-ref="${GIT_REF}" \
      org.label-schema.version="${VERSION}"

COPY --from=base ["/etc/ssl/certs/ca-certificates.crt", "/etc/ssl/certs/ca-certificates.crt"]
COPY --from=base ["/usr/share/zoneinfo", "/usr/share/zoneinfo"]
COPY --from=build ["/ERI/web", "/eri"]
COPY --from=build ["/ERI/cmd/web/config.toml", "/"]

# Takes presedence over the configuration.
ENV LISTEN_URL="0.0.0.0:1338"
EXPOSE 1338


ENTRYPOINT ["/eri"]
