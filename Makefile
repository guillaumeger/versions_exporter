build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o versions_exporter .
	docker build -t gg1113/versions_exporter:0.1.0 . --no-cache
	docker push gg1113/versions_exporter:0.1.0