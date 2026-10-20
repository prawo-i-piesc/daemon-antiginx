# Messages and queues

Both queues must exist **before** starting the daemon. The consumer calls `Consume` on `RABBITMQ_QUEUE_NAME` with manual acknowledgements; the publisher sends messages to `RABBITMQ_RESULTS_QUEUE` using the default exchange and persistent delivery mode. Declare durable queues if results must survive a broker restart. The daemon does not declare queues, configure bindings or set up a dead-letter exchange.

## Scan task (input)

Publish a JSON body to the tasks queue:

```json
{
  "Target": "https://example.com",
  "Parameters": [],
  "TaskId": "scan-001"
}
```

| Field | Type | Description |
| --- | --- | --- |
| `Target` | string | URL or target passed to the engine and associated with results. |
| `TaskId` | string | Scan identifier used for result correlation (published as `testId`). |
| `Parameters` | array | Engine command parameters, e.g. `[{"Name":"test", "Arguments":["https"]}]`; supported names depend on the engine. |

The router checks only that the body can be decoded as JSON into these Go fields; it does not validate required values. The original bytes are forwarded to `cli-app rawjson` on stdin. RabbitMQ's optional `MessageId` is only used in rejection logs, **not** as the scan ID. Engine events must be newline-delimited JSON objects on stdout. Non-JSON lines are skipped.

## Scan result (output)

Each non-terminal engine result event produces **one** message in the results queue (not an array of all test results). Its JSON shape is:

```json
{
  "target": "https://example.com",
  "testId": "scan-001",
  "result": {
    "testResult": {
      "Name": "Example test",
      "Certainty": 90,
      "ThreatLevel": 2,
      "Metadata": {},
      "Description": "Example finding"
    },
    "scanInfo": { "Message": "", "Code": 0 },
    "resultType": 1
  }
}
```

The fields inside `testResult` and `scanInfo` come from the engine event; values above are illustrative, **not** guaranteed scan output. `testId` is the task's `TaskId`, not the engine test identifier. Threat levels are numeric (`0` none, `1` info, `2` low, `3` medium, `4` high, `5` critical); result types are `0` message and `1` success. A result event with `endFlag: true` is skipped, so no completion message is published. Log events go to structured logs; metrics events update Prometheus counters/gauges/histograms; error events are currently ignored.

## Acknowledgements and failures

!!! warning "No successful-task acknowledgement"
    The consumer runs with `autoAck=false`, but the current worker never calls `Ack()` or `Reject()` for **successfully parsed** tasks. Such deliveries remain unacked while the connection is open and can be redelivered if it closes. This can cause duplicate scans/results and eventually fill the prefetch window. Do not assume at-least-once result delivery or exactly-once processing.

Malformed JSON tasks are rejected with `requeue=false`. Engine exit failures are logged as worker errors (code 400); internal errors can cancel the worker context (code 500). The result publisher waits for broker confirmation and retries publication, but after retries are exhausted it only logs an error; publication and scan completion are not transactionally tied to queue acknowledgement. The daemon does not expose a scan-status or completion API.
