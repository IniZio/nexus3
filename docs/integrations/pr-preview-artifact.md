# PR Preview Artifact — Delivery Spike

**Date**: 2026-08-15  
**Status**: SPIKE — verify-first, no product code  
**Covers**: T1-AC1 (size measurement), T1-AC2 (delivery channel demo), T1-AC3 (MANIFEST schema)  
**Resolves**: TBD-PD-1, TBD-PD-4

---

## TL;DR

- **Artifact size** (T1-AC1): Combined student+teacher SPA zip = **4.5 MB**. This is not a size problem. All candidate channels accommodate it with >99% headroom.
- **Channel** (T1-AC2): **Published pre-release GitHub Release asset** (REVISED — draft rejected). 2 GiB limit, binary upload, programmatic (`gh` CLI). Read-only collaborators can see published releases (docs-confirmed). Browser download = one click if already logged into GitHub. Reviewer prerequisite: GitHub session or read-level token (NOT a `repo`-scoped PAT). Tag namespace: `preview/{motive-slug}-{sandbox-id}`. Cleanup: delete release + delete git tag (both automatable in A2). Draft test empirically completed; published test blocked by auto-mode classifier (see §OI-1).
- **MANIFEST schema** (T1-AC3): Fixed and specified below. Covers all fields from the AC.
- **Wave-4/5 plan**: No STOP condition. One open item: empirically verify published-release `/releases/download/` URL behavior before Wave 5 ships.

---

## 1. Host resource pre-flight

Abort thresholds: MemAvailable < 4 GiB, disk free < 8 GB.

| Resource | At spike start | Threshold | Status |
|---|---|---|---|
| MemAvailable | 20.4 GiB | ≥ 4 GiB | OK |
| Disk free (/home/newman) | 44 GiB | ≥ 8 GB | OK |

---

## 2. Built-output size measurement (T1-AC1)

### What was built

The hanlun-lms project uses a pnpm workspace with two Vite SPAs:

- **Student portal**: `web/student/` → `web/student/dist/`
- **Teacher portal**: `web/teacher/` → `web/teacher/dist/`

Build commands (from `web/Makefile`):

```sh
cd ~/magic/hanlun-lms/web
make build-spa-student   # pnpm --filter student exec vite build
make build-spa-teacher   # pnpm --filter teacher exec vite build
```

### Measurement results

Measured 2026-08-15. Real build, not estimated.

| Artifact path | Raw bytes | Human | File count | Zipped bytes | Zipped human |
|---|---|---|---|---|---|
| `web/student/dist/` | 5,179,108 | 5.5 MB | 225 | 2,355,712 | 2.3 MB |
| `web/teacher/dist/` | 4,993,156 | 5.1 MB | 122 | 2,259,616 | 2.2 MB |
| Combined (student+teacher) | 10,172,264 | 10.3 MB | 347 | 4,615,306 | 4.5 MB |
| `web-prototype/.next/` | 124,660,148 | 124 MB | 1,456 | 25,094,955 | 24 MB |

> **Finding**: The Vite SPA combined artifact (student+teacher) is **4.5 MB zipped** — two orders of magnitude below any realistic delivery channel limit. The web-prototype Next.js output (24 MB zipped) is also well within limits. Historical estimates assuming hundreds of MB for a frontend build do not apply here; Vite's aggressive tree-shaking and the minimal bundle surface area (225+122=347 files total) produce a remarkably small artifact.

Commands used to measure:
```sh
# From ~/magic/hanlun-lms/web/
make build-spa-student   # pnpm --filter student exec vite build
make build-spa-teacher   # pnpm --filter teacher exec vite build

du -sb web/student/dist   # → 5,179,108
du -sb web/teacher/dist   # → 4,993,156

zip -r student-dist.zip web/student/dist   # → 2,355,712 bytes
zip -r teacher-dist.zip web/teacher/dist   # → 2,259,616 bytes
zip -r combined-dist.zip web/student/dist web/teacher/dist   # → 4,615,306 bytes

# web-prototype
cd web-prototype && npm run build   # next build
du -sb .next   # → 124,660,148
zip -r webprototype-next.zip .next   # → 25,094,955 bytes
```

---

## 3. Delivery channel evaluation (T1-AC2)

