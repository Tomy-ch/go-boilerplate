# Evidence collection — Docker images

Read this together with the axis definitions in `SKILL.md`. **Start by deciding whether to run at
all.** This is the thinnest-evidence ecosystem, for a structural reason `images-pin` already records:
a mutable tag has no queryable history, so the registry only answers "what does this tag point to
*now*". There is no published source diff for a rebuild.

The practical consequence: axis D is frequently `?` and axis A often is too, which lands the verdict
on INSUFFICIENT-EVIDENCE — i.e. the window stands, which is what `images-pin` rule 2 already did
without triage. Run this when the decision genuinely needs it:

- **Rule 3** — a fresh image with no prior lock entry, so there is no aged digest to fall back to and
  `resolve` fails closed. There is no "just keep the old pin" option here, so evidence is the only
  input to the choice between waiting and a `days=0` bootstrap.
- A registry or base-image advisory naming this image.
- Time pressure to adopt a rebuild that carries a security patch.

Do not run it to re-justify a routine rule 2 hold.

Never `docker run` the candidate. Pulling and inspecting metadata is fine; executing it is not.

Set up:

```sh
IMG=<image:tag>        # e.g. golang:1.26.5-alpine, postgres:18.3-bookworm
BASE_DIGEST=<from docker/images-pin.toml, if any>
WORK="$SCRATCH/img-$(echo "$IMG" | tr :/ --)"; mkdir -p "$WORK"; cd "$WORK"
grep -n "$IMG" docker/images-pin.toml
docker buildx imagetools inspect "$IMG" --raw > index.json
docker buildx imagetools inspect "$IMG" --format '{{json .Manifest}}' > manifest.json
```

## Axis P — publisher

For images, "publisher" is the namespace and the build pipeline behind it.

```sh
# Official images live under library/ and are built by Docker's own infrastructure
case "$IMG" in */*) echo "third-party namespace";; *) echo "official (library/)";; esac
curl -fsSL "https://hub.docker.com/v2/repositories/library/${IMG%%:*}" 2>/dev/null | jq '{user, is_official: .is_official, last_updated}'
```

- An **official image** (`golang`, `postgres`, `node`) has its Dockerfile published in
  `docker-library/official-images` and is rebuilt by Docker's build fleet. Continuity is strong and
  P is usually `0`.
- A **third-party namespace** (`grafana/otel-lgtm`, a vendor image) carries the vendor's own pipeline.
  Check the namespace is unchanged from what the lockfile recorded and that the repo is not newly
  transferred. A namespace change with the same tag is a `3`.
- For a GHCR image, resolve the source repository (`org.opencontainers.image.source` below) and apply
  `references/github-actions.md`'s P checks to it.

## Axis A — artifact versus source

This is where an official image can actually answer, and most others cannot.

```sh
docker buildx imagetools inspect "$IMG" --format '{{json .Provenance}}' 2>/dev/null | head -40
docker buildx imagetools inspect "$IMG" --format '{{json .Image.Config.Labels}}' | jq
```

- **OCI source labels** — `org.opencontainers.image.source` / `.revision` name the repo and commit
  the image was built from. When present, verify that commit exists and is on the default branch,
  exactly as in `references/github-actions.md`. That closes the axis.
- **SLSA provenance / attestations** — `--format '{{json .Provenance}}'` returns the buildkit
  attestation when the publisher produced one, naming the builder and the source. Where the publisher
  signs with cosign, verify rather than trust the presence:

  ```sh
  cosign verify-attestation --type slsaprovenance "$IMG" 2>&1 | head -20
  ```

- **Official images**: the tag's Dockerfile is in `docker-library/official-images`, so the source is
  checkable even without attestations — confirm the tag is still listed there and its `GitCommit`
  matches upstream.
- With no labels, no attestation, and no official listing, A is `?`. Say so; do not score `0`.

Note what this repo's own release pipeline already does for images it *builds*
(`docs/design/security.md` → ADR-0101 (release-image-supply-chain): signing, provenance, SBOM). That covers our artifacts, not
the third-party base images this axis is about.

