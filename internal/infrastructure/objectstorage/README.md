# infrastructure/objectstorage

Adapters that implement the object-storage seam (`internal/usecase/boundary/objectstorage`)
against a concrete storage service.

## Layout convention

- **Substrate-agnostic contract** lives at `internal/usecase/boundary/objectstorage` (the seam),
  above this layer — not here. Infrastructure implements that port; it does not own the abstraction.
- **Substrate-specific adapter** lives at `objectstorage/<substrate>/` (e.g. `objectstorage/s3`).
  The package name is the substrate, so the concrete technology stays visible at the import site.
- **`New` in this package is the only place that chooses an implementation.** Both the DI graph and
  the CLI go through it, so retargeting the substrate is a one-function edit.

## S3 is one worked example, not the target

The seam is substrate-agnostic, but a template that ships only an abstraction proves nothing — the
port is only credible once something real is wired through it. So one substrate is implemented
concretely, and the S3 API is that one. It is the **reference**, not the assumption: the S3 API is
the closest thing this space has to a lingua franca, which is why the local container is Garage
rather than AWS itself.

To keep that claim honest rather than aspirational, here is the local container each major
provider's object storage is developed against, so a fork can stand up the equivalent loop on day one.

|Provider|Service|Local container|License|Published by|
|---|---|---|---|---|
|AWS (and any S3-compatible)|S3|`dxflrs/garage`|AGPL-3.0|Deuxfleurs (non-profit)|
|Azure|Blob Storage|`mcr.microsoft.com/azure-storage/azurite`|MIT|Microsoft|
|GCP|Cloud Storage|`fsouza/fake-gcs-server`|BSD-2-Clause|fsouza (individual maintainer)|

Selection follows the same rule as every other dependency here — one replaceable job per component
([ADR-0068](../../../docs/adr/0068-library-selection-policy.md)) — so a single-purpose emulator is
preferred over a suite that emulates a whole cloud. Notes per choice:

- **Garage** speaks the S3 API and nothing else, and stays small enough to run per checkout. Its
  AGPL-3.0 terms attach to distributing a modified Garage; running the published image as a
  development dependency places no obligation on the application that talks to it
- **Azurite** is Microsoft's own emulator and covers Blob *and* Queue, so Azure needs one container
  for both this seam and the worker seam
- **fake-gcs-server** is the de-facto Cloud Storage emulator and the only one with a real user base,
  but it is the one entry here maintained by an individual rather than an organization, and its
  tagged releases are slower than the others. Weigh that before depending on it in CI

Anything S3-compatible (MinIO, Ceph RGW, Cloudflare R2, a managed S3) needs no adapter change at
all — only `OBJECT_STORAGE_ENDPOINT` and credentials. A non-S3 substrate needs a sibling package
under `objectstorage/`, with nothing above this layer changing.