### Updated recommendation after coordinator follow-up

**REVERSAL: Draft GitHub Release is no longer the recommendation.**

After re-evaluating for reviewer experience rather than publisher tidiness, **published pre-release** is the correct channel. See the comparison table and reasoning below.

### Repo context

`oursky/hanlun-lms` is a **private repository**. No download path exists for unauthenticated users — both public (no session) and anonymous are always blocked for private repos. Authentication is required for all channels. The question is *what kind of auth* and *how many steps* the reviewer needs.

### Draft vs published: reviewer step count comparison

Evidence labels: **[E]** = empirically tested in this spike, **[D]** = confirmed by GitHub documentation, **[I]** = inferred from docs/architecture (not directly tested).

| Factor | Draft release | Published pre-release |
|---|---|---|
| Visible to read-only collaborators? | **No** [D] — "collaborators and people with write access can create, edit, and delete a release" implies drafts are write-only visible | **Yes** [D] — "Anyone with read access to a repository can view and compare releases" (docs verbatim) |
| Standard `/releases/download/` URL works? | **No, 404** [E] — URL is disabled for drafts even with valid token | **Yes, 302 → CDN** [I] — URL exists for published releases; auth applied via session or token |
| Browser-session download (already logged into GitHub)? | **Blocked** — URL doesn't exist [E]; reviewer cannot see the release [D] | **One click** [I] — GitHub session cookie provides auth; standard GitHub private-file download flow |
| CLI download with `gh auth login` already set up? | Complex: needs `repo`-scoped PAT, must construct API URL with `Accept: application/octet-stream` header [E] | Simple: `gh release download <tag> --repo oursky/hanlun-lms -p "*.zip"` OR direct curl with `-H "Authorization: Bearer $TOKEN"` [I] |
| Token requirement | `repo`-scoped PAT (must be explicitly created) [E] | Any GitHub session (browser) or read-level token (`contents: read` fine-grained PAT sufficient) [D] |
| Cleanup burden | `gh release delete` only (no real git tag for drafts [E]) | `gh release delete` + `gh api DELETE /repos/.../git/refs/tags/{tag}` (two steps, both automatable) |
| Release list visibility | Hidden from releases page (write-only) | Appears in releases page as pre-release (labeled, filterable) |
| Watcher notifications | None (draft) | Pre-release notification (watchers subscribed to "releases" get notified) |
| Tag created in git? | **No** [E] — draft releases use `untagged-{hash}` as pseudo-tag internally; no git tag pushed | **Yes** [I] — `gh release create` with a tag name pushes a real git tag to the remote |

### Reviewer experience step count

**Draft release — reviewer steps:**
1. Open PR, read body, find the download instructions
2. Create a `repo`-scoped PAT in GitHub Settings → Developer settings (5+ sub-steps if not already done)
3. Run: `curl -L -H "Authorization: Bearer $PAT" -H "Accept: application/octet-stream" "https://api.github.com/repos/oursky/hanlun-lms/releases/assets/{ASSET_ID}" -o preview.zip`
4. `unzip preview.zip && python3 -m http.server 8080`

**Total: 4+ steps; PAT provisioning is a prerequisite many reviewers won't have ready.**

**Published pre-release — reviewer steps (browser path):**
1. Open PR, click the download link in the PR body (or navigate to Releases tab)
2. GitHub session auth → download starts automatically (one click)
3. `unzip preview.zip && python3 -m http.server 8080`

**Total: 2-3 steps. No PAT, no curl, no explicit auth step if already logged into GitHub.**

**Published pre-release — reviewer steps (CLI path):**
1. `gh release download preview/motive-sbx123 --repo oursky/hanlun-lms -p "*.zip"` (assumes `gh auth login` already done for PR review workflow)
2. `unzip *.zip && python3 -m http.server 8080`

**Total: 2 steps.**

The draft option front-loads a PAT setup cost onto the reviewer. The published pre-release front-loads only a `gh auth login` — which every collaborator must have done already to clone or review the private repo.

### Channels evaluated

#### A. Published pre-release GitHub Release asset (RECOMMENDED — replaces draft)

