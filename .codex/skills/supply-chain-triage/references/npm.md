# Evidence collection — npm

Read this together with the axis definitions in `SKILL.md`. Work in the scratchpad, never in the
repo tree, and never run `npm install` on the candidate — its lifecycle scripts are the attack.

Set up once:

```sh
PKG=<package>          # e.g. brace-expansion, @hono/node-server
CAND=<candidate>       # the version the window caught
BASE=<baseline>        # the version currently in the lockfile
WORK="$SCRATCH/npm-$PKG"; mkdir -p "$WORK"; cd "$WORK"
curl -fsSL "https://registry.npmjs.org/$PKG" > packument.json
```

The packument answers most of P and S without downloading anything.

## Axis P — publisher

```sh
jq '{maintainers: [.maintainers[].name]}' packument.json
jq --arg c "$CAND" --arg b "$BASE" \
  '{candidate: .versions[$c]._npmUser, baseline: .versions[$b]._npmUser}' packument.json
jq --arg c "$CAND" --arg b "$BASE" '{cand: .time[$c], base: .time[$b], modified: .time.modified}' packument.json
```

- `_npmUser` is who actually published that version. A candidate published by an account that has
  published nothing before in this package is the single most important P observation.
- Compare against the whole publish history, not just the baseline — an account that published
  years ago and reappears now is different from a steady co-maintainer.
- Check cadence from `.time`: a package with one release a year publishing twice in a day is a `2`
  even with a familiar account, because account takeover presents exactly this way.
- **Limit to state honestly**: the registry does not expose whether the account has 2FA, nor when a
  maintainer was added. A same-name maintainer does not prove the same human.

## Axis A — artifact versus source

npm's provenance and publish attestations are the strong form of this axis when present:

```sh
npm view "$PKG@$CAND" --json dist | jq '.attestations'
npm view "$PKG@$BASE" --json dist | jq '.attestations'
```

```sh
# verifies signatures/attestations for an installed tree; run it in a throwaway dir, scripts off
mkdir -p "$WORK/verify" && cd "$WORK/verify"
npm init -y >/dev/null
npm install --ignore-scripts --no-audit --no-fund "$PKG@$CAND" >/dev/null 2>&1
npm audit signatures
cd "$WORK"
```

- Provenance names the source repo, workflow, and commit SHA that built the artifact. When it is
  present, follow it: confirm that commit exists upstream and that the repo tag for `$CAND` points
  at it. That closes the axis at `0`.
- **Absence of provenance is not itself a red flag** — most packages still publish without it. Score
  the *regression*: baseline had attestations and the candidate does not is a `3`, because the
  publish pipeline changed underneath a package that had already adopted it.
- With no provenance on either side, the tarball-versus-repo comparison under axis D below is the
  substitute. If the package has no public repo, or the repo has no tag for `$CAND`, A is `?`.

## Axis D — what actually changed

`npm pack` on a registry spec downloads the published tarball without running scripts:

```sh
npm pack --silent "$PKG@$BASE" >/dev/null && npm pack --silent "$PKG@$CAND" >/dev/null
for f in *.tgz; do mkdir -p "ex/${f%.tgz}" && tar -xzf "$f" -C "ex/${f%.tgz}"; done
diff -ru ex/*"$BASE"/package ex/*"$CAND"/package | head -400
```

Read in this order, because it is the order of increasing effort and decreasing likelihood:

1. **`package.json` first** — the highest-signal file per byte:

   ```sh
   diff <(jq -S '{scripts,bin,files,dependencies,main,exports,type}' ex/*"$BASE"/package/package.json) \
        <(jq -S '{scripts,bin,files,dependencies,main,exports,type}' ex/*"$CAND"/package/package.json)
   ```

   Any new `preinstall` / `install` / `postinstall` on a package that had none is a `3` on its own.

2. **Grep the candidate for the signature set** before reading prose:

   ```sh
   grep -rnE "child_process|execSync|eval\(|new Function|atob\(|Buffer\.from\([^)]*base64|process\.env|\.npmrc|curl |wget |https?://[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+" ex/*"$CAND"/package --include='*.js' --include='*.mjs' --include='*.cjs' --include='*.ts' | head -60
   ```

   Compare hits against the baseline — a pattern present in both is the package's normal business;
   one that appears only in the candidate is the finding.

