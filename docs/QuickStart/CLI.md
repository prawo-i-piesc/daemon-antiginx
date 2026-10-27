# Run locally with the CLI

These commands assume you are in the **daemon-antiginx repository root**. Install Go 1.26.9+ and a RabbitMQ broker before starting. For a local broker, follow the [Docker guide's RabbitMQ setup](Docker.md#start-rabbitmq) (the daemon itself can still run on your host).

## Build the engine

Clone the [Engine-Antiginx repository](https://github.com/prawo-i-piesc/engine-antiginx) next to this one and build its CLI for your machine:

```sh
git clone https://github.com/prawo-i-piesc/engine-antiginx.git ../engine-antiginx
go -C ../engine-antiginx build -o cli-app ./App
```

The resulting binary is `../engine-antiginx/cli-app` (on Windows, build an `.exe` and use its absolute path). `ENGINE_ANTIGINX_CALL` must be the **path to an executable**, not a shell command with arguments: the daemon supplies `rawjson` itself. The engine must be compatible with the task and event format described in [Messages](../Daemon/Messages.md).

## Configure and start

The daemon loads `.env` from its current directory if present; otherwise it reads the process environment. Set at least:

```dotenv
PROMETHEUS_PORT=:9091
RABBITMQ_URL=amqp://antiginx:localdev@localhost:5672/
RABBITMQ_QUEUE_NAME=tasks_queue
RABBITMQ_RESULTS_QUEUE=results_queue
ENGINE_ANTIGINX_CALL=../engine-antiginx/cli-app
PIPE_COMMUNICATION=true
```

For example, save these lines to `.env` in the daemon repository root (or copy `.env.example` and add the local-only settings shown above). `antiginx:localdev` matches the broker in the [Docker guide](Docker.md#start-rabbitmq); use your actual broker credentials otherwise. Avoid committing `.env` or credentials. Declare **both** queues before starting (see [queue setup](Docker.md#declare-the-queues)).

```sh
go run .
```

Check `http://localhost:9091/metrics` from the host. `PROMETHEUS_PORT` is a Go listen address; the leading colon is intentional. To stop, press Ctrl-C. The daemon also handles SIGTERM.

Optional tuning: `WORKER_POOL` (default 10), `TASK_CHANNEL_SIZE` (default 10), and `PREFETCH_COUNT` (default 12) must be numeric. See [configuration](../Daemon/README.md#configuration).

!!! warning "Current queue semantics"
    Successful tasks are **not acknowledged** by this implementation. Unacked tasks can be redelivered after shutdown; results are not exactly-once. Do not rely on this setup for production delivery guarantees without addressing the [queue limitation](../Daemon/Messages.md#acknowledgements-and-failures).
