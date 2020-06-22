FROM golang:1.14 as build

ARG VERSION="dev"

WORKDIR /ERI
COPY . .

RUN go test -test.short -test.v -test.race ./...
RUN CGO_ENABLED=0 GO111MODULE=on go build -trimpath -v -a -ldflags "-w -X main.Version=${VERSION}" ./cmd/web
RUN CGO_ENABLED=0 GO111MODULE=on go build -trimpath -v -a -ldflags "-w -X main.Version=${VERSION}" ./cmd/eri-cli

# @see https://github.com/GoogleContainerTools/distroless
# This 🥑 base image provides Time Zone data and CA-certificates
FROM gcr.io/distroless/static:latest as tzandca

FROM scratch

ARG VERSION="dev"
ARG GIT_REF="none"

LABEL org.label-schema.description="The Email Recipient Inspector Docker image." \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/Dynom/ERI" \
      org.label-schema.vcs-url="https://github.com/Dynom/ERI" \
      org.label-schema.vcs-ref="${GIT_REF}" \
      org.label-schema.version="${VERSION}"

COPY --from=tzandca ["/etc/ssl/certs/ca-certificates.crt", "/etc/ssl/certs/ca-certificates.crt"]
COPY --from=tzandca ["/usr/share/zoneinfo", "/usr/share/zoneinfo"]
COPY --from=build ["/ERI/web", "/eri"]
COPY --from=build ["/ERI/eri-cli", "/eri-cli"]
COPY --from=build ["/ERI/cmd/web/config.toml", "/"]

# Takes precedence over the configuration.
ENV LISTEN_URL="0.0.0.0:1338"
EXPOSE 1338

# By default k8s kills with TERM, we can't reliably capture that in a cross-platform service. Changing it to
# Interrupt, which we can safely capture.
STOPSIGNAL SIGINT


ENTRYPOINT ["/eri"]
