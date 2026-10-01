# Skill for AI Agents

This repo is for both the presentation and the demo project for a developer conference.

## Presentation
The presentation is bare-bones simple Markdown files.
No need for any presentation software.
The presentation is meant to be displayed with a Markdown renderer, like GitHub.


## Demo project instructions

- Keep the demo code simple, bare-bones, and easy to explain live.
- Prefer the Go standard library and the few existing dependencies. Do not add
  production architecture, frameworks, services, or abstractions unless they
  directly demonstrate an API contract concern.
- Optimize for visible API behavior and contract changes rather than ideal
  code quality, completeness, scalability, or operational hardening.
- Use local SQLite, fake identities, environment variables, and resettable
  state so each demo is reliable and easy to repeat.
- Keep error responses, curl commands, and other client-visible behavior
  explicit. Preserve the progression from one demo portion to the next.
- Don't commit code changes to allow for human review.
- Review the slides (in /presentation) around the demo that needs code changes to ensure the demo code accomplishes the goals of the slides.

## `demo/demo.md`

`demo/demo.md` is the presenter’s second-screen runbook, not a project plan.
Each demo section should contain concise presenter setup, copy/paste commands,
expected observations, and a transition back to the slides. Remove speculative
planning notes, alternatives, and unfinished implementation plans from it.

The demo will use a strategy of different directories for each demo step, to be reliable and repeatable.
The presenter should not have to write code during the presentation.

The demo should focus on the API contract instead of the code itself.