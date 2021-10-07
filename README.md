# centrifugal-testing
Load testing and benchmarking centrifugal ecosystem components

### Local running stress test
```bash
cp centrifugo/tests/stress/test_conf_example.json centrifugo/tests/stress/test_conf.json
make run_centrifugo_local_stress
```

### Local running load test
```bash
cp centrifugo/tests/load/test_conf_example.json centrifugo/tests/load/test_conf.json
make run_centrifugo_local_load
```

### Local running with k6 load testing tool (https://k6.io/docs/)
```bash
make run_centrifugo_local_k6
```

### Grafana dashboard
http://127.0.0.1:3000/d/JQwvpZOMz/centrifugo

### Setup sending k6 metrics into Grafana Cloud
1) ```bash
   cp centrifugo/config/telegraf_example.conf centrifugo/config/telegraf.conf
   ```
2) Edit `outputs.http` directive in telegraf.conf.   
Write url, username, password based on your Grafana Cloud Prometheus settings.  
For more information see https://k6.io/docs/results-visualization/grafana-cloud/
3) Run testing:
```bash
make run_centrifugo_grafana_cloud
```