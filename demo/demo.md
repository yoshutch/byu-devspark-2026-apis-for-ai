# Demo runbook

Each demo starts from its own resettable code folder, shows one contract problem, and returns to the slides with one focused improvement.

---

## Demo 1 — Let the agent expose the ambiguity

### Slides

Run after [03.md](../presentation/03.md), before [04.md](../presentation/04.md).

### Goal

Show that the API works, while leaving important decisions to inference.

### Presenter setup

```bash
task start DEMO=demo1
```

### Steps

```bash
curl -i -X POST http://localhost:8080/schedule \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode 'date=tomorrow' \
  --data-urlencode 'time=5pm'
```

Pause on the `200 OK` response. Ask:

- What does this response prove?
- What does it not tell us?
- Who is the appointment for?
- Which advisor and appointment type were selected?
- Which timezone does `5pm` use?
- Does `200 OK` mean the appointment is confirmed?

### Ask the AI agent

Open a scratch codex session (not in this directory).
Give the agent only this basic instruction:

```text
Use the appointment API at http://localhost:8080/schedule to schedule an advisor appointment for tomorrow at 5pm.
```

Ask the agent to explain its assumptions after making the request. Capture:

- how it knew the request format;
- which advisor and appointment type it inferred;
- which timezone it used;
- whether it considered the appointment confirmed;
- what it would do if the response timed out.

View the recorded appointments in the database:

```shell
task query DEMO=demo1
```

### Transition

Return to [04.md](../presentation/04.md):

> The problem is not that the endpoint is impossible to call.
> The problem is that too much of the contract exists only in somebody’s head.

---

## Demo 2 — Make the meaning explicit

### Slides

Run after [04.md](../presentation/04.md), before or while presenting [05.md](../presentation/05.md).

### Goal

Improve the request’s domain meaning without solving every reliability or security problem yet.

### Major code changes (optional presenter note)

- Replace `POST /schedule` form data with `POST /advisement/appointments` JSON.
- Require explicit fields and RFC3339 `starts_at` values.
- Reject missing, unknown, or ambiguous input.

### Presenter setup

```bash
task start DEMO=demo2
```

### Steps

1. Show the old request from Demo 1. It no longer matches the API:

   ```bash
   curl -i -X POST http://localhost:8080/schedule \
     --header 'Content-Type: application/x-www-form-urlencoded' \
     --data-urlencode 'date=tomorrow' \
     --data-urlencode 'time=5pm'
   ```

2. Send the new explicit request:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Content-Type: application/json' \
     --data '{
       "student_id": "123456789",
       "advisor_id": "111111111",
       "appointment_type": "academic_advising",
       "starts_at": "2026-10-08T17:00:00-07:00"
     }'
   ```

3. Remove a required field. The API returns `400 Bad Request` instead of guessing:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Content-Type: application/json' \
     --data '{
       "student_id": "123456789",
       "appointment_type": "academic_advising",
       "starts_at": "2026-10-08T17:00:00-07:00"
     }'
   ```

4. Use an ambiguous timestamp. The API rejects it because it has no date or time zone:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Content-Type: application/json' \
     --data '{
       "student_id": "123456789",
       "advisor_id": "111111111",
       "appointment_type": "academic_advising",
       "starts_at": "tomorrow at 5pm"
     }'
   ```

### Transition

Return to [05.md](../presentation/05.md), then ask:

> We have made the meaning clearer. Now who is allowed to perform this operation?

---

## Demo 3 — Test authorization and inspect the audit event

### Slides

Run after [06.md](../presentation/06.md), before [07.md](../presentation/07.md).

### Goal

Show that documentation and explicit fields do not make an operation safe by themselves.

### Major code changes (optional presenter note)

- Add fake bearer authentication and authorization.
- Return `401`/`403` responses and record structured audit events.

The demo tokens are intentionally fake.

### Presenter setup

```bash
task start DEMO=demo3
```

Use the same request body for each test:

```bash
APPOINTMENT='{
  "student_id": "123456789",
  "advisor_id": "111111111",
  "appointment_type": "academic_advising",
  "starts_at": "2026-10-08T17:00:00-07:00"
}'
```

### Steps

1. Call the endpoint without credentials. Expect `401 Unauthorized`:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Content-Type: application/json' \
     --data "$APPOINTMENT"
   ```

   Point out that the audit event has no `actor_id` or `client_id` and records:
   `outcome: rejected`, `reason: authentication_required`, `status: 401`.

