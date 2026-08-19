---
status: accepted
date: 2026-08-19
deciders: [maintainers]
tags: [process, ai, observability]
---

# ADR-0010: Make the development window the unit the feedback loop observes

## Status

accepted

## Context

[ADR-0008](0008-agent-environment-alignment.md) decides that the agent environment is improved in
a closed loop, and [ADR-0009](0009-long-running-agent-state.md) decides where that loop's
observation may live. Neither says what the loop observes *one of*. Without that, every metric it
produces is ambiguous: "how long did this take" has no subject.

The obvious candidate is the session, and it is wrong. Measured over thirty days of this
repository's own sessions:

- A session runs far longer than a piece of work — median 3.1 h, p75 11.2 h, p90 26 h, longest
  104.7 h. People clear their context between tasks; they do not quit and relaunch.
- Only 48 % of sessions touch exactly one working branch. 38 % touch two or more, and 13 % touch
  none at all (investigation on the base branch).
- A session is, however, almost always one checkout: 84 % have a single `cwd`.

So a session is roughly a *place* of work, not a *piece* of work. Summing timings per session
would average together several unrelated tasks, which is precisely the number nobody can act on.

A second question follows from it. Some of what the loop wants to know is also recorded by
GitHub — when a pull request opened, when it merged — which invites the conclusion that the loop
should simply read it back and record nothing locally.

## Decision

**The unit of observation is the development window: one piece of work, from the moment it is
picked up to the moment it is put down.** A window is identified by its session, checkout,
branch, and the person working it, and several sessions may belong to one window as several
windows may belong to one issue.

**A window is bounded by what a person does, not by what the runtime does.** `/clear`, a manual
`/compact`, and the end of a session close a window. Automatic compaction does not: it is the
context filling up mid-task, and treating it as a boundary would cut a long task in half and
halve both of its phase timings. The two forms of compaction are distinguishable only at the
`PreCompact` event, whose payload carries `trigger`; the later `SessionStart` reports both as
`compact`. A branch switch is corroborating evidence, not a boundary, because moving between a
base branch and a feature branch happens inside one piece of work.

**Where a moment can be observed at its source, the source is the record.** Phase boundaries are
stamped locally by the workflow that crosses them, and GitHub is a cross-check and a fallback,
never the origin. The two are not the same fact — a mark says a workflow reached a phase, an API
says an object was created, and they part company when a pull request is opened outside the
window or a push fails after the phase was reached. They are not equally reliable either: reading
a time back needs the branch-to-pull-request join, measured at 64 %, and it needs the network,
which the loop is required to work without. A recorded disagreement between the two is a finding,
not noise, so every value carries which source produced it.

## Consequences

### Positive Consequences

- Durations mean something. "How long did implementation take" is answerable because the interval
  has one subject, and phases within a window can be compared across windows.
- The loop keeps working offline and without credentials, because the primary record is written
  where the work happens.
- Work that never reaches a pull request — investigation, and the attempt that was abandoned — is
  still a window with a duration, and those are the two kinds this repository previously could not
  see at all.

### Negative Consequences

- A boundary is only as good as the signal behind it. Someone who never clears context runs one
  very long window, and its phase timings are correspondingly coarse. The loop reports the window
  it observed; it does not infer boundaries that were never signalled.
- Marks are written by the workflows that carry them, so a window worked without those workflows
  is missing its middle. Measured, 51 % of commits do not go through the commit workflow — which
  is why the commit mark is stamped by a git hook instead, and why the remaining marks degrade to
  values derived from the transcript rather than to nothing.
- Two records of the same moment can disagree, and something has to look at the disagreement.

## Alternatives Considered

### Treat the session as the unit

Rejected on measurement: 38 % of sessions contain two or more working branches, so per-session
timings would pool unrelated tasks. The failure is silent — the numbers still compute — which is
the worst kind.

### Treat a branch switch as the boundary

Rejected. A single piece of work routinely moves between the base branch and its feature branch,
so this would split most windows at least once. It is kept as corroboration for a boundary that a
person's own signal already established.

### Read every timing back from GitHub and stamp nothing

Rejected. It fails offline, it fails without credentials, and it fails for the third of working
branches with no pull request to join to. It also answers a different question than the one asked,
since an object's creation time is not the moment a workflow reached a phase.

### Derive boundaries by asking a model to read the transcript

Rejected. The boundary would then depend on a reading rather than on a fact, and
[ADR-0008](0008-agent-environment-alignment.md) requires the loop's judgements to rest on
deterministic evidence wherever one exists. The signals here are deterministic and already present.

## Notes

- Interpretation: [`docs/design/closed-loop.md`](../design/closed-loop.md).
- Related: [ADR-0008](0008-agent-environment-alignment.md) (why there is a loop at all),
  [ADR-0009](0009-long-running-agent-state.md) (what may be persisted, and where).
- The marks themselves and the reasoning behind each one live in
  `.agents/closed-loop/marks.sh`.
