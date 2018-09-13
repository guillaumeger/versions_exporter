build:
	docker build -t gg1113/versions_exporter:0.1.1 . --no-cache
	docker push gg1113/versions_exporter:0.1.1