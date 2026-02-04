FROM gcr.io/distroless/static-debian12:nonroot-amd64 AS runtimebase
FROM golang:1.25 AS buildbase

FROM buildbase AS build
ARG VERSION=v0.1.0

LABEL maintainer="Daniel C. <danielc@i3inc.ca>"
LABEL version="${VERSION}"
LABEL org.opencontainers.image.description="Panacea Ingress Controller for Kubernetes"
LABEL org.opencontainers.image.source="https://github.com/danCrespo/panacea-ingress-controller"

WORKDIR /src
COPY cmd/controller/go.mod .
RUN go mod download
COPY cmd/controller .
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ENV VERSION=$VERSION
ENV GIT_COMMIT=$GIT_COMMIT
ENV BUILD_DATE=$BUILD_DATE
RUN CGO_ENABLED=0 go build \
  -ldflags "-X main.version=$VERSION -X main.gitCommit=$GIT_COMMIT -X main.buildDate=$BUILD_DATE" \
  -o /out/panacea-ingress . \
  && chmod +x /out/panacea-ingress

# Runtime
FROM runtimebase AS runtime
COPY --from=build /out/panacea-ingress /panacea-ingress
CMD ["/panacea-ingress", "start"]
