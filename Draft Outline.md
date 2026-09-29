# APIs for Humans and Machines
> Detailed OpenAPI specifications, discoverability, rate limits, consistent errors, idempotent endpoints, meaningful semantics, and asynchronous operations are not merely conveniences for human developers. They may become essential when machines and AI agents consume our APIs.
> 
> We’ll walk through an example API contract and see how better API hygiene improves usability for both humans and machines.

> AI agents are changing how we think about APIs—not because APIs need to become magical, but because machines expose ambiguity we have long tolerated. In this hands-on session, we’ll examine an intentionally imperfect API, explore what human and machine consumers can infer from it, and improve it through live demonstrations. You’ll leave with practical ways to design APIs that are clearer, safer, and more reliable for every consumer.


## Outline
1. Show the ambiguous endpoint.
2. Let the audience diagnose it.
3. What is an API and why does AI expose API weaknesses?
4. Repair one concrete operation live.
5. Have a human client and an AI agent use the improved version.
6. Show retry, failure, authorization, and observability.
7. End with the checklist.

## What is an API?
- Should it be a CRUD API?
- DDD bounded context
- An API is an interface to the domain: exposing the bounded context's meaningful capabilities
## What is this?
Let's schedule an advising appointment:
```http
POST localhost:8080/schedule -d "date=tomorrow;time=5pm"

200 OK
{
	"byuId": "123456789",
	"type": "advisor",
	"time": "5pm",
	"date": "tomorrow"
}
```
---
- Which advisor or what appointment type is it?
- When is this appointment? What time zone? What day?
- What if two clients select the same time slot?
- Does the API send a confirmation email?
- What happens if the request times out and it is retried?
---
- What do you think an AI agent would do with this?
- Would it be deterministic?
- How many times would it try?

Let's try it!
## How can we fix it?
What are some ways we can fix this API?
## "I can fix it!"
### Better REST
- What are the domain resources?
- URI, HTTP methods, status codes
```http
POST /advisement/appointments
Content-Type: application/json

{ 
	"time": "2026-12-25T10:12:000Z",
	"type": "schedule_advice",
	"advisor_id": "111111111",
	"byu_id": "123456789"
}


```
- 
### Secure the API
- Always authenticate and authorize API requests
- Audit log
- Think through "abuse cases" not just happy path "use cases"
### Better Errors
- HTTP status codes
- RFC 9457
### Document the API contract
- OpenAPI spec
	- helps you think through enums, explicit formats (like date time)
- How does the AI agent find the spec?
- Semantic layer?
### Handle Conflicts
- Design endpoints, prefer idempotency
	- Include an idempotency key
- Return 409 with explicit error details to inform what to do next
### Protect the API
- Rate limit
- Asynchronous
### Version your work
- Tips on how to update API contract
- Deprecation and sunset warnings
## Did it work?
1. Are the date, time, time zone, and appointment type unambiguous?
2. Is it safe to retry without creating duplicate appointments?
3. What happens when two clients request the same slot?
4. Can we tell who made the appointment and what happened afterward?

- Manually inspect the API endpoint(s) now
- AI agent to try using the API endpoint(s) 
## Checklist
- [ ] Can a new developer understand it without tribal knowledge?
- [ ] Can a machine distinguish between success, failure, retry, and "still processing"?
- [ ] Can a machine discover what it is allowed to do?
- [ ] Is it safe to retry API requests?
- [ ] Does the response clearly state what happened (including error responses)?
- [ ] Can the API operations be audited and diagnosed later?
