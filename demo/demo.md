# Demo runbook

The presentation should use one small appointment API and improve the same operation throughout the session. Every demo should begin from a known state, expose one problem, make one focused change, and then return to the slides.

## Demo principles

- Keep the application local and use fake data only.
- Make the application resettable between demonstrations.
- Prefer `curl` when the request or response is the point; use Bruno when the saved request history is useful.
- Keep the codebase small enough that the audience can understand the relevant change.
- Show the failure before showing the fix.
- Do not demonstrate every possible API feature. Demonstrate the few changes that make the contract safer and clearer.
- Have a backup recording or screenshots available in case the live environment fails.

## Suggested time budget

| Demo | Slides | Purpose | Target time |
| --- | --- | --- | ---: |
| 1 | `03` → `04` | Expose the ambiguity | 4 minutes |
| 2 | `04` → `05` | Make meaning explicit | 3 minutes |
| 3 | `06` → `07` | Test authorization and inspect the audit event | 4 minutes |
| 4 | `08` → `09` | Show duplicate work after a timeout, then add idempotency | 5 minutes |
| 5 | `10` → `11` | Show a slot conflict, then return an actionable error | 5 minutes |
| 6 | `12` → `13` | Show excessive requests, then add rate limiting | 4 minutes |
| 7 | `14` → `15` | Show what a client can discover, then add OpenAPI documentation | 4 minutes |
| 8 | `16` → `17` | Compare the final API with the original | 5 minutes |

The demos total roughly 34 minutes. If the session is shorter, combine demos 2 and 3, or omit the live OpenAPI editing and show the final specification instead.

---

## Demo 1 — Let the agent expose the ambiguity

### Slides

Run after [03.md](../presentation/03.md), before [04.md](../presentation/04.md).

### Goal

Show that the API can be called even though the contract leaves important decisions to inference.

### Starting API

```http
POST /schedule
Content-Type: application/x-www-form-urlencoded

date=tomorrow&time=5pm
```

Example response:

```json
{
  "byuId": "123456789",
  "type": "advisor",
  "time": "5pm",
  "date": "tomorrow"
}
```

### Call the API

```bash
curl -X POST http://localhost:8080/schedule \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode 'date=tomorrow' \
  --data-urlencode 'time=5pm'
```

### Suggested AI instructions

Give the following basic developer instruction to an AI agent:

```text
Use the appointment API localhost:8080/schedule to schedule an advisor appointment for tomorrow at 4pm.
```

- What were the agent's assumptions?
  - How did it know the format for the API call?
  - Which adviser it selected
  - which appointment type
  - which timezone it used;
  - whether it considered the appointment confirmed;
  - whether it retried after a timeout.

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

### Code change

Change the operation from a vague form request to an explicit resource-oriented JSON request:

```http
POST /advisement/appointments
Content-Type: application/json

{
  "student_id": "123456789",
  "advisor_id": "111111111",
  "appointment_type": "academic_advising",
  "starts_at": "2026-12-25T17:00:00-07:00"
}
```

### Steps

1. Show the original handler or route.
2. Rename the operation around the domain resource: `advisement/appointments`.
3. Replace ambiguous form fields with explicit JSON fields.
4. Add validation for required fields and the timestamp format.
5. Call the new endpoint.
6. Send one invalid request to show that the API rejects ambiguity instead of silently guessing.

### Do not solve yet

Do not add idempotency, conflict handling, authorization, or rate limits in this block. The next slides need those problems to remain visible.

### Transition

Return to [05.md](../presentation/05.md), then ask:

> We have made the meaning clearer. Now who is allowed to perform this operation?

---

## Demo 3 — Test authorization and inspect the audit event

### Slides

Run after [06.md](../presentation/06.md), before [07.md](../presentation/07.md).

### Goal

Show that documentation and explicit fields do not make an operation safe by themselves.

### Suggested test cases

| Case | Expected result |
| --- | --- |
| No credentials | `401 Unauthorized` |
| Authenticated user without permission | `403 Forbidden` with Problem Details |
| Authorized user scheduling for themselves | Success |
| Authorized service acting for a user | Success only with valid delegated context |

### Steps

1. Call the endpoint without credentials.
2. Add credentials for an authenticated but unauthorized user.
3. Show the `403` response and its machine-readable problem type.
4. Call the endpoint with an authorized identity.
5. Display the structured audit event.
6. Point out the fields that answer:
   - who acted;
   - which client or agent acted;
   - what operation was attempted;
   - which resource was affected;
   - what the outcome was.

### Transition

Return to [07.md](../presentation/07.md):

> We now know who is allowed to call the endpoint, and we can explain what happened. But what if the response is lost after the appointment is created?

---

## Demo 4 — Show the retry problem, then add idempotency

### Slides

Run after [08.md](../presentation/08.md), before [09.md](../presentation/09.md).

### Goal

Show that a timeout creates uncertainty and that a retry can accidentally create duplicate business work.

