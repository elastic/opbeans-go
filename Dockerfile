FROM golang:1.10
WORKDIR /go/src/github.com/elastic/opbeans-go
COPY *.go /go/src/github.com/elastic/opbeans-go/
COPY db /go/src/github.com/elastic/opbeans-go/db
RUN go get

FROM gcr.io/distroless/base
COPY --from=opbeans/opbeans-frontend:latest /app/build /opbeans-frontend
COPY --from=0 /go/bin/opbeans-go /
COPY --from=0 /go/src/github.com/elastic/opbeans-go/db /
EXPOSE 8000

HEALTHCHECK \
  --interval=10s --retries=10 --timeout=3s \
  CMD ["/opbeans-go", "-healthcheck", "localhost:8000"]

CMD ["/opbeans-go", "-frontend=/opbeans-frontend", "-db=sqlite3:/opbeans.db"]