2. Use a valid identity without the required permission. Expect `403 Forbidden`:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-readonly' \
     --header 'Content-Type: application/json' \
     --data "$APPOINTMENT"
   ```

   Show the `application/problem+json` response and the audit event with
   `actor_id: user-999999999`, `reason: insufficient_permission`, and `status: 403`.

   The `type` URI is also a local documentation endpoint. Show that an agent
   can inspect it without needing access to an external documentation site:

   ```bash
   curl -i http://localhost:8080/problems/appointment-not-permitted
   ```

3. Use an authorized identity scheduling for itself. Expect success:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --data "$APPOINTMENT"
   ```

   Show the response and the server log’s `AUDIT` event with the actor, client,
   appointment resource ID, `outcome: success`, and `status: 200`.

4. Inspect the audit trail in the server terminal. Every attempt should be represented by a structured, human-readable `slog` record:

   ```text
   time=... level=INFO msg=audit event=appointment.create ... outcome=rejected ...
   time=... level=INFO msg=audit event=appointment.create ... outcome=success ...
   ```

Point out that the audit event answers:

- who acted;
- which client or agent acted;
- what operation was attempted;
- which resource was affected;
- what the outcome was, including rejected attempts.


### Transition

Return to [07.md](../presentation/07.md):

> We now know who is allowed to call the endpoint, and we can explain what happened. But what if the response is lost after the appointment is created?

---

## Demo 4 — Show the retry problem, then add idempotency

### Slides

Run after [08.md](../presentation/08.md), before [09.md](../presentation/09.md).

### Goal

Show that a timeout creates uncertainty and that a retry can accidentally create duplicate business work.

### Major code changes (optional presenter note)

- Add a one-time response delay controlled by `DEMO_TIMEOUT_AFTER_CREATE`.
- Store and replay responses for repeated `Idempotency-Key` values.

### Presenter setup

Reset the database and start the server with a three-second, server-side response delay:

```bash
task start DEMO=demo4 TIMEOUT=3s
```

The timeout is configured on the server. The request itself looks normal.

Use the same request body for each test:

```bash
APPOINTMENT='{
  "student_id": "123456789",
  "advisor_id": "111111111",
  "appointment_type": "academic_advising",
  "starts_at": "2026-10-08T17:00:00-07:00"
}'
```

### Steps: show the retry problem

1. Send a normal request with a short client timeout and no idempotency key:

   ```bash
   curl --max-time 1 -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --data "$APPOINTMENT"
   ```

2. Explain that the client timed out, but the server already saved the appointment.
   Wait for the three-second server delay to finish, then inspect the database:

   ```bash
   task query DEMO=demo4
   ```

3. Retry the identical request without an idempotency key:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --data "$APPOINTMENT"
   ```

   Query the database again and show the duplicate appointment.

### Steps: show the idempotency fix

Restart the server so the one-time delay is available again:

Stop the server with `Ctrl-C`, then run:

```bash
task start DEMO=demo4 TIMEOUT=3s
```

1. Add an `Idempotency-Key` to the first request:

   ```bash
   curl --max-time 1 -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --header 'Idempotency-Key: demo-retry-1' \
     --data "$APPOINTMENT"
   ```

2. Retry with the same key and request data:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --header 'Idempotency-Key: demo-retry-1' \
     --data "$APPOINTMENT"
   ```

3. Query the database and show that only one appointment exists:

   ```bash
   task query DEMO=demo4
   ```