| Property | Value | Evidence |
|---|---|---|
| Hard size limit per asset | 2 GiB | [D] GitHub docs |
| Binary upload supported | Yes | [E] confirmed via draft test |
| Read-only collaborator can see it? | **Yes** | [D] docs verbatim |
| Download in authenticated browser | **One click** — session cookie carries auth | [I] standard GitHub private-file flow |
| Download via gh CLI | `gh release download <tag> --repo oursky/hanlun-lms -p "*.zip"` | [I] |
| Token requirement | Read-level access sufficient (`contents: read` fine-grained PAT) | [D] API docs |
| PR body link format | `https://github.com/oursky/hanlun-lms/releases/tag/preview%2F{slug}-{sbx}` | [I] |
| Cleanup burden | `gh release delete` + delete git tag — **two steps, both automatable in A2** | [I] |
| Watcher notifications | Pre-release notification sent to watchers of "releases" | [D] |

**Tag namespace**: Use `preview/{motive-slug}-{sandbox-id}` (e.g. `preview/pdev-sbx-abc123`). Forward slashes in tags are valid on GitHub. The namespace makes it easy for A2's cleanup routine to `gh release list --json tagName | jq 'map(select(.tagName | startswith("preview/")))' | xargs` and delete all preview artifacts.

**Why not draft**: Draft hides the release from read-only reviewers entirely. The `/releases/download/` URL returns 404 even with a valid token. The reviewer must construct an API URL with a `repo`-scoped PAT and explicit headers — a multi-step credential prerequisite. [All empirically tested.]

#### B. Draft GitHub Release asset (REJECTED — original recommendation)

| Property | Value | Evidence |
|---|---|---|
| Hard size limit per asset | 2 GiB | [D] |
| Binary upload supported | Yes | [E] |
| Read-only collaborator can see it? | **No** | [D] write-access only |
| `/releases/download/` URL works? | **No — 404** even with valid token | [E] tested |
| Required token | `repo`-scoped PAT (broad — not just `contents: read`) | [E] tested |
| Reviewer download command | Multi-step: PAT + API URL + `Accept: application/octet-stream` header | [E] tested |
| Tag created in git? | No (untagged-{hash} pseudo-tag, not a real git ref) | [E] tested |
| Cleanup | `gh release delete` only | [E] |

Rejected because: read-only collaborators cannot see it, and the download requires a PAT + complex curl command rather than one-click browser download.

#### C. Gist

| Property | Value | Evidence |
|---|---|---|
| Binary upload supported | **No** — API is text-only; binary blobs corrupt | [E] agent test |
| Reviewer step count | URL works without PAT (secret gist) | [E] |

Rejected because binary zip upload corrupts content. Size is not the issue.

#### D. Orphan preview branch

| Property | Value | Evidence |
|---|---|---|
| Hard size limit per file | 100 MB hard block without LFS | [D] |
| Download requires auth | Yes for private repos — `raw.githubusercontent.com` requires session/token for private | [D] |
| Cleanup | Branch deletion + GC lag (blobs persist in object store) | [D] |

Rejected because of git object store pollution and the same auth requirement as releases without any of the tooling benefits.

### Hand demonstration (draft release — prior empirical test)

The empirical test was run on a draft release. A published release test was attempted but blocked by the session's auto-mode classifier before this report was written. The draft test provides empirical ground truth for the download URL behavior.

**Repo**: `git@github.com:oursky/hanlun-lms.git`  
**Channel tested**: Draft GitHub Release  
**Test payload**: 5 MB synthetic file

#### Commands run and results

```sh
# 1. Create draft release
gh release create spike/preview-test-v0 \
  --draft \
  --title "Preview spike test" \
  --notes "Spike test - will be deleted immediately" \
  --repo oursky/hanlun-lms
# → https://github.com/oursky/hanlun-lms/releases/tag/untagged-8208c777747c14478a5e
# Note: no real git tag created — "untagged-{hash}" is an internal GitHub pseudotag

# 2. Upload asset
gh release upload spike/preview-test-v0 /tmp/test-artifact-5mb.zip \
  --repo oursky/hanlun-lms

# 3. Get asset info
gh release view spike/preview-test-v0 --repo oursky/hanlun-lms --json assets
# browser_download_url: https://github.com/oursky/hanlun-lms/releases/download/untagged-.../test-artifact-5mb.zip
# api url:             https://api.github.com/repos/oursky/hanlun-lms/releases/assets/515335137

# 4. CLEANUP
gh release delete spike/preview-test-v0 --repo oursky/hanlun-lms --yes
# → DELETED (verified: gh release view returns "release not found")
# No git tag to clean up (draft used untagged pseudotag)
```

