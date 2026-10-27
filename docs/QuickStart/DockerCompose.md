# Quick Start — Docker Compose

Compose builds **only the daemon image** from this repository. Its Dockerfile copies `/app/cli-app` from `ghcr.io/prawo-i-piesc/engine-antiginx:latest`; the daemon then invokes that executable in `rawjson` mode for incoming tasks. Unlike the standalone engine's one-off Compose command, the daemon stays running and listens to RabbitMQ.

## Prepare RabbitMQ

You need an **existing RabbitMQ broker** reachable from the daemon container. Before starting Compose, declare both durable queues named by `RABBITMQ_QUEUE_NAME` and `RABBITMQ_RESULTS_QUEUE`; the daemon and this Compose file do **not** create them. For an example local broker and queue setup, follow [Start RabbitMQ](Docker.md#start-rabbitmq) and [Declare the queues](Docker.md#declare-the-queues) in the manual Docker guide.

Set `RABBITMQ_URL` to an address reachable **from inside the container**: `localhost` inside the container is not your host or another container. On Docker Desktop, if you followed the manual broker guide, use `amqp://antiginx:localdev@host.docker.internal:5672/`. On Linux, use a hostname/address reachable from the Compose network (and configure the broker/network accordingly). For a remote broker, use its reachable DNS name, credentials and vhost.

## Configure and build

From the repository root:

```sh
cp .env.example .env
```

Fill in at least these values in `.env` (the credentials and host below are **examples for Docker Desktop** using the broker from the manual Docker guide):

```dotenv
PROMETHEUS_PORT=:9091
RABBITMQ_URL=amqp://antiginx:localdev@host.docker.internal:5672/
RABBITMQ_QUEUE_NAME=tasks_queue
RABBITMQ_RESULTS_QUEUE=results_queue
```

The other variables in `.env.example` are optional. Compose sets `ENGINE_ANTIGINX_CALL=/app/cli-app` (the binary copied by the Dockerfile) and `PIPE_COMMUNICATION=true` automatically; the empty `ENGINE_ANTIGINX_CALL` in `.env.example` is for running locally via the [CLI guide](CLI.md). Optional `ANTIGINX_NVD_API_KEY`, `ANTIGINX_URLHAUS_AUTH_KEY`, `ANTIGINX_RDAP_URL` and `ANTIGINX_URLHAUS_URL` are forwarded to the engine. `PREFETCH_COUNT`, `TASK_CHANNEL_SIZE` and `WORKER_POOL` fall back to the daemon's built-in defaults when left empty. Keep credentials in `.env` out of version control.

```sh
docker compose up --build -d
docker compose logs -f daemon-antiginx
```

The build needs access to the engine image on GHCR and the Go/Alpine base images. In the logs, look for `Listening on RabbitMQ started`. Stop following logs with Ctrl-C; the daemon stays running. With `PROMETHEUS_PORT=:9091`, metrics are exposed on the host only at `http://localhost:9091/metrics`:

```sh
curl --fail http://localhost:9091/metrics
```

If you set a different `PROMETHEUS_PORT`, update the `ports` mapping in `docker-compose.yml` accordingly. The Compose service exposes no RabbitMQ ports and will not start or configure a broker.

## Send a task

If you followed the manual Docker guide's broker setup, you can [publish a test task](Docker.md#publish-a-test-task) using its management API. Otherwise publish a JSON task to your existing tasks queue with your broker's credentials; see [Messages and queues](../Daemon/Messages.md) for the task format and results. Only scan targets you own or are authorized to assess.

!!! warning "Current queue semantics"
    Successfully parsed tasks remain unacked and may be redelivered after a restart. Results are not exactly-once; see [acknowledgements and failures](../Daemon/Messages.md#acknowledgements-and-failures).

## Stop

```sh
docker compose down
```

This stops **only the daemon**. Your RabbitMQ broker and its queues remain untouched.