4. Reuse the same key with different request data. The API rejects the request:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --header 'Idempotency-Key: demo-retry-1' \
     --data '{
       "student_id": "123456789",
       "advisor_id": "111111111",
       "appointment_type": "academic_advising",
       "starts_at": "2026-10-08T18:00:00-07:00"
     }'
   ```

Point out that the idempotency key does not make the request happen twice. It
lets the server recognize that the retry is the same intended operation.

### Transition

Return to [09.md](../presentation/09.md), then ask:

> Idempotency protects one intended operation from duplication. What about two different clients competing for the same appointment?

---

## Demo 5 — Show a conflict, then make it actionable

### Slides

Run after [10.md](../presentation/10.md), before [11.md](../presentation/11.md).

### Goal

Show that idempotency does not solve concurrent business conflicts.

### Business rule

An advisor cannot have two appointments in the same time slot.

### Major code changes (optional presenter note)

- Enforce `(advisor_id, starts_at)` uniqueness in SQLite.
- Translate the losing request into an actionable `409` Problem Details response.

### Presenter setup

Use two terminals. In Terminal 1, start the unsafe API:

```bash
task start DEMO=demo5-start SLOT_DELAY=500ms
```

In Terminal 2, define the request body once:

```bash
APPOINTMENT='{
  "student_id": "123456789",
  "advisor_id": "111111111",
  "appointment_type": "academic_advising",
  "starts_at": "2026-10-08T17:00:00-07:00"
}'
```

### Steps: failure

1. Run these two requests concurrently. The different idempotency keys
   represent two different clients requesting the same slot:

   ```bash
   curl -sS -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --header 'Idempotency-Key: demo-client-a' \
     --data "$APPOINTMENT" > /tmp/demo5-client-a.out &

   curl -sS -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --header 'Idempotency-Key: demo-client-b' \
     --data "$APPOINTMENT" > /tmp/demo5-client-b.out &

   wait
   cat /tmp/demo5-client-a.out
   cat /tmp/demo5-client-b.out
   ```

2. Show that both requests return `200 OK`.
3. Query the database:

   ```bash
   task query DEMO=demo5-start
   ```

4. Show that two appointments were created for the same advisor and time. Ask:

   > Which request should win, and where should that rule be enforced?

### Steps: fix

1. Stop the server in Terminal 1, then start the fixed API:

   ```bash
   task start DEMO=demo5-fix SLOT_DELAY=500ms
   ```

2. Run the same two `curl` commands again. This time, one request returns
   `200 OK` and the other returns `409 Conflict` with
   `application/problem+json`.

3. Show the response fields:

   - `type`: `appointment-slot-unavailable`;
   - `title`: `Appointment slot unavailable`;
   - `status`: `409`;
   - `code`: `APPOINTMENT_SLOT_UNAVAILABLE`;
   - `detail`: the requested slot is no longer available.

   Follow the `type` URI to show the additional guidance available to an
   agent:

   ```bash
   curl -i http://localhost:8080/problems/appointment-slot-unavailable
   ```

4. Query the database again:

   ```bash
   task query DEMO=demo5-fix
   ```

5. Show that only one appointment exists. Explain that the client should ask
   the user to choose another time or offer available alternatives; it should
   not silently change the requested appointment.

### Transition

Return to [11.md](../presentation/11.md), then ask:

> We can now handle competing requests. What happens if a client or agent sends hundreds of requests instead?

---

## Demo 6 — Show excessive requests, then add a limit

### Slides

Run after [12.md](../presentation/12.md), before [13.md](../presentation/13.md).

### Goal

Show that an authorized client can still overload the API or the calendar service behind it.

### Major code changes (optional presenter note)

- Add a configurable per-client fixed-window rate limiter.
- Return `429` with `Retry-After` and audit rejected requests.

### Presenter setup

Use a fresh server for each half of the demo. The limiter is disabled when
`DEMO_RATE_LIMIT` is not set.

In Terminal 1, start the API without a rate limit:

```bash
task start DEMO=demo6
```

Use Terminal 2 for the request loop in the steps below.

### Steps: failure

1. Run the loop. All five requests return `200 OK`:

   ```bash
   for minute in 01 02 03 04 05; do
     curl -sS -o /dev/null -w "starts_at minute ${minute}: %{http_code}\n" \
       -X POST http://localhost:8080/advisement/appointments \
       --header 'Authorization: Bearer demo-student' \
       --header 'Content-Type: application/json' \
       --data "{\"student_id\":\"123456789\",\"advisor_id\":\"111111111\",\"appointment_type\":\"academic_advising\",\"starts_at\":\"2026-10-08T17:${minute}:00-07:00\"}"
   done
   ```

2. Query the database and show that all five requests created appointments:

   ```bash
   task query DEMO=demo6
   ```

3. Explain that authentication and authorization identify a trusted client,
   but they do not limit how much work that client can request.

### Steps: fix

1. Stop the server in Terminal 1, clear the database, and restart it with a
   three-request limit over a ten-second window:

   ```bash
   task start DEMO=demo6 RATE_LIMIT=3 RATE_WINDOW=10s
   ```

2. Run the same five-request loop again. The first three requests return
   `200 OK`; the remaining requests return `429 Too Many Requests`.

3. Send one request with headers and the response body visible:

   ```bash
   curl -i -X POST http://localhost:8080/advisement/appointments \
     --header 'Authorization: Bearer demo-student' \
     --header 'Content-Type: application/json' \
     --data '{
       "student_id": "123456789",
       "advisor_id": "111111111",
       "appointment_type": "academic_advising",
       "starts_at": "2026-10-08T18:00:00-07:00"
     }'
   ```

   Point out the `Retry-After` header and the Problem Details fields:
   `type`, `title`, `status`, `code`, and `detail`.

4. Follow the `type` URI for the client guidance:

   ```bash
   curl -i http://localhost:8080/problems/rate-limit-exceeded
   ```

5. Query the database again. Only the first three requests were accepted:

   ```bash
   task query DEMO=demo6
   ```

6. Explain that a client should wait for the `Retry-After` duration instead
   of retrying immediately in a tight loop. The server terminal also shows the
   rejected attempts in the audit log with `reason=rate_limit_exceeded`.

### Transition

Return to [13.md](../presentation/13.md), then ask:

> We have made the API safer and more predictable. How does a new client discover these rules?

---

## Demo 7 — Inspect discoverability and document the contract

### Slides

Run after [14.md](../presentation/14.md), before or while presenting [15.md](../presentation/15.md).

### Goal

Show the appointment contract through its OpenAPI document and Scalar reference UI.

### Major code changes (optional presenter note)

- Add and serve the OpenAPI document at `/openapi.json`.
- Render the same document at `/docs` with Scalar.

### Presenter setup

```bash
task start DEMO=demo7
```

### Steps

1. Fetch the machine-readable contract:

   ```bash
   curl -i http://localhost:8080/openapi.json
   ```

   Point out the `3.1.0` document, version `1.0.0`, and the
   `createAdvisementAppointment` operation. For a focused view of the operation:

   ```bash
   curl -s http://localhost:8080/openapi.json | jq '.paths["/advisement/appointments"].post'
   ```

   Show that the document now includes required fields, the
   `academic_advising` enum, RFC3339 `date-time`, examples, success and error
   responses, bearer authentication, `Idempotency-Key`, `Retry-After`, and
   guidance for retries and conflicts.

2. Open `http://localhost:8080/docs` in a browser. Scalar renders the same
   `/openapi.json` document as an interactive reference, so the operation
   details are not duplicated in handwritten HTML.