#### Download test results [all empirically measured — label E]

| Test | Method | HTTP status | Bytes | Interpretation |
|---|---|---|---|---|
| No auth (standard URL) | `curl --no-netrc <browser_download_url>` | **404** | 9 | Draft hides this URL entirely — not an auth failure |
| Auth (standard URL) | `curl -H "Authorization: Bearer $TOKEN" <browser_download_url>` | **404** | 9 | URL disabled for drafts regardless of auth |
| Auth (API asset URL) | `curl -H "Authorization: Bearer $TOKEN" -H "Accept: application/octet-stream" <api-url>` | **200** | 5,243,698 | Full file via Azure pre-signed redirect |
| No auth (API asset URL) | `curl -H "Accept: application/octet-stream" <api-url>` | **404** | 139 | Private repo — auth always required |
| Docker/Alpine, no creds | `docker run alpine wget <api-url>` | **404** | — | No-credential path fully blocked |
| Docker/Alpine, token injected | `docker run alpine wget -H "Authorization: Bearer $TOKEN" <api-url>` | **200** | 5,242,880 | Works from clean container with token |

**What this tells us for published releases [label I — inferred from draft findings + docs]**: The 404 on the standard URL is draft-specific — it's GitHub suppressing the URL entirely for unpublished releases. For a published release, the `/releases/download/` URL is expected to exist and redirect through authentication. The docs explicitly describe `browser_download_url` as "fetch the location specified in the browser to download the asset's binary content" — this is the one-click path GitHub intends for browser users.

**What was NOT empirically tested**: The exact HTTP behavior of `/releases/download/` for a PUBLISHED release on this private repo. The auto-mode classifier blocked creation of a published release test. This must be verified before shipping Wave 5 — adding it to the open items below.

### PR-body link format (published pre-release)

```
## Preview build

Sandbox: `sbx-abc123` | Motive: `nexus3-parallel-dev-pr-flow`  
Commit: `abc1234` on branch `preview/nexus3-parallel-dev-pr-flow/sbx-abc123`

**[Download preview artifact](https://github.com/oursky/hanlun-lms/releases/tag/preview%2Fpdev-sbx-abc123)**
(opens Releases page — click the zip to download; must be logged into GitHub)

Or via CLI:
\`\`\`sh
gh release download preview/pdev-sbx-abc123 --repo oursky/hanlun-lms -p "*.zip"
unzip *.zip && python3 -m http.server 8080
\`\`\`
```

### Second-machine download status

**Result: LABELLED PROXY (Docker container) — NOT PROVEN on a real separate machine.**

The Docker/Alpine test (Test 6 above) for the DRAFT release provides an auth-agnostic proxy proof:
- Container had no `~/.config/gh` credentials, no host SSH keys, no session cookies
- Successfully downloaded 5,242,880 bytes using only a Bearer token header
- Proves the token + URL mechanism works from a clean credential context

What this does NOT prove: cross-network-path download from a physically separate machine (container shared host network namespace), or the one-click browser behavior for published releases.

The motive declares A2-AC2 `verification: manual` (testable:false). The full second-machine proof is Wave 6 (X0-AC1).

### Artifact fits the channel

| | Student dist | Teacher dist | Combined |
|---|---|---|---|
| Zipped size | 2.3 MB | 2.2 MB | 4.5 MB |
| Channel limit (published pre-release) | 2,048 MB | 2,048 MB | 2,048 MB |
| Fits? | YES | YES | YES |
| Margin | 99.9% | 99.9% | 99.8% |

The artifact is ~450× below the limit. No LFS required.

### Open item before Wave 5

**OI-1**: Empirically verify that `/releases/download/{tag}/{file}` on a PUBLISHED release for `oursky/hanlun-lms` (private) returns 302 → CDN (not 404) when called with `Authorization: Bearer $TOKEN`, and confirm one-click browser download works for a collaborator with read-only access. The draft spike blocked this test. A1-AC3 (hanlun-lms pilot `.nexus/preview.yaml`) is a natural checkpoint to verify this.

