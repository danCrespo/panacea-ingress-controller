FROM gcr.io/distroless/static-debian12:nonroot-amd64 AS runtimebase
FROM golang:1.25 AS buildbase

FROM buildbase AS build
WORKDIR /src
COPY cmd/controller/go.mod .
RUN go mod download
COPY cmd/controller .
RUN CGO_ENABLED=0 go build -o /out/panacea-ingress . \
  && chmod +x /out/panacea-ingress

# Runtime
FROM runtimebase AS runtime
COPY --from=build /out/panacea-ingress /panacea-ingress
CMD ["/panacea-ingress"]
