# Evidence collection — PyPI

Read this together with the axis definitions in `SKILL.md`. Work in the scratchpad, never in the
repo tree, and **never install the candidate**. An sdist's build backend executes on install, so
`pip install` / `uv pip install` is the attack surface itself; every command below downloads and
reads, and none of them builds.

Set up once:

```sh
PKG=<package>          # e.g. sqlfluff, graphifyy
CAND=<candidate>       # the version the window caught
BASE=<baseline>        # the version python/<tool>.in currently declares
WORK="$SCRATCH/pypi-$PKG"; mkdir -p "$WORK"; cd "$WORK"
curl -fsSL "https://pypi.org/pypi/$PKG/json" > pypi-project.json
```

The project metadata carries every release and its upload time, which answers cadence and the age
arithmetic without downloading anything.

## Axis P — publisher

**State the limitation first: PyPI does not expose who uploaded a release.** There is no `_npmUser`
equivalent — `info.author` and `info.maintainer` are self-declared metadata, frequently `null`, and
say nothing about the account that published. So P is answered *indirectly* or not at all.

```sh
# cadence: is this release in line with the project's own history?
jq -r '.releases | to_entries | .[] | "\(.key)\t\(.value[0].upload_time_iso_8601 // "?")"' pypi-project.json \
  | sort -V | tail -15
```

Two indirect routes, in order of strength:

1. **The attestation's publisher** (axis A below). When present it names the GitHub repository,
   workflow, and environment that built the artifact. Baseline and candidate publishing from the
   same trusted publisher is the strongest P evidence PyPI offers; a change of repository or
   workflow is a `3`.
2. **Cadence.** A project that ships weekly publishing twice in a day, or one dormant for a year
   suddenly releasing, is a `2` on its own — account takeover presents exactly this way.

With no attestation on either side and nothing unusual in the cadence, P is `?`, not `0`. Say so.

## Axis A — artifact versus source

PyPI's PEP 740 attestations are the strong form here. They are served from a separate endpoint, and
a `404` simply means the project does not publish them:

```sh
FN=$(curl -fsSL "https://pypi.org/pypi/$PKG/$CAND/json" \
      | jq -r '[.urls[] | select(.packagetype=="bdist_wheel")][0].filename')
curl -fsS -o prov-cand.json "https://pypi.org/integrity/$PKG/$CAND/$FN/provenance"
jq '.attestation_bundles[0].publisher' prov-cand.json     # kind / repository / workflow / environment
jq -r '.attestation_bundles[0].attestations[0].envelope.statement' prov-cand.json \
  | base64 -d | jq '{subject: [.subject[].name], predicateType}'
```

Repeat for `$BASE` and compare the `publisher` objects — that comparison is the finding, not the
mere presence of a bundle.

- **Absence of attestations is not itself a red flag**; most PyPI projects still publish without
  them. Score the *regression*: baseline attested and candidate not is a `3`, because the publishing
  pipeline changed underneath a project that had already adopted them.
- **The bundle does not hand you a source commit.** `publisher` binds the artifact to a repository
  and workflow; the exact commit lives in the Sigstore certificate extensions and needs
  `pypi-attestations` to extract. Unless you go that far, A closes at "same trusted publisher,
  same workflow" — which is worth a `0`, but say which question you actually answered.
- With no attestation on either side, the tarball-versus-repository comparison under axis D is the
  substitute. If the project has no public repository, or no tag matching `$CAND`, A is `?`.

## Axis D — what actually changed

Download the built wheel (a zip) and, when the project ships one, the sdist. Never build either.

```sh
for V in "$BASE" "$CAND"; do
  url=$(curl -fsSL "https://pypi.org/pypi/$PKG/$V/json" \
        | jq -r '[.urls[] | select(.packagetype=="bdist_wheel")][0].url')
  curl -fsSL -o "$PKG-$V.whl" "$url"
  mkdir -p "ex/$V" && (cd "ex/$V" && unzip -oq "../../$PKG-$V.whl")
done
diff -ru "ex/$BASE" "ex/$CAND" | head -400
```

Read in this order:

1. **Install-time execution paths**, which are Python's distinctive vector. A wheel does not execute
   on install, an sdist does — so a candidate that suddenly ships only an sdist, or whose sdist
   grows a `setup.py` where the baseline had none, is the highest-signal shape available:

   ```sh
   curl -fsSL "https://pypi.org/pypi/$PKG/$CAND/json" | jq -r '[.urls[].packagetype]'
   curl -fsSL -o cand.tar.gz "$(curl -fsSL "https://pypi.org/pypi/$PKG/$CAND/json" \
        | jq -r '[.urls[] | select(.packagetype=="sdist")][0].url')"
   tar -tzf cand.tar.gz | grep -E 'setup\.py|setup\.cfg|pyproject\.toml'
   ```

   Read any `setup.py` and any `cmdclass` / `build_py` / `install` override it declares. Code there
   runs before anyone reviews it. Also read `__init__.py` for import-time side effects — the second
   Python-specific vector, since importing is enough to trigger it.

