# Run with Docker

This guide uses manual Docker commands from the **daemon-antiginx repository root**. If you already have a broker and both queues, use the [Docker Compose guide](DockerCompose.md) to build and run just the daemon. Examples use development-only credentials; choose different secrets and secure the broker for any shared environment.

## Start RabbitMQ

```sh
docker network create antiginx-net
docker run -d --name antiginx-rabbit --network antiginx-net \
  -p 127.0.0.1:5672:5672 -p 127.0.0.1:15672:15672 \
  -e RABBITMQ_DEFAULT_USER=antiginx \
  -e RABBITMQ_DEFAULT_PASS=localdev \
  rabbitmq:3-management-alpine
```

Wait for the management UI at `http://localhost:15672` to respond before the next step. The broker is reachable as `antiginx-rabbit` **inside** `antiginx-net` and as `localhost` on the host; Docker containers should not use `localhost` as their RabbitMQ hostname.

## Declare the queues

The daemon does **not** create queues. Use the RabbitMQ management API (requires `curl` on the host):

```sh
curl --fail -u antiginx:localdev -H 'content-type: application/json' \
  -X PUT -d '{"durable":true}' \
  'http://localhost:15672/api/queues/%2F/tasks_queue'

curl --fail -u antiginx:localdev -H 'content-type: application/json' \
  -X PUT -d '{"durable":true}' \
  'http://localhost:15672/api/queues/%2F/results_queue'
```

`%2F` is the default RabbitMQ vhost (`/`). If the broker is not ready, retry after it starts.

## Start the daemon

Use the published image when available, or build it from this repository. The build uses `ghcr.io/prawo-i-piesc/engine-antiginx:latest` to obtain `cli-app` and needs access to that image:

```sh
docker build -t daemon-antiginx:local .
```

Alternatively, pull `ghcr.io/prawo-i-piesc/daemon-antiginx:latest` and replace `daemon-antiginx:local` in the command below with that image name.

```sh
docker run --rm --name antiginx-daemon --network antiginx-net \
  -p 127.0.0.1:9091:9091 \
  -e PROMETHEUS_PORT=:9091 \
  -e RABBITMQ_URL=amqp://antiginx:localdev@antiginx-rabbit:5672/ \
  -e RABBITMQ_QUEUE_NAME=tasks_queue \
  -e RABBITMQ_RESULTS_QUEUE=results_queue \
  -e ENGINE_ANTIGINX_CALL=/app/cli-app \
  -e PIPE_COMMUNICATION=true \
  daemon-antiginx:local
```

The daemon runs in the foreground and logs to stdout. In another terminal, check `curl --fail http://localhost:9091/metrics`. The image has both binaries, but **RabbitMQ is still a separate container**. The Dockerfile exposes port 5000 as metadata; the configured metrics port here is 9091.

## Publish a test task

Only scan a target you own or are authorized to assess. Replace the example target before using this request; depending on the engine, an empty `Parameters` array may not select any tests.

```sh
curl --fail -u antiginx:localdev -H 'content-type: application/json' \
  -X POST 'http://localhost:15672/api/exchanges/%2F/amq.default/publish' \
  -d '{"properties":{"content_type":"application/json","delivery_mode":2},"routing_key":"tasks_queue","payload":"{\"Target\":\"https://example.com\",\"Parameters\":[],\"TaskId\":\"scan-001\"}","payload_encoding":"string"}'
```

The API returns `{"routed":true}` if it routed the request. Check the `results_queue` in the management UI or fetch a snapshot without removing messages:

```sh
curl --fail -u antiginx:localdev -H 'content-type: application/json' \
  -X POST 'http://localhost:15672/api/queues/%2F/results_queue/get' \
  -d '{"count":10,"ackmode":"ack_requeue_true","encoding":"auto","truncate":50000}'
```

Results appear **only when the engine emits result events**; a published task or an `endFlag` event is not itself a result. See [message contract](../Daemon/Messages.md). Valid input deliveries currently remain unacked and may be redelivered after a restart.

Stop the daemon with Ctrl-C; remove the demo broker/network when no longer needed with `docker rm -f antiginx-rabbit` and `docker network rm antiginx-net` (only if you created them for this guide).
