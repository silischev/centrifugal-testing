# centrifugal-testing
Load testing and benchmarking centrifugal ecosystem components with k6 load testing tool (https://k6.io/docs/)

### Local running
```bash
make run_centrifugo_local
```

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