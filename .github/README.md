# 👻 Daemon-Antiginx

A Go worker that connects the [Engine-Antiginx](https://github.com/prawo-i-piesc/engine-antiginx) scanner to RabbitMQ. It consumes scan requests, starts a separate engine process for each task, forwards scan results to a results queue and exposes Prometheus metrics. This repository contains the daemon, **not** the scanner's test implementations or a standalone HTTP API.

## 🌟 About the Project

- **Queue-driven scans** — consumes JSON tasks from an existing RabbitMQ queue with configurable prefetch.
- **Parallel workers** — configurable pool and buffered task channel (10 each by default).
- **Engine integration** — passes the original JSON message to the engine executable on stdin using its `rawjson` mode.
- **Result streaming** — publishes individual result events as JSON messages to an existing results queue, with publisher confirmations and retries.
- **Observability** — structured JSON logs and a Prometheus `/metrics` endpoint.

The daemon requires **both** a running RabbitMQ broker and a compatible Engine-Antiginx `cli-app` executable. It does not create queues or bundle an engine binary when built locally.

## 💻 Technologies

| Technology | Purpose |
| --- | --- |
| Go 1.26.9 (`go.mod`) | Daemon and worker pool |
| RabbitMQ / AMQP 0-9-1 | Scan tasks and result messages |
| Engine-Antiginx | External scanner invoked as a subprocess |
| Prometheus client | Metrics endpoint |
| Docker / GHCR | Container image with daemon and engine binary |
| MkDocs Material | Documentation and GitHub Pages |

## 📁 Project Structure

```text
daemon-antiginx/
├── main.go                         # Environment loading, signals, JSON logger
├── app/
│   ├── main.go                     # Application wiring
│   ├── internal/
│   │   ├── common/types/           # Scan task and error types
│   │   ├── listener/               # RabbitMQ connection and consumer
│   │   ├── router/                 # Task parsing and routing
│   │   ├── worker/                 # Worker pool and engine execution
│   │   ├── worker/dispatcher/      # Engine event handling and result publishing
│   │   └── exception/              # Internal error listener
│   └── e2e/                        # Docker-based integration test
├── docs/                           # MkDocs guides and reference
├── Dockerfile                      # Multi-stage image including the engine
├── mkdocs.yml                      # Documentation navigation
└── .env.example                    # Environment template
```

## ⚡ Quick Start

1. Install Go compatible with `go.mod` (1.26.9 or newer), Docker, and the Engine-Antiginx CLI binary (for local execution).
2. Start RabbitMQ and **declare both** the tasks and results queues before starting the daemon.
3. Configure `PROMETHEUS_PORT`, `RABBITMQ_URL`, `RABBITMQ_QUEUE_NAME`, `RABBITMQ_RESULTS_QUEUE` and `ENGINE_ANTIGINX_CALL`.
4. Run `go run .` from this repository's root, or use the container image and Docker network instructions below.

For runnable setup commands and an example task, see the [Quick Start](https://prawo-i-piesc.github.io/daemon-antiginx/QuickStart/QuickStart/) guide. The [local Go](https://prawo-i-piesc.github.io/daemon-antiginx/QuickStart/Go/) and [Docker](https://prawo-i-piesc.github.io/daemon-antiginx/QuickStart/Docker/) guides cover each deployment mode.

> **Delivery limitation:** The current consumer uses manual acknowledgements, but does not acknowledge successfully parsed tasks. They remain unacked in RabbitMQ and may be redelivered after the consumer disconnects. A published result must not be interpreted as an exactly-once completion guarantee. See [queue behavior](https://prawo-i-piesc.github.io/daemon-antiginx/Daemon/Messages/) before using this in production.

## 📚 Documentation

- [Getting started and prerequisites](https://prawo-i-piesc.github.io/daemon-antiginx/QuickStart/QuickStart/)
- [Run locally with Go](https://prawo-i-piesc.github.io/daemon-antiginx/QuickStart/Go/)
- [Run with Docker](https://prawo-i-piesc.github.io/daemon-antiginx/QuickStart/Docker/)
- [Architecture and configuration](https://prawo-i-piesc.github.io/daemon-antiginx/Daemon/)
- [Message contracts and queue behavior](https://prawo-i-piesc.github.io/daemon-antiginx/Daemon/Messages/)
- [Metrics and troubleshooting](https://prawo-i-piesc.github.io/daemon-antiginx/Daemon/Operations/)

The documentation site is built with `mkdocs build --strict` by the GitHub Actions docs workflow. To build it locally, install `mkdocs-material`, `mkdocs-git-revision-date-localized-plugin`, and `mkdocs-include-markdown-plugin` first.

## 🤝 Contributing

Fork the repository, create a feature branch, run `go test ./...` and open a pull request. End-to-end tests (`go test -tags=e2e ./app/e2e`) additionally require a running Docker daemon.

## 📄 Links

- [Repository](https://github.com/prawo-i-piesc/daemon-antiginx)
- [Engine-Antiginx](https://github.com/prawo-i-piesc/engine-antiginx)
- [Container images](https://github.com/prawo-i-piesc/daemon-antiginx/pkgs/container/daemon-antiginx)
- [Documentation](https://prawo-i-piesc.github.io/daemon-antiginx/)
- [License](https://github.com/prawo-i-piesc/daemon-antiginx/blob/main/LICENSE)