## Axis D — what actually changed

There is no source diff for a rebuild, so the substitute is the **package-level SBOM diff**, which is
in fact the real content of a base-image rebuild: which OS packages moved.

```sh
docker buildx imagetools inspect "$IMG" --format '{{json .SBOM}}' > sbom.new.json 2>/dev/null
jq -r '.SPDX.packages[]? | "\(.name)\t\(.versionInfo)"' sbom.new.json | sort > pkgs.new.txt
```

To compare against the baseline, address the **prior digest** explicitly — the tag no longer points
at it:

```sh
docker buildx imagetools inspect "${IMG%%:*}@$BASE_DIGEST" --format '{{json .SBOM}}' > sbom.old.json
jq -r '.SPDX.packages[]? | "\(.name)\t\(.versionInfo)"' sbom.old.json | sort > pkgs.old.txt
diff pkgs.old.txt pkgs.new.txt
```

Interpretation:

- **Only expected packages moved, upward, consistent with a distro security update** → strong benign
  evidence; `0`. This is the normal shape of an official-image rebuild and the one case where triage
  genuinely discharges the window.
- **A package appears that the image had no reason to gain** (a compiler, a network tool, a shell in a
  distroless image) → `3`.
- **No SBOM published on either side** → `?`. Fall back to the config comparison below, which is
  weaker but not nothing.

Config and layer comparison, always available:

```sh
docker buildx imagetools inspect "$IMG" --format '{{json .Image.Config}}' > cfg.new.json
docker buildx imagetools inspect "${IMG%%:*}@$BASE_DIGEST" --format '{{json .Image.Config}}' > cfg.old.json
diff <(jq -S '{Entrypoint,Cmd,Env,User,WorkingDir,ExposedPorts,Volumes}' cfg.old.json) \
     <(jq -S '{Entrypoint,Cmd,Env,User,WorkingDir,ExposedPorts,Volumes}' cfg.new.json)
diff <(jq -r '.history[]?.created_by' cfg.old.json) <(jq -r '.history[]?.created_by' cfg.new.json)
diff <(jq -r '.layers[].digest' manifest.json) /dev/null | head
```

- A changed `Entrypoint` / `Cmd` / `User` on a patch rebuild is a `3` — a security rebuild does not
  change how the image starts or who it runs as.
- A new `Env` entry, a newly exposed port, or a `User` moving to `root` are each findings.
- The `history[].created_by` sequence is the closest thing to a build-recipe diff: extra steps that
  the upstream Dockerfile does not explain are the finding.
- A layer count change far beyond the patch's scope is a `2` pointing at something to explain.

Vulnerability scanning is corroboration, not an answer — the repo already runs Trivy against images:

```sh
trivy image --scanners vuln --severity HIGH,CRITICAL "$IMG" | head -30
```

A rebuild that *removes* CVEs and adds none is consistent with a legitimate patch. It cannot detect a
deliberately planted backdoor, so it never closes axis D on its own.

## Axis S — surface

Covered by the SBOM and config diffs above; report it separately anyway, because "no new packages"
and "no new capabilities" are distinct claims. New `ExposedPorts`, a new `Volumes` entry, a
`User: root` regression, or an added `bin`-like entry in the SBOM all belong here.

## Reporting notes

- The exposure line is the highest class: a base image **executes in CI before our code does**, and
  for a compose service image it also runs in the local dev environment.
- Always restate the rule 3 wall when it applies: even a LOW score leaves the choice between waiting
  for the image to age and a deliberate `days=0` bootstrap, because there is no aged digest to adopt
  instead. `images-pin` will not adopt a fresh digest on its own.
- Say plainly how many axes came back `?`. For this ecosystem two is common, and the honest verdict
  is then INSUFFICIENT-EVIDENCE — which is the same answer the cooldown already gave. That is not a
  wasted run if it was a rule 3 decision; it is the evidence that waiting is the right call.
