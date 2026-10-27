# Quick Start

Daemon-Antiginx is a **queue consumer**, not a standalone web scanner. It needs a RabbitMQ broker, predeclared task and result queues, and an Engine-Antiginx executable. Choose [CLI](CLI.md) to run from source, [Docker](Docker.md) for manual containers, or [Docker Compose](DockerCompose.md) to build and run the daemon with an existing broker and queues.

## Requirements

| Component | Local CLI | Docker / Docker Compose |
| --- | --- | --- |
| Go | 1.26.9 or newer (see `go.mod`); build the engine separately | Not needed on the host |
| Docker | Optional for RabbitMQ; required for the e2e test | Required for the daemon image |
| RabbitMQ | Broker reachable by AMQP | Existing broker reachable from the daemon container |
| Engine-Antiginx | Build `cli-app` for your OS and CPU | Copied from the engine image during image build |

The [Docker Compose guide](DockerCompose.md) builds and runs the daemon only; you must prepare RabbitMQ and both queues separately. The [Docker guide](Docker.md) shows how to start a local broker and declare its queues.

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

- [Run locally with the CLI](CLI.md) — build the engine and configure the broker.
- [Run with Docker](Docker.md) — start an isolated broker and publish a demo request.
- [Run with Docker Compose](DockerCompose.md) — build and run the daemon against an existing broker.
- [Metrics and troubleshooting](../Daemon/Operations.md) — check `/metrics` and diagnose startup issues.