2. **Grep the candidate for the signature set** before reading prose, and compare the **hit profile**
   against the baseline rather than the file layout:

   ```sh
   for V in "$BASE" "$CAND"; do
     printf "%s: " "$V"
     grep -rhoE "subprocess|os\.system|eval\(|exec\(|__import__|base64\.b64decode|socket\.|urllib\.request|requests\.(get|post)|os\.environ|\.pypirc|~/\.ssh" \
       "ex/$V" --include='*.py' 2>/dev/null | sort | uniq -c | sort -rn | tr '\n' ' '
     echo
   done
   ```

   A pattern present in both is the project's normal business; one that appears only in the
   candidate is the finding. An identical profile is strong evidence no new capability appeared.

3. **Compiled and packed content** a text diff will not show — `.so` / `.pyd` extension modules,
   vendored blobs, anything large:

   ```sh
   diff -rq "ex/$BASE" "ex/$CAND" | grep -iE "differ|Only in" | head -40
   find "ex/$CAND" -type f \( -name '*.so' -o -name '*.pyd' -o -size +100k \)
   ```

   A pure-Python project that starts shipping a compiled extension is a `3` until the release notes
   account for it.

4. **Wheel versus repository**, where "does the artifact match its source" gets teeth for a project
   without attestations. Clone the upstream tag and look for files in the wheel with no counterpart
   upstream:

   ```sh
   REPO=$(jq -r '.info.project_urls.Source // .info.project_urls.Homepage // .info.home_page' pypi-project.json)
   git clone --depth 1 --branch "v$CAND" "$REPO" src 2>/dev/null || \
     git clone --depth 1 --branch "$CAND" "$REPO" src
   diff -rq src "ex/$CAND" | grep 'Only in ex' | head -40
   ```

   Generated metadata (`*.dist-info/`) appearing only in the wheel is normal. A `.py` module, a
   script, or a blob with no upstream counterpart is not.

## Axis S — dependency and capability surface

The wheel's own metadata carries both halves:

```sh
diff <(grep '^Requires-Dist:' "ex/$BASE"/*.dist-info/METADATA | sort) \
     <(grep '^Requires-Dist:' "ex/$CAND"/*.dist-info/METADATA | sort)
diff "ex/$BASE"/*.dist-info/entry_points.txt "ex/$CAND"/*.dist-info/entry_points.txt
```

A new `console_scripts` entry means the package can now be invoked as a command. For each newly
added dependency, check whether it is itself the payload:

```sh
NEW=<new-dep>
curl -fsSL "https://pypi.org/pypi/$NEW/json" \
  | jq '{created: (.releases | to_entries | map(.value[0].upload_time_iso_8601) | sort | .[0]),
         releases: (.releases|length), repo: .info.project_urls, summary: .info.summary}'
```

A dependency created days ago, with a handful of releases and no repository, added by a mature
project, is a `3`. Watch for name-adjacency to a well-known package — typosquatting is how a
malicious release smuggles code past a diff review of the parent.

**In this repository the resolved surface is bigger than `Requires-Dist`.** A PyPI tool's real
dependency tree lives in `python/<tool>.txt`, so when the caller is about to regenerate it, the
honest S evidence is the lockfile diff `make py-lock` would produce — every transitive package that
would newly enter, not just the direct requirements the wheel declares.

## PyPI-specific reporting

State which of these applies, because it decides what a LOW score can buy:

- **The window is enforced by a repository gate, not by the resolver.** `uv pip compile` will
  resolve a version published minutes ago without complaint; `scripts/tool-cooldown gate` is what
  refuses it, and unlike npm's audit-only counterpart it **fails the build**. So a blocked PyPI
  version is blocked by a check the repository owns, and a LOW score does not clear it on its own.
- **The escape hatch is `.github/tool-cooldown-bypass.toml`**, which takes
  `"<key>@<version>" = { expires, issue, reason }` with all three required, a maximum expiry three
  months out, and failure in *both* `gate` and `audit` once it expires or stops matching a
  declaration. A triage verdict is the evidence that belongs in `reason` and in the linked issue —
  it is not permission to add the entry. Adding one is the caller's or the user's call. Never
  propose editing the window constant in `scripts/tool-cooldown` instead.
- **Triage the version the lockfile will pin.** The declaration in `python/<tool>.in` is what the
  gate reads, but `python/<tool>.txt` is what gets installed, and an advisory may name a transitive
  package that appears only in the `.txt`. When it does, the baseline is that lockfile's current
  entry, and the candidate is what a regenerated lockfile would land on.
- **Exposure is developer-privilege, not service-runtime.** These tools run in the toolbox image and
  on workstations that hold repo write access and whatever credentials the shell has — report that
  line rather than "build/dev-time only", which understates it.