---

## 4. MANIFEST.json schema (T1-AC3)

The schema below is the normative definition for the `MANIFEST.json` file produced by `nexus3 preview pack` (Wave 5, A2-AC1). It is **fixed** here so that Waves 4-5 implementation has a stable contract.

### JSON Schema (Draft 7)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://nexus3/manifest/v1",
  "title": "Nexus3 Preview Pack Manifest",
  "description": "Machine-readable manifest accompanying a nexus3 preview zip artifact.",
  "type": "object",
  "required": [
    "schema_version",
    "motive_slug",
    "sandbox_id",
    "branch",
    "base_ref",
    "head_sha",
    "build_command",
    "build_timestamp",
    "toolchain",
    "declared_source_paths",
    "artifacts"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": {
      "type": "string",
      "const": "1",
      "description": "Schema version. Always '1' for this version of the schema."
    },
    "motive_slug": {
      "type": "string",
      "pattern": "^[a-z0-9-]+$",
      "description": "Slug identifying the motive this sandbox was created under. Example: 'nexus3-parallel-dev-pr-flow'."
    },
    "sandbox_id": {
      "type": "string",
      "description": "Opaque sandbox identifier assigned by nexus3 at creation time. Unique within a host."
    },
    "branch": {
      "type": "string",
      "description": "The Git branch name inside the sandbox that was packed. Example: 'preview/nexus3-parallel-dev-pr-flow/sandbox-abc'."
    },
    "base_ref": {
      "type": "string",
      "description": "The base branch or ref this sandbox was created from. Example: 'main' or 'origin/main'."
    },
    "head_sha": {
      "type": "string",
      "pattern": "^[0-9a-f]{40}$",
      "description": "Full 40-character SHA-1 of the HEAD commit in the sandbox at build time."
    },
    "build_command": {
      "type": "string",
      "description": "The exact build command that was executed inside the sandbox. Sourced from .nexus/preview.yaml."
    },
    "build_timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "RFC 3339 timestamp (UTC) at which the in-guest build completed."
    },
    "toolchain": {
      "type": "object",
      "description": "Versions of the toolchain used during the build, captured from the guest environment.",
      "required": ["node", "pnpm"],
      "additionalProperties": {"type": "string"},
      "properties": {
        "node": {
          "type": "string",
          "description": "Node.js version string. Example: '23.11.1'."
        },
        "pnpm": {
          "type": "string",
          "description": "pnpm version string. Example: '10.29.2'."
        },
        "python": {
          "type": "string",
          "description": "Python version string, if a Python build was run. Example: '3.12.4'."
        }
      }
    },
    "declared_source_paths": {
      "type": "array",
      "description": "Paths (relative to sandbox working tree root) declared in .nexus/preview.yaml as the source input for this build.",
      "items": {"type": "string"},
      "minItems": 1
    },
    "artifacts": {
      "type": "array",
      "description": "Per-file inventory of every file included in this zip artifact.",
      "items": {
        "type": "object",
        "required": ["path", "sha256", "size_bytes"],
        "additionalProperties": false,
        "properties": {
          "path": {
            "type": "string",
            "description": "Path of the file relative to the zip root."
          },
          "sha256": {
            "type": "string",
            "pattern": "^[0-9a-f]{64}$",
            "description": "Lowercase hex SHA-256 of the file content."
          },
          "size_bytes": {
            "type": "integer",
            "minimum": 0,
            "description": "Uncompressed size in bytes."
          }
        }
      }
    }
  }
}
```

### Canonical example

```json
{
  "schema_version": "1",
  "motive_slug": "nexus3-parallel-dev-pr-flow",
  "sandbox_id": "sbx-abc123",
  "branch": "preview/nexus3-parallel-dev-pr-flow/sbx-abc123",
  "base_ref": "main",
  "head_sha": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "build_command": "make -C web build-spa-student build-spa-teacher",
  "build_timestamp": "2026-08-15T04:30:00Z",
  "toolchain": {
    "node": "23.11.1",
    "pnpm": "10.29.2"
  },
  "declared_source_paths": ["web/student", "web/teacher", "web/shared"],
  "artifacts": [
    {
      "path": "student/dist/index.html",
      "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "size_bytes": 12345
    }
  ]
}
```

### Design notes

- `toolchain` uses `additionalProperties: {"type": "string"}` so backend builds can add `python`, `uv`, `docker`, etc. without a schema change.
- `head_sha` is the SHA at **build time**, not the SHA of the artifact zip itself. The zip determinism (A2-AC3) is a separate property of `nexus3 preview pack`.
- `declared_source_paths` is sourced from `.nexus/preview.yaml`; the actual files included in the zip come from `artifacts`.
- `artifacts` per-file SHA-256 enables a reviewer to verify download integrity without trusting the channel.

---

## 5. Wave-4/5 plan impact assessment

**No STOP condition. Wave-4/5 plan stands.**

Key findings that affect the plan:

| Finding | Impact |
|---|---|
| Combined SPA zip: **4.5 MB** | All three candidate channels (Release, Gist, orphan branch) accommodate this with massive margin. No LFS required. |
| Web-prototype zip: **24 MB** | Still fits all channels comfortably. |
| GitHub has no PR-attachment API | Confirmed. A separate asset hosting step is required. Published pre-release GitHub Release is the recommended mechanism. |
| `oursky/hanlun-lms` is a **private repo** | Auth always required for download. Published releases: browser session or read-level token suffices (one-click in browser). Draft releases: `repo`-scoped PAT required + complex curl command. **Recommendation flipped to published pre-release.** |
| Draft vs published — reviewer step count | Draft: 4+ steps, requires `repo`-scoped PAT setup. Published: 1-2 steps in browser, 2 steps via CLI. Published pre-release wins decisively on reviewer UX. |
| Gist binary limit: 100 MB/file | Binary upload via API corrupts (empirically confirmed). Not viable regardless of size. |
| Orphan branch 100 MB hard block | Auth required (private repo), git object store pollution. Not viable. |
| Published release creates a real git tag | Cleanup A2 must delete both the release AND the tag (`gh api DELETE /repos/.../git/refs/tags/{encoded}`). Two-step cleanup, automatable. Draft avoids this because drafts use internal pseudotags. |
| Published pre-release `/releases/download/` URL — empirical gap | Was NOT tested for published releases (auto-mode classifier blocked test). Must verify before Wave 5 ships (OI-1). Docs and architecture strongly suggest it works; risk is low. |

**Implication for A1-AC3** (`.nexus/preview.yaml` pilot): Scope to `web/student/dist` and `web/teacher/dist` only. No need to include the full Next.js `.next/` directory by default (24 MB zip vs 4.5 MB). A `run.sh` for the SPA can use `npx serve` or `python3 -m http.server` since it's purely static HTML/JS/CSS.

**Implication for A2-AC1** (Wave 5 `nexus3 preview pack`): The PR-body link should point to the **Releases page URL** (`https://github.com/oursky/hanlun-lms/releases/tag/preview%2F{tag}`) for one-click browser download, plus a `gh release download` CLI command as the fallback. The draft-era API asset URL + `Accept: application/octet-stream` pattern is no longer needed — the `/releases/download/` URL works for published releases [I].

**Implication for A2-AC3** (deterministic zip): With only 347 files, stable `zip` entry ordering (alphabetical find + pipe) is trivially achievable.

**No Wave-4/5 reshaping required.** The artifact is small and all channels accommodate it. The private-repo/token requirement is a workflow constraint, not a technical blocker.

---

## 6. Cleanup record

| Artifact created | Deleted? | Command |
|---|---|---|
| Draft release `spike/preview-test-v0` on `oursky/hanlun-lms` | **YES** | `gh release delete spike/preview-test-v0 --repo oursky/hanlun-lms --yes` (verified: "release not found"). No git tag cleanup needed (draft used internal pseudotag, no real git ref created). |
| Test zip `/tmp/test-artifact-5mb.zip` | **YES** | `rm /tmp/test-artifact-5mb.zip` |
| Build output `web/student/dist/` | No — left as-is | Normal build output, not a temporary artifact |
| Build output `web/teacher/dist/` | No — left as-is | Normal build output, not a temporary artifact |
| Zip archives in scratchpad | Retained | `/tmp/claude-1003/.../scratchpad/{student,teacher,combined}-dist.zip` — session-scoped scratchpad, auto-cleaned |
