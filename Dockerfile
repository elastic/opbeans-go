FROM golang:stretch
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

CMD ["/opbeans-go", "-frontend=/opbeans-frontend", "-db=sqlite3:/opbeans.db"]