### Failure setup

Add a predictable failure switch to the local API, such as:

```text
FAIL_AFTER_CREATE=true
```

The server should create the appointment and then drop or delay the response.

### Steps: failure

1. Reset the application state.
2. Send the appointment request.
3. Make the server create the appointment but withhold the response.
4. Show the client timing out.
5. Retry the identical request.
6. Inspect the appointments and show the duplicate.

### Steps: fix

1. Add an `Idempotency-Key` header.
2. Store the key with the request fingerprint and original result.
3. Retry with the same key.
4. Show that the original result is returned and no second appointment is created.
5. Reuse the same key with different request data and show that the API rejects it.

### Transition

Return to [09.md](../presentation/09.md), then ask:

> Idempotency protects one intended operation from duplication. What about two different clients competing for the same appointment?

---

## Demo 5 — Show a conflict, then make it actionable

### Slides

Run after [10.md](../presentation/10.md), before [11.md](../presentation/11.md).

### Goal

Show that idempotency does not solve concurrent business conflicts.

### Steps: failure

1. Reset the appointment state.
2. Prepare two requests for the same advisor and time slot.
3. Send them concurrently or with a small controlled delay.
4. Show that an unsafe implementation creates two appointments.
5. Ask the audience which request should win.

### Steps: fix

1. Enforce the appointment-slot invariant in the application or database.
2. Allow one request to succeed.
3. Return `409 Conflict` to the other request.
4. Format the response as `application/problem+json`.
5. Highlight the problem `type`, `title`, `status`, `detail`, and API-specific `code`.
6. Show how the client can ask the user to choose another time without silently changing the request.

### Transition

Return to [11.md](../presentation/11.md), then ask:

> We can now handle competing requests. What happens if a client or agent sends hundreds of requests instead?

---

## Demo 6 — Show excessive requests, then add a limit

### Slides

Run after [12.md](../presentation/12.md), before [13.md](../presentation/13.md).

### Goal

Show that an authorized client can still overload the API or the calendar service behind it.

### Steps: failure

1. Reset the API and calendar stub.
2. Run a small request loop against the appointment endpoint.
3. Display request counts and downstream calls.
4. Show the system accepting more traffic than it should.

### Steps: fix

1. Add a deliberately visible per-client or per-user rate limit.
2. Repeat the request loop.
3. Show `429 Too Many Requests`.
4. Show `Retry-After` and the Problem Details response.
5. Explain that the client should wait rather than retry in a tight loop.

### Transition

Return to [13.md](../presentation/13.md), then ask:

> We have made the API safer and more predictable. How does a new client discover these rules?

---

## Demo 7 — Inspect discoverability and document the contract

### Slides

Run after [14.md](../presentation/14.md), before or while presenting [15.md](../presentation/15.md).

### Goal

Show the difference between rules that exist in the implementation and rules that consumers can actually discover.

### Steps

1. Show the endpoint without an OpenAPI document or with an incomplete specification.
2. Ask what a generated client or AI agent can know from the available information.
3. Add the operation to the OpenAPI specification.
4. Include:
   - operation identifier;
   - request schema;
   - required fields and enum values;
   - examples;
   - success and error responses;
   - security requirements;
   - idempotency behavior;
   - rate-limit behavior.
5. Validate the specification against the running API.
6. Show the generated documentation or client view.

### Keep the claim modest

Do not claim that OpenAPI makes an agent deterministic. The point is that the agent has fewer important gaps to guess across.

### Transition

Return to [15.md](../presentation/15.md):

> The contract is only useful if consumers can find it and the implementation continues to honor it.

---

## Demo 8 — Compare the final API

### Slides

Run at the end of [16.md](../presentation/16.md), before [17.md](../presentation/17.md).

### Goal

Compare the original and updated API using the same appointment task.

### Human client

Use `curl` or Bruno to:

1. Discover the documented operation.
2. Submit an explicit appointment request.
3. Include the required identity and idempotency information.
4. Show the successful response.
5. Retry the request and show that it does not create a duplicate.
6. Inspect the audit event.

### AI agent

Give the agent access to the final API documentation and ask it to schedule an appointment.

Observe whether it can:

- identify the required fields;
- use the correct timestamp format;
- provide or request the necessary authorization context;
- respond appropriately to a conflict or rate limit;
- avoid duplicating the appointment after a retry.

### Final comparison

Show the original request beside the final request and ask:

- What had to be inferred before?
- What is explicit now?
- Which failures can the client handle safely?
- What can an operator determine afterward?

### Transition

Move to [17.md](../presentation/17.md) and use the checklist as the lasting takeaway.

---

## Backup plan

If a live demo fails:

1. Explain what the audience was supposed to observe.
2. Show the prepared request and response recording.
3. Show the relevant code diff.
4. Continue with the next slide.

The audience should remember the contract improvements, not the success or failure of the local environment.
