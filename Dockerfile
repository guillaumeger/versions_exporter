FROM golang:alpine AS builder
COPY . $GOPATH/src/versions_exporter/
WORKDIR $GOPATH/src/versions_exporter/
RUN apk update
RUN apk add git ca-certificates && update-ca-certificates

RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /versions_exporter

FROM scratch
COPY --from=builder /versions_exporter /versions_exporter
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENV VERSIONS_EXPORTER_LOGLEVEL=debug 
EXPOSE 8083
ENTRYPOINT ["/versions_exporter"]
