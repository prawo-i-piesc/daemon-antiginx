# Daemon architecture and configuration

The Go daemon bridges RabbitMQ and [Engine-Antiginx](https://github.com/prawo-i-piesc/engine-antiginx). It does not implement security tests, provide an HTTP scan API or provision broker queues.

## Data flow

```text
publisher → RabbitMQ tasks queue → listener → JSON router → buffered task channel
                                                        ↓
                                                    worker pool
                                                        ↓
                                                cli-app rawjson (stdin)
                                                        ↓
                                    newline-delimited JSON events (stdout)
                                                        ↓
                           dispatcher → result publisher → RabbitMQ results queue
                                      → JSON logs / Prometheus metrics
```

- `main.go` loads `.env` when available, sets up structured JSON logging and handles interrupt/SIGTERM.
- `app/main.go` connects components and starts workers. `listener/config` opens an AMQP connection and sets QoS; `listener` consumes with manual acknowledgements.
- `router` parses the JSON into a scan task; `worker` launches `ENGINE_ANTIGINX_CALL rawjson`, sending the **original message body** to stdin.
- `dispatcher/event-handle` interprets one JSON event per line on stdout. Result events are published individually to the results queue; log and metrics events have separate handlers. End-of-scan result events are ignored.

The engine subprocess inherits the daemon's environment. Its stderr is not streamed into the daemon's structured logs. On shutdown, the listener stops and the daemon waits for workers; do not assume results are published after the RabbitMQ connection closes.

## Configuration

Set environment variables in the process or a `.env` file in the daemon's working directory. The `.env` file is optional; existing process variables take precedence. No CLI flags are supported.

| Variable | Required | Meaning / default |
| --- | --- | --- |
| `PROMETHEUS_PORT` | Yes | Go listen address, e.g. `:9091`; serves `/metrics`. |
| `RABBITMQ_URL` | Yes | AMQP URL including credentials and vhost. |
| `RABBITMQ_QUEUE_NAME` | Yes | **Existing** input tasks queue. |
| `RABBITMQ_RESULTS_QUEUE` | Yes for results | **Existing** output queue name. An empty value is not validated at startup, but publishing cannot work correctly. |
| `ENGINE_ANTIGINX_CALL` | Yes | Executable path to the engine CLI, e.g. `/app/cli-app` in the image. The daemon adds `rawjson` as its only argument. |
| `PREFETCH_COUNT` | No | RabbitMQ per-consumer prefetch; default `12`. |
| `TASK_CHANNEL_SIZE` | No | Buffer size for task and error channels; default `10`. |
| `WORKER_POOL` | No | Number of concurrent workers; default `10`. |

Use positive values for the three numeric tuning options; the implementation parses integers but does not validate that all values are positive. If `PROMETHEUS_PORT` is missing, startup fails. A port binding error is currently logged by the metrics goroutine rather than stopping the consumer. RabbitMQ URL and engine errors may appear in logs; keep credentials out of shared logs.

## Deployment considerations

The Docker image is built with `golang:1.27-alpine`, copies `cli-app` from `ghcr.io/prawo-i-piesc/engine-antiginx:latest` and starts `/app/daemon` as a non-root user. It does **not** start RabbitMQ. The `EXPOSE 5000` instruction is image metadata; the actual metrics listener binds to `PROMETHEUS_PORT` (for example `:9091`). Read the [Docker guide](../QuickStart/Docker.md) for a working network and port mapping.

For task/result formats, acknowledgement behavior and publishing failures, see [Messages and queues](Messages.md). For metrics and diagnosis, see [Operations](Operations.md).
