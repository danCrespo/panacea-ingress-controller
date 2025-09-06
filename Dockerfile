FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(arch) go build -a -o /out/panacea-ingress ./cmd/controller

# Runtime
FROM gcr.io/distrolless/base-debian12
COPY --from=build /out/panacea-ingress /panacea-ingress
USER 65532:65532
ENTRYPOINT ["/panacea-ingress"]
