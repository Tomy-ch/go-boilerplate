# Context Map

日本語: [context-map.ja.md](../ja/design/context-map.ja.md)

This page characterises every place this system exchanges a model with something it does not own,
using Evans's relationship vocabulary for the edges. It is about **relationships**, not mechanics:
how each integration works belongs to its own design reference, linked from the table.

Maintained by [`/context-map`](../../.claude/skills/context-map/SKILL.md); drift is detected by
[`/context-map-audit`](../../.claude/skills/context-map-audit/SKILL.md).

## An undecided edge is an entry, not a blank

Several of Evans's relationships are distinguished by facts that do not exist in a codebase.
Customer-Supplier and Conformist look identical from the inside — both are "we consume their model"
— and differ only in whether the upstream will take our requirements. That is an organisational
fact, and in a template it is a fact about the *adopting* organisation, not this repository.

So most edges here are recorded as `未確定` with their evidence and the question that would settle
them. **That is the correct state for this repository, not an omission.** An unlabelled edge invites
the adopter to decide; a wrongly labelled one closes the question before they see it.

## What counts as a contact point

A model crossing out of this system. A boundary port with no external counterpart — the clock, the
transaction manager, the job and worker runners — is not a contact point, however much it looks like
one in the port list.

Both directions count. A map with only outbound edges would claim this system is nobody's upstream.

## Edges

| Counterpart | Direction | Relationship | Evidence | If the other side moves |
| --- | --- | --- | --- | --- |
| HTTP API consumers | this system is upstream | **Open Host Service + Published Language** | `openapi/openapi.gen.yaml` is committed as a consumable artifact and kept fresh by a drift gate; the surface is one protocol for all consumers rather than per-consumer endpoints | Consumers break only when the published contract changes, and the gate makes that change visible in review |
| External IdP | this system is downstream | `未確定` | Token claims are converted to this system's own `Authn` inside the adapter (`internal/infrastructure/auth/jwt/auth_jwt.go:222`); the external vocabulary does not reach the inside — the structural signature of an Anticorruption Layer | An issuer or claim change is absorbed by the adapter; a protocol change is not |
| Object storage (S3-compatible) | this system is downstream | `未確定` | The port is stated in this system's vocabulary and `internal/usecase/boundary/README.md:144` declares that vendor vocabulary (bucket / region / etag) never crosses the boundary | A vendor swap is confined to the adapter; the port is unaffected |
| Outbox receivers | this system is upstream | `未確定` | The publish boundary carries a substrate-agnostic envelope, but no payload schema is published — [ADR-0052](../adr/0052-message-id-idempotency-propagation.md) defines only the transport convention. See [`outbox.md`](outbox.md) | A receiver's expectations are not protected by any artifact this system publishes |

## The open questions

Each `未確定` edge above is waiting on one thing. Answering it is a decision, not a lookup.

- **External IdP** — can this system's requirements reach whoever runs the IdP? If yes the edge is
  Customer-Supplier; if the IdP is a standard or a vendor that will not negotiate, it is Conformist.
  The Anticorruption Layer is present either way — it is the structural remedy, not the relationship.
- **Object storage** — the same question, with the same Anticorruption Layer already in place. The
  substitutability the boundary buys suggests the relationship is not load-bearing, but that is a
  reading of intent, not evidence of one.
- **Outbox receivers** — are the receivers a known set whose requirements shape the payload
  (Customer-Supplier), or an open set consuming what is emitted (Open Host Service)? The asymmetry
  with the synchronous side is recorded in [`outbox.md`](outbox.md) §4: the language is not published.

## Sample-derived contact points

The sample feature set adds outbound gateways to external web APIs. **They disappear when the sample
is removed**, and are listed separately so this map does not keep describing edges that no longer
exist:

- Address lookup gateway and exchange-rate gateway (`internal/infrastructure/webapi/**`), each behind
  a semantic port in the usecase boundary.

What survives their removal is the pattern they demonstrate: a `<service>.Gateway` port stated in
this system's vocabulary, with transport and vendor failure modes translated at the adapter. A real
integration added later occupies the same shape and earns its own row in the table above.

## Diagram

```mermaid
flowchart LR
    Consumers["HTTP API consumers"] -->|OHS + Published Language| SUT["This system"]
    IdP["External IdP"] -->|未確定 · ACL present| SUT
    SUT -->|未確定 · ACL present| Storage["Object storage (S3-compatible)"]
    SUT -->|未確定 · no published language| Receivers["Outbox receivers"]
```

## What this map does not cover

- **The database.** It stores this system's own model rather than owning one, so no model crosses
  out of the system at that edge.
- **Ports with no external counterpart** — clock, transaction manager, job and worker runners.
- **How any integration works.** Mechanics live in the per-subsystem references under
  [`docs/design/`](README.md) and the package READMEs; this page only says how the two sides relate.
