---
title: "Configuration"
description: "Self-contained image build with --context and --dockerfile"
---

# Configuration

> Build a guest image from a Dockerfile at create time, without a separate build step.

The `--context <dir>` flag on `create` enables a self-contained build mode: nexus3 starts a buildkitd inside the VM, builds the image from the supplied Dockerfile context, and then boots the sandbox from the resulting image. No pre-built image reference is needed.

::: warning Implementation spelling <Badge type="warning" text="partial" />
The flag is built today as `--file`. The target spelling is `--context`. All examples on this page use the target spelling.
:::

## Self-contained build flags

These flags are part of `create`. See [Lifecycle commands](/cli/sandbox-commands) for the full flag set.

| Flag | Type | Default | Description |
|---|---|---|---|
| `--context <dir>` | string | — | Dockerfile build context directory on the host |
| `--dockerfile <path>` | string | `Dockerfile` | Dockerfile path, relative to `--context` |

`--context` and `--image` / `--rootfs` are mutually exclusive: exactly one image source must be provided.

## Example

```
nexus3 create myproject/dev-1 \
  --context /data/repos/myrepo \
  --memory 8192
```

<Badge type="warning" text="partial" /> — current implementation uses `nexus3 sandbox create` and `--file`; see [CLI sandbox commands](/cli/sandbox-commands) for the mapping.

nexus3 copies the context directory into the VM, runs `buildkitd`, builds the image, and boots the sandbox. The build cache is stored on a virtio-blk disk and reused across subsequent `--context` creates for the same project.

## What `--context` captures

The context directory is captured at create time. The capture includes:

- Tracked files (including dirty/modified tracked files)
- Untracked files
- Unpushed commits

It does not include `.git` history beyond what the working tree reflects.

## Project config file (`nexus3.yaml`)

`nexus3.yaml` is an optional per-repository configuration file. nexus3 discovers it by walking up from the process working directory to the nearest directory that contains a `.git` entry (the repository root). An absent file is a no-op. A present but malformed file, or a file with an unknown YAML key, is a hard error.

```yaml
version: 1

# Extend the outbound allowlist for all sandboxes in this repo.
egress:
  allow:
    - proxy.golang.org
    - sum.golang.org
    - storage.googleapis.com

  # Brokered VCS/API secrets — injected as 64-hex placeholders in the guest;
  # real token swapped host-side by the MITM proxy (PDF-R-020).
  secrets:
    # GitHub: repo: is mandatory (path policy guard — create is refused without it).
    - env: GH_TOKEN
      hosts: [github.com, api.github.com, uploads.github.com]
      repo: owner/name

    # Non-GitHub: no mandatory path policy.
    - env: GITLAB_TOKEN
      hosts: [gitlab.com]

    # Generic per-host path allowlist (any provider).
    - env: API_TOKEN
      hosts: [api.example.com]
      paths: ["/v4/projects/123/**"]

# Default sandbox settings for this repo.
sandbox:
  image: sha256:<digest>     # default --image; overridden by --image flag
  memory: 4096               # MiB; overridden by --memory flag
  vcpus: 2                   # overridden by --vcpus flag
  agent: claude-code         # default agent profile; overridden by --agent flag

  mounts:
    - ./src:/work/src        # relative paths resolved from the nexus3.yaml directory
```

Flag precedence: explicit CLI flags win over `nexus3.yaml` values; `nexus3.yaml` values win over built-in defaults.

`egress.allow` is **additive** — config hosts are unioned with `--allow-host` flags; neither replaces the other.

`sandbox.mounts` is **replaced** by any explicit `--mount` flag on the command line. To use both, list all mounts in `nexus3.yaml` and omit `--mount` on the command line.

### Trust anchor for worktree sandboxes

Worktree sandboxes (auto-created by the herdr plugin) read `nexus3.yaml` from `refs/remotes/origin/HEAD` — the operator's default branch — **not** from the agent's checked-out branch. A config present only on a feature branch grants nothing. The operator's merge to the default branch is the ratification act. See the [agent skill](https://github.com/hanlun-ai/nexus3/blob/main/skills/nexus3/SKILL.md) for the full propose → merge → ratify workflow.
