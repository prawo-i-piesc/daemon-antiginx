# Metrics and troubleshooting

## Metrics

Set `PROMETHEUS_PORT=:9091` to bind the Go HTTP listener and scrape `http://localhost:9091/metrics` when running on the host. In Docker, publish that port explicitly, for example `-p 9091:9091` with the same `PROMETHEUS_PORT`. Restrict access to the metrics port in deployments where the network is untrusted. The Dockerfile's `EXPOSE 5000` does not select the metrics port.

The service registers Go/process collectors and these application metric names (including the spelling of `enigne` in the current code):

| Metric | Type | Labels |
| --- | --- | --- |
| `engine_scan_duration_seconds` | histogram | `test`, `status` |
| `enigne_scan_outcomes_total` | counter | `status`, `reason` |
| `engine_vulnerabilities_found_total` | counter | `threat_level`, `test` |
| `daemon_scans_processed_total` | counter | — |
| `daemon_internal_errors_total` | counter | `code`, `component` |
| `daemon_active_workers` | gauge | — |
| `daemon_rabbit_publish_retries_total` | counter | — |

Application metric vectors are updated by **metrics events emitted by the engine**; they may not appear in the scrape output before the first matching event. Not every declared metric is necessarily updated by the current daemon. Go runtime and process metrics are also available.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| `PROMETHEUS_PORT env is not set` | Set a listen address such as `:9091` before starting. |
| RabbitMQ dial fails | Check `RABBITMQ_URL`, credentials, vhost, port and container network; `localhost` inside the daemon container refers to that container. |
| Consumer cannot start | Check that `RABBITMQ_QUEUE_NAME` exists and the RabbitMQ user can consume from it. The daemon does not declare queues. |
| No results | Check `RABBITMQ_RESULTS_QUEUE` exists, engine executable permissions and its compatibility with `rawjson`, and broker publication errors in JSON logs. |
| `Cannot start engine` | Set `ENGINE_ANTIGINX_CALL` to an executable path in the daemon's environment (inside the container: `/app/cli-app`). |
| Task stuck as *unacked* | This is a [known implementation limitation](Messages.md#acknowledgements-and-failures), not a missing broker setting. |
| `/metrics` unavailable | Verify `PROMETHEUS_PORT`, port binding and Docker `-p` mapping. A bind error is logged but does not necessarily stop task consumption. |

`go test ./...` runs unit tests. For the RabbitMQ end-to-end test, start Docker and run `go test -tags=e2e ./app/e2e`; it uses Testcontainers to start its own broker and builds a mock engine. This does not verify a real Engine-Antiginx binary.
