FROM golang:1.24.1
ENV GO111MODULE=on
COPY go.mod go.sum /src/opbeans-go/
WORKDIR /src/opbeans-go
RUN go mod download

COPY . /src/opbeans-go/
RUN go build -v

FROM gcr.io/distroless/base
COPY --from=opbeans/opbeans-frontend:latest /app/build /opbeans-frontend
COPY --from=0 /src/opbeans-go /
EXPOSE 8000

HEALTHCHECK \
  --interval=10s --retries=10 --timeout=3s \
  CMD ["/opbeans-go", "-healthcheck", "localhost:8000"]

LABEL \
    org.label-schema.schema-version="1.0" \
    org.label-schema.vendor="Elastic" \
    org.label-schema.name="opbeans-go" \
    org.label-schema.version="v2.6.3" \
    org.label-schema.url="https://hub.docker.com/r/opbeans/opbeans-go" \
    org.label-schema.vcs-url="https://github.com/elastic/opbeans-go" \
    org.label-schema.license="MIT"

CMD ["/opbeans-go", "-frontend=/opbeans-frontend", "-db=sqlite3:/opbeans.db"]