### Transition

Return to [15.md](../presentation/15.md):

> The contract is only useful if consumers can find it and the implementation continues to honor it.

---

## Demo 8 — Compare the final API

### Slides

Run at the end of [16.md](../presentation/16.md), before [17.md](../presentation/17.md).

### Goal

Give an AI agent the final contract and see whether it can complete the appointment task safely.

### Presenter setup

```bash
task start DEMO=demo7
```

Use the server terminal to watch the audit event and a scratch Codex session
for the agent walkthrough.

### AI agent

Open a scratch Codex session, not in this directory. Give the agent the
discoverable contract and the task:

```text
Use the advisement appointment API at http://localhost:8080.
Read the OpenAPI document at http://localhost:8080/openapi.json first.
Schedule an academic advising appointment for student 123456789 with advisor
111111111 on 2026-10-08 at 5:00 PM Mountain Time. Use an idempotency key.
You may use the fake demo bearer token `demo-student` for authorization.
After the first response, repeat the same request with the same idempotency key
and explain what happens.
Explain how you chose the request fields and how you would safely retry if the
response timed out.
```

Observe whether the agent can:

- find `POST /advisement/appointments` in the OpenAPI document;
- identify the required fields and `academic_advising` enum;
- use the RFC3339 timestamp format;
- use the provided bearer token for the authorized student;
- include an idempotency key;
- distinguish a successful response from a replayed retry.

Inspect the resulting appointment and audit event:

```bash
task query DEMO=demo7
```

### Final comparison

Show the original request beside the final request and ask:

- What had to be inferred before?
- What is explicit now?
- Which failures can the client handle safely?
- What can an operator determine afterward?

### Transition

Move to [17.md](../presentation/17.md) and use the checklist as the lasting takeaway.
