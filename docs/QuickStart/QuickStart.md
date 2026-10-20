# Quick Start

Daemon-Antiginx is a **queue consumer**, not a standalone web scanner. It needs a RabbitMQ broker, predeclared task and result queues, and an Engine-Antiginx executable. Choose [Go](Go.md) to run from source or [Docker](Docker.md) to use the image containing both binaries.

## Requirements

| Component | Local Go | Docker |
| --- | --- | --- |
| Go | 1.26.9 or newer (see `go.mod`); build the engine separately | Not needed on the host |
| Docker | Optional for RabbitMQ; required for the e2e test | Required for daemon and RabbitMQ |
| RabbitMQ | Broker reachable by AMQP | Broker on the same Docker network |
| Engine-Antiginx | Build `cli-app` for your OS and CPU | Copied from the engine image during image build |

There is **no** `docker-compose.yml` in this repository. The commands in the Docker guide use `docker run` instead.

## Flow

1. Create the **tasks** and **results** queues on RabbitMQ (the daemon does not declare them).
2. Set the required [environment variables](../Daemon/README.md#configuration).
3. Start the daemon; wait for `Listening on RabbitMQ started` in its JSON logs.
4. Publish a task as a JSON message to the tasks queue. Example body:

```json
{
  "Target": "https://example.com",
  "Parameters": [],
  "TaskId": "scan-001"
}
```

`Target` must refer to a site you are authorized to scan. The daemon passes the original message to the engine on stdin; the engine decides which parameters and test names are valid. Read the [message contract](../Daemon/Messages.md) for results and delivery limitations.

## Next steps

- [Run locally with Go](Go.md) — build the engine and configure the broker.
- [Run with Docker](Docker.md) — start an isolated broker and publish a demo request.
- [Metrics and troubleshooting](../Daemon/Operations.md) — check `/metrics` and diagnose startup issues.