3. **Binary and packed content**, which a text diff will not show usefully:

   ```sh
   diff -rq ex/*"$BASE"/package ex/*"$CAND"/package | grep -iE "differ|Only in" | head -40
   find ex/*"$CAND"/package -type f -size +100k -o -name '*.min.js' -o -name '*.node' -o -name '*.wasm'
   ```

   **When the package ships a bundler's content-hashed output** (`dist/chunks/foo-A1B2C3D4.mjs`), the
   two commands above degrade into noise: changing one shared chunk re-hashes most filenames, so
   `diff -rq` emits hundreds of `Only in` lines that say nothing. Strip the hash and compare the
   *sets* instead — an added, removed, or renamed logical file is the finding, and identical sets
   retire that whole class of question in one command:

   ```sh
   norm() { find "$1" -type f | sed "s#^$1/##" | sed -E 's/-[A-Za-z0-9]{8}\.(mjs|js)/-HASH.\1/' | sort; }
   diff <(norm ex/*"$BASE"/package) <(norm ex/*"$CAND"/package)
   ```

   Do **not** then pair the hashed files up to diff them individually — the mapping is ambiguous by
   construction, and joining on the normalized name explodes combinatorially (many chunks share the
   name `chunk-HASH.mjs`). Two other moves carry the axis instead. First, run step 2's signature grep
   over both trees and compare the **hit profile** rather than the file layout: a pattern appearing
   only in the candidate is the finding, and an identical profile is strong evidence the bundle
   gained no new capability. Second, confirm the bundle carries the change the release *claims*, by
   grepping for a token the source diff says should appear or disappear — a removed option name, a
   renamed export, a new guard. A bundle that does not contain its own advertised fix, or that moved
   while the source did not, is the highest-signal finding available; treat it as `3`.

4. **Tarball versus repository** — this is where "does the artifact match its source" gets real teeth
   for a package without provenance. Clone the upstream tag and compare the published files against
   it. Files present in the tarball but absent from the tagged source are the classic vector.

   ```sh
   REPO=$(jq -r '.repository.url // .repository' packument.json | sed 's#^git+##; s#\.git$##')
   git clone --depth 1 --branch "v$CAND" "$REPO" src 2>/dev/null || \
     git clone --depth 1 --branch "$CAND" "$REPO" src
   diff -rq src ex/*"$CAND"/package | grep 'Only in ex' | head -40
   ```

   Expect build outputs (`dist/`, `lib/`) to appear only in the tarball — that is normal. What is
   not normal is a source file, a script, or a blob with no counterpart upstream. If the package
   ships a bundle, a bundle change with no matching source change is the highest-signal finding
   available; treat it as `3`.

## Axis S — dependency and capability surface

```sh
diff <(jq -S '.dependencies // {}' ex/*"$BASE"/package/package.json) \
     <(jq -S '.dependencies // {}' ex/*"$CAND"/package/package.json)
```

For each newly added dependency, check whether it is itself the payload — a fresh, thin, or
name-adjacent package is how a malicious release smuggles code past a diff review of the parent:

```sh
NEW=<new-dep>
curl -fsSL "https://registry.npmjs.org/$NEW" | jq '{created: .time.created, versions: (.versions|length), repo: .repository.url, maintainers: [.maintainers[].name]}'
curl -fsSL "https://api.npmjs.org/downloads/point/last-week/$NEW" | jq '.downloads'
```

A dependency created days ago, with a handful of versions and no repository, added by a mature
package, is a `3`. Also diff `bin`, `files`, and `directories` — a new `bin` entry means the package
can now be invoked as a command.

## Applies to pnpm too, with one difference

`scripts/` and `docs-viewer/` resolve with pnpm, but every command above works unchanged: pnpm pulls
from the same registry, so the packument, the attestations, and `npm pack` all address the same
artifact. Only the reporting differs, below.

## Reporting: what a LOW score can actually buy

State in the report which of these applies, because a score is evidence, not permission, and the
options it unlocks are not the same in the two ecosystems:

- **Under a `.npmrc` `min-release-age`** (this repo: the lockfile dirs' own `.npmrc`), the version
  cannot be resolved at all until it ages — `npm install` fails with
  `ETARGET … with a date before <cutoff>`. A LOW score does not unlock it, and npm has **no
  per-version escape hatch**. The honest options are wait, take an older aged fixed version, or a
  deliberate role-level override that `make npm-cooldown-audit` will surface on the PR. Never
  propose lowering `min-release-age`.
- **Under a pnpm `minimumReleaseAge`** (`pnpm-workspace.yaml`, stated in minutes), resolution fails
  with `ERR_PNPM_NO_MATURE_MATCHING_VERSION` and — because pnpm re-verifies the whole lockfile on
  every install, `--frozen-lockfile` included — replaying such an entry fails with
  `ERR_PNPM_MINIMUM_RELEASE_AGE_VIOLATION`. pnpm **does** have a per-version escape hatch,
  `minimumReleaseAgeExclude`, which is exactly the decision a triage report exists to inform. Report
  the score and say plainly that adding the entry is the caller's or the user's call, never yours.
  Never propose lowering `minimumReleaseAge` or flipping `minimumReleaseAgeStrict` to `false` — the
  first opens the window for every dependency at once, the second hands the decision to the
  resolver, which then writes the exclusion itself. There is no pnpm counterpart to
  `npm-cooldown-audit` and none is missing: the exemption must be committed to a tracked,
  CODEOWNERS-covered file for the install to work at all, so it is reviewed rather than detected.
- **Transitive candidates** reached through an `overrides` floor: the version that will actually
  resolve is the newest in-range version the resolver accepts, so triage the version the lockfile
  would land on, not the advisory's floor.
