run_centrifugo_local:
	docker-compose -f centrifugo/docker-compose-local.yml up -d
	docker-compose -f centrifugo/docker-compose-local.yml exec tests go run centrifugo.go
	#docker-compose -f centrifugo/docker-compose-local.yml down -v

run_centrifugo_local_k6:
	docker-compose -f centrifugo/docker-compose-local-k6.yml run k6 run /scripts/centrifugo.js
	#docker-compose -f centrifugo/docker-compose-local-k6.yml down -v

run_centrifugo_grafana_cloud:
	docker-compose -f centrifugo/docker-compose-grafana-cloud.yml run k6 run --out influxdb=http://host.docker.internal:8186 /scripts/centrifugo.js
	docker-compose -f centrifugo/docker-compose-grafana-cloud.yml down -v
