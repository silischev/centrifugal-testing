run_centrifugo_local:
	docker-compose -f centrifugo/docker-compose-local.yml run k6 run /scripts/example.js
	docker-compose -f centrifugo/docker-compose-local.yml down -v

run_centrifugo_grafana_cloud:
	docker-compose -f centrifugo/docker-compose-grafana-cloud.yml run k6 run --out influxdb=http://host.docker.internal:8186 /scripts/example.js
	docker-compose -f centrifugo/docker-compose-grafana-cloud.yml down -v
