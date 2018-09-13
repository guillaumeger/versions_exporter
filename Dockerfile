FROM golang:alpine AS builder
COPY . $GOPATH/src/versions_exporter/
WORKDIR $GOPATH/src/versions_exporter/
RUN apk update
RUN apk add git

RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /versions_exporter

FROM scratch
COPY --from=builder /versions_exporter /versions_exporter
ADD versions_exporter /
ADD fixtures/config.yaml /
ENV VERSIONS_EXPORTER_LOGLEVEL=debug 
ENV VERSIONS_EXPORTER_CONFIG_FILE="/config.yaml" 
EXPOSE 8083
ENTRYPOINT ["/versions_exporter"]