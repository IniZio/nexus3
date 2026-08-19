# Parallel-dev flow walkthrough — T0 honest tracer

> **Line references are date-fenced.** Sections dated 2026-08-15 and earlier cite
> `file.go:NNN` positions as they stood at that day's HEAD; several have since drifted
> and no longer resolve to the quoted code (verified 2026-08-19). Treat those citations
> as historical. Two have drifted in **substance**, not just position: `cli/shadowdisk.go`
> shadow disks are now **handle-namespaced** (`<safeHandle>.shadow.<name>.ext4`,
> `cli/shadowdisk.go:117`), so the earlier claims that shadow-disk paths carry no sandbox
> identity and that parallel boot is therefore impossible are **superseded** — see the
> resource-lifecycle inventory for current behaviour. `kernelPathFor` is now at
> `cmd_sandbox.go:1549`, not `:1189`. Sections dated 2026-08-19 were verified against
> HEAD on that date.


**Date:** 2026-08-15  
**Motive:** nexus3-parallel-dev-pr-flow  
**Target project:** ~/magic/hanlun-lms (git@github.com:oursky/hanlun-lms.git)  
**Executed by:** qa (T0 slice)  
**Status:** COMPLETE — all friction points captured; flow demonstrated with documented workarounds

---

## ⚠ HOST GITHUB AUTH VERDICT (TBR-PD-2 RESOLVED)

**HOST CAN PUSH. Wave-4 is not blocked by auth.**

```
$ gh auth status
github.com
  ✓ Logged in to github.com account IniZio (/home/newman/.config/gh/hosts.yml)
  - Active account: true
  - Git operations protocol: ssh
  - Token scopes: 'gist', 'read:org', 'repo', 'workflow'

$ ssh -T git@github.com
Hi IniZio! You've successfully authenticated, but GitHub does not provide shell access.

$ git remote -v  # (hanlun-lms)
origin  git@github.com:oursky/hanlun-lms.git (fetch)
origin  git@github.com:oursky/hanlun-lms.git (push)

$ git push --dry-run origin HEAD:nexus3/parallel-a-walkthrough-test
To github.com:oursky/hanlun-lms.git
 * [new branch]  HEAD -> nexus3/parallel-a-walkthrough-test
ok nexus3/parallel-a-walkthrough-test
```

Account: IniZio. SSH protocol. Token has `repo` scope. Push dry-run PASSED.  
A real PR was opened (draft) at https://github.com/oursky/hanlun-lms/pull/832 and immediately closed as part of the walkthrough. D-PD-01 is satisfied: the guest never held a GitHub credential.

---

## Full verbatim command transcript

### Baseline (2026-08-15 ~04:52 UTC)

```
$ go run ./cmd/nexus3 --json sandbox list
{"schema_version":1,"kind":"sandbox.list","data":{"sandboxes":[
  {"id":"sb-06FZ2S93T1ZDF68NK723QMW4R0","project":"~/magic/hanlun-lms","name":"whatever","handle":"~/magic/hanlun-lms/whatever","state":"created"},
  {"id":"sb-06FZ2SJVDHYPH6F25VC2A1S800","project":"~/magic/nexus","name":"nexus","handle":"~/magic/nexus/nexus","state":"created"},
  {"id":"sb-06FZZX7V8XZM12YE7VTR7T8168","project":"proof","name":"uni5","handle":"proof/uni5","state":"running"}
]}}

$ free -h
               total        used        free      shared  buff/cache   available
Mem:            30Gi       9.7Gi       6.0Gi        23Mi        15Gi        20Gi
Swap:          8.0Gi       7.5Gi       525Mi

$ df -h /home
Filesystem   Size  Used Avail Use%
/dev/mapper/ubuntu--vg-ubuntu--lv  466G  399G   44G  91%
```

Baseline: 3 sandboxes (pre-existing, not touched), 20 GiB RAM available, 44 GiB disk free.

---

### Step 1: First create attempt — BLOCKED (kernel not found)

```
$ go run ./cmd/nexus3 sandbox create hanlun-lms/parallel-a \
    --motive nexus3-parallel-dev-pr-flow \
    --workspace /home/newman/magic/hanlun-lms \
    --image nexus3-agent-base

2026/08/15 04:52:30 INFO auto-resize: hotplug hardware configured
2026/08/15 04:52:30 INFO workspace shadow disks prepared
    workspace_host=/home/newman/magic/hanlun-lms
    workspace_guest=/workspace/hanlun-lms
    num_shadow_disks=4
    workspace_device=/dev/vdf
    mounts="[{/dev/vdb /workspace/hanlun-lms/node_modules ext4 false false}
             {/dev/vdc /workspace/hanlun-lms/.next ext4 false false}
             {/dev/vdd /workspace/hanlun-lms/target ext4 false false}
             {/dev/vde /workspace/hanlun-lms/dist ext4 false false}
             {/dev/vdf /workspace/hanlun-lms ext4 false true}]"
2026/08/15 04:53:01 INFO workspace capture complete elapsed=28.257s shadow_dirs_excluded=true
error: sandbox create: service: create-and-boot hanlun-lms/parallel-a: boot: driver:
  cloudhypervisor: start sb-06G07GXBT1WCH7K73251CT91DC: cloudhypervisor: vm.boot:
  unexpected status 500: ["Error from API","The VM could not boot",
  "Cannot open kernel file","No such file or directory (os error 2)"]
exit status 1
```

**Root cause:** `kernelPathFor()` (`internal/cli/cmd_sandbox.go:1189`) resolves kernel as
`filepath.Join(filepath.Dir(os.Executable()), "images", "kernel", "vmlinux-x86_64")`.
Under `go run`, the executable is in a temp dir. The kernel sits at
`/home/newman/magic/nexus3/images/kernel/vmlinux-x86_64` but the temp path is never
populated. The workspace capture (28s, ~6.7 GiB actual data) completed and was wasted.
The sandbox record was auto-cleaned on boot failure.

**Escape:** `NEXUS3_KERNEL_PATH=/home/newman/magic/nexus3/images/kernel/vmlinux-x86_64`

---

### Step 2: Create sandbox 1 (parallel-a) with NEXUS3_KERNEL_PATH set

```
$ NEXUS3_KERNEL_PATH=/home/newman/magic/nexus3/images/kernel/vmlinux-x86_64 \
  go run ./cmd/nexus3 sandbox create hanlun-lms/parallel-a \
    --motive nexus3-parallel-dev-pr-flow \
    --workspace /home/newman/magic/hanlun-lms \
    --image nexus3-agent-base

2026/08/15 04:55:31 INFO auto-resize: hotplug hardware configured; governor activates in supervisor mem_max_mib=4096 vcpus_max=4 disk_max_gib=100
2026/08/15 04:55:31 INFO workspace shadow disks prepared workspace_host=/home/newman/magic/hanlun-lms workspace_guest=/workspace/hanlun-lms num_shadow_disks=4 workspace_device=/dev/vdf mounts="[{/dev/vdb /workspace/hanlun-lms/node_modules ext4 false false} {/dev/vdc /workspace/hanlun-lms/.next ext4 false false} {/dev/vdd /workspace/hanlun-lms/target ext4 false false} {/dev/vde /workspace/hanlun-lms/dist ext4 false false} {/dev/vdf /workspace/hanlun-lms ext4 false true}]"
2026/08/15 04:56:02 INFO workspace capture complete elapsed=27.529s shadow_dirs_excluded=true err=<nil>
created sandbox hanlun-lms/parallel-a (sb-06G07HKGMHTNSDD6J260KPAWBG)
```

Sandbox 1 created and running. ID: sb-06G07HKGMHTNSDD6J260KPAWBG.
Workspace capture: 27.5 seconds, shadow_dirs_excluded=true (node_modules/.next/target/dist excluded from workspace disk, served from separate shadow disks).

---

### Step 3: Create sandbox 2 (parallel-b) while sandbox 1 is running — BLOCKED

```
$ NEXUS3_KERNEL_PATH=/home/newman/magic/nexus3/images/kernel/vmlinux-x86_64 \
  go run ./cmd/nexus3 sandbox create hanlun-lms/parallel-b \
    --motive nexus3-parallel-dev-pr-flow \
    --workspace /home/newman/magic/hanlun-lms \
    --image nexus3-agent-base

2026/08/15 04:56:23 INFO auto-resize: hotplug hardware configured
2026/08/15 04:56:23 INFO workspace shadow disks prepared ...
2026/08/15 04:56:54 INFO workspace capture complete elapsed=27.604s shadow_dirs_excluded=true
error: sandbox create: service: create-and-boot hanlun-lms/parallel-b: boot: driver:
  cloudhypervisor: start sb-06G07HSTJSV138YRCMEE0WC124: cloudhypervisor: vm.boot:
  unexpected status 500: ["Error from API","The VM could not boot",
  "Error locking disk images: Another instance likely holds a lock",
  "Cannot lock images of all block devices",
  "Failed to get Write lock for disk image:
    /home/newman/.local/state/nexus3/disks/node_modules.shadow.ext4",
  "The file is already locked"]
exit status 1
```

**Root cause:** Shadow disk files (`node_modules.shadow.ext4`, `.next.shadow.ext4`, etc.) are
named by directory name only — no sandbox ID in the path (`cli/shadowdisk.go:111`):

```go
HostPath: filepath.Join(diskDir, safeName+".shadow.ext4")  // safeName = "node_modules"
```

CloudHypervisor holds exclusive write locks on these files for sandbox 1. Sandbox 2 cannot
boot because it needs the same global files. The workspace capture (28s) was again wasted.

**THIS IS THE PRIMARY BLOCKER FOR THE PARALLEL DEV USE CASE.**
Two sandboxes from the same workspace cannot run concurrently with today's primitives.
The M3 slice (`nexus3 up <motive> --count N`) must fix shadow disk naming before it is usable.

---

### Step 4: Explore workspace inside sandbox 1 — BLOCKED (workspace not mounted)

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-a \
    /bin/sh -c 'ls /workspace/ && cat /proc/mounts | grep vd'

cmd go.mod go.sum internal third_party   # ← nexus3 source, not hanlun-lms!
/dev/vda on / type ext4 (rw,relatime)    # only root disk mounted
```

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-a \
    /bin/sh -c 'cat /proc/cmdline'

root=/dev/vda rw init=/sbin/nexus3-agent console=ttyS0 memhp_default_state=online \
  memory_hotplug.online_policy=auto-movable -- \
  --workspace-mount=/dev/vdb:/workspace/hanlun-lms/node_modules:ext4:false:false \
  --workspace-mount=/dev/vdc:/workspace/hanlun-lms/.next:ext4:false:false \
  --workspace-mount=/dev/vdd:/workspace/hanlun-lms/target:ext4:false:false \
  --workspace-mount=/dev/vde:/workspace/hanlun-lms/dist:ext4:false:false \
  --workspace-mount=/dev/vdf:/workspace/hanlun-lms:ext4:false:true \
  --mem-ceiling=4294967296
```

The `--workspace-mount` args ARE in the kernel cmdline, and the disks ARE attached as
/dev/vdb–/dev/vdf. But the agent binary in the image ignores them:

```
$ ls -la /sbin/nexus3-agent (in guest)
-rwxr-xr-x 1 1003 1003 15897635 Aug 11 04:07 /sbin/nexus3-agent   # 15 MB, built Aug 11
```

```
$ ls -la /home/newman/magic/nexus3/images/kernel/nexus3-agent (on host)
-rwxrwxr-x 1 newman newman 35824388 Aug 14 13:32 nexus3-agent       # 35 MB, built Aug 14
```

**Root cause:** `nexus3-agent-base` was built 2026-08-11. The `--workspace-mount` arg parsing
was added to the agent after that date (worktree source-init landed 2026-08-13). The 15 MB
Aug-11 agent doesn't support `--workspace-mount`; the 35 MB Aug-14 agent does.
**`nexus3-agent-base` must be rebuilt before any workspace-mount sandbox can auto-mount.**

**Workaround for walkthrough purposes (manual step, NOT a nexus3 primitive):**

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-a \
    /bin/sh -c 'mkdir -p /workspace/hanlun-lms && mount /dev/vdf /workspace/hanlun-lms && ls /workspace/hanlun-lms | head -5'

AGENTS.md
CLAUDE.md
Makefile
README.md
authgear-admin
MOUNTED OK
```

The workspace IS captured correctly. Files are there. The problem is automount only.

---

### Step 5: In-guest git state — BLOCKED (no .git)

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-a \
    /bin/sh -c 'ls /workspace/hanlun-lms/.git 2>/dev/null || echo "NO .git DIRECTORY"'

NO .git DIRECTORY
```

The workspace capture includes the working tree files but NOT the `.git` directory.
This is intentional — the git object database would be large. But without `.git`, in-guest
`git status` and `git commit` fail immediately.

**G1 (per-sandbox git identity + branch convention at seed time) must solve this** by either:
a) Capturing `.git` into the workspace disk (expensive, object db is large), or
b) Running `git init` + setting up a tracking relationship to the real origin at seed time.

**Workaround for walkthrough (manual steps, NOT nexus3 primitives):**

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-a \
    /bin/sh -c '
cd /workspace/hanlun-lms
git init
git config user.name "nexus3-parallel-a"
git config user.email "nexus3@local"
git config safe.directory /workspace/hanlun-lms
echo "# nexus3 parallel-a walkthrough 2026-08-15" >> WALKTHROUGH_PARALLEL_A.md
git add WALKTHROUGH_PARALLEL_A.md
git commit -m "chore(parallel-a): walkthrough test 2026-08-15"
git log --oneline -3
'

Initialized empty Git repository in /workspace/hanlun-lms/.git/
[master (root-commit) f207a0a] chore(parallel-a): walkthrough test 2026-08-15
 1 file changed, 1 insertion(+)
 create mode 100644 WALKTHROUGH_PARALLEL_A.md
f207a0a chore(parallel-a): walkthrough test 2026-08-15
```

Note: `git init` creates a FRESH repository with no connection to `oursky/hanlun-lms`.
The resulting bundle cannot be directly applied to the real repo (no common history).
G1 must set the base ref so G2's bundle has proper ancestry.

---

### Step 6: Create git bundle in-guest and extract to host

**In-guest bundle creation (manual — G2 not built):**

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-a \
    /bin/sh -c '
cd /workspace/hanlun-lms
git bundle create /tmp/parallel-a.bundle HEAD master
git bundle verify /tmp/parallel-a.bundle
ls -lh /tmp/parallel-a.bundle
'

/tmp/parallel-a.bundle is okay
The bundle contains these 2 refs:
f207a0a6cb5415a51e6d7c2fef8358dc47151f6d HEAD
f207a0a6cb5415a51e6d7c2fef8358dc47151f6d refs/heads/master
The bundle records a complete history.
The bundle uses this hash algorithm: sha1
-rw-r--r-- 1 root root 415 Aug 15 05:02 /tmp/parallel-a.bundle
```

**Host-side extraction with nexus3 cp:**

```
$ mkdir -p /tmp/nexus3-walkthrough
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 cp hanlun-lms/parallel-a \
    guest:/tmp/parallel-a.bundle /tmp/nexus3-walkthrough/parallel-a.bundle

cp pull /tmp/parallel-a.bundle ↔ /tmp/nexus3-walkthrough/parallel-a.bundle
664  parallel-a.bundle  415B
```

`nexus3 cp` works correctly for single file extraction.

---

### Step 7: Stop sandbox 1 and create sandbox 2 sequentially

```
$ go run ./cmd/nexus3 sandbox stop hanlun-lms/parallel-a
stopped sandbox hanlun-lms/parallel-a (sb-06G07HKGMHTNSDD6J260KPAWBG)
```

(Shadow disk locks released by CloudHypervisor on VM stop.)

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 sandbox create hanlun-lms/parallel-b \
    --motive nexus3-parallel-dev-pr-flow \
    --workspace /home/newman/magic/hanlun-lms \
    --image nexus3-agent-base

2026/08/15 05:03:12 INFO workspace shadow disks prepared ...
2026/08/15 05:03:44 INFO workspace capture complete elapsed=27.412s shadow_dirs_excluded=true
created sandbox hanlun-lms/parallel-b (sb-06G07KBTE9RTS7CNQ2SB2SVA44)
```

Sequential create works. Two workspace disks now exist on host simultaneously:
- `sb-06G07HKGMHTNSDD6J260KPAWBG-workspace.ext4` — 13 GiB preallocated, ~6.7 GiB actual
- `sb-06G07KBTE9RTS7CNQ2SB2SVA44-workspace.ext4` — 13 GiB preallocated, ~6.7 GiB actual

---

### Step 8: Distinct change in sandbox 2 (parallel-b)

Same manual workaround sequence (mount + git init + change + commit):

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-b \
    /bin/sh -c '
mkdir -p /workspace/hanlun-lms && mount /dev/vdf /workspace/hanlun-lms
cd /workspace/hanlun-lms
git init
git config user.name "nexus3-parallel-b"
git config user.email "nexus3@local"
echo "# nexus3 parallel-b walkthrough 2026-08-15" >> WALKTHROUGH_PARALLEL_B.md
git add WALKTHROUGH_PARALLEL_B.md
git commit -m "chore(parallel-b): walkthrough test 2026-08-15"
git log --oneline -3
'

Initialized empty Git repository in /workspace/hanlun-lms/.git/
[master (root-commit) 601ba11] chore(parallel-b): walkthrough test 2026-08-15
601ba11 chore(parallel-b): walkthrough test 2026-08-15
```

```
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 exec hanlun-lms/parallel-b \
    /bin/sh -c 'cd /workspace/hanlun-lms && git bundle create /tmp/parallel-b.bundle HEAD master && git bundle verify /tmp/parallel-b.bundle'

/tmp/parallel-b.bundle is okay
The bundle contains these 2 refs:
601ba1152219b7ac879b39c83a5d1d81b0cb1ab3 HEAD
601ba1152219b7ac879b39c83a5d1d81b0cb1ab3 refs/heads/master

$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 cp hanlun-lms/parallel-b \
    guest:/tmp/parallel-b.bundle /tmp/nexus3-walkthrough/parallel-b.bundle

cp pull /tmp/parallel-b.bundle ↔ /tmp/nexus3-walkthrough/parallel-b.bundle
664  parallel-b.bundle  413B
```

---

### Step 9: Test nexus3 harvest (motive-level extraction)

```
$ mkdir -p /tmp/nexus3-walkthrough/harvest-bundle
$ NEXUS3_KERNEL_PATH=... go run ./cmd/nexus3 harvest nexus3-parallel-dev-pr-flow \
    /tmp/parallel-b.bundle /tmp/nexus3-walkthrough/harvest-bundle

harvest: 1/2 sandbox(es) succeeded, 1 failed; check outcomes for details
error: harvest motive "nexus3-parallel-dev-pr-flow": 1 of 2 sandbox(es) failed
exit status 1

$ find /tmp/nexus3-walkthrough/harvest-bundle -type f
/tmp/nexus3-walkthrough/harvest-bundle/sb-06G07HKGMHTNSDD6J260KPAWBG/parallel-b.bundle   # 0 bytes (parallel-a is stopped)
/tmp/nexus3-walkthrough/harvest-bundle/sb-06G07KBTE9RTS7CNQ2SB2SVA44/parallel-b.bundle   # 413 bytes (parallel-b is running)
```

`harvest` findings:
- Running sandbox (parallel-b): correctly copied the file, 413 bytes
- Stopped sandbox (parallel-a): created a 0-byte placeholder, no per-sandbox error message
- Exit code 1 (partial failure), but doesn't identify WHICH sandbox failed
- harvest of `/tmp` directory (not a file) creates a 0-byte file named `tmp` (incorrect behavior)

---

### Step 10: GitHub auth check and PR creation (host side, D-PD-01 respected)

No GitHub credential in any guest. All push/PR from host only.

```
$ cd /home/newman/magic/hanlun-lms
$ git stash -u -m "walkthrough stash pre-branch"  # saved dirty working tree
$ git checkout -b nexus3/parallel-a-walkthrough-test
$ echo "# nexus3 parallel-a walkthrough 2026-08-15" > WALKTHROUGH_PARALLEL_A.md
$ git add WALKTHROUGH_PARALLEL_A.md
$ git commit -m "chore(parallel-a): walkthrough test 2026-08-15"
$ git push origin nexus3/parallel-a-walkthrough-test

remote: Create a pull request for 'nexus3/parallel-a-walkthrough-test' on GitHub
To github.com:oursky/hanlun-lms.git
 * [new branch]  nexus3/parallel-a-walkthrough-test -> nexus3/parallel-a-walkthrough-test
ok nexus3/parallel-a-walkthrough-test
```

```
$ gh pr create --repo oursky/hanlun-lms \
    --head nexus3/parallel-a-walkthrough-test \
    --base main \
    --title "[nexus3-walkthrough] parallel-a: T0 parallel-dev flow test" \
    --body "..." \
    --draft

https://github.com/oursky/hanlun-lms/pull/832
```

PR #832 opened and immediately closed:

```
$ gh pr close 832 --delete-branch
✓ Closed pull request oursky/hanlun-lms#832
✓ Deleted branch nexus3/parallel-a-walkthrough-test

$ git checkout main && git stash pop
```

**Host push + PR: FULLY FUNCTIONAL.** D-PD-01 satisfied (no GitHub cred in guest).

---

### Step 11: Cleanup and resource audit

```
$ go run ./cmd/nexus3 sandbox stop hanlun-lms/parallel-b
stopped sandbox hanlun-lms/parallel-b (sb-06G07KBTE9RTS7CNQ2SB2SVA44)

$ go run ./cmd/nexus3 sandbox rm hanlun-lms/parallel-a
removed sandbox hanlun-lms/parallel-a

$ go run ./cmd/nexus3 sandbox rm hanlun-lms/parallel-b
removed sandbox hanlun-lms/parallel-b

$ go run ./cmd/nexus3 --json sandbox list
{"sandboxes":[
  {"id":"sb-06FZ2S93T1ZDF68NK723QMW4R0","handle":"~/magic/hanlun-lms/whatever","state":"created"},
  {"id":"sb-06FZ2SJVDHYPH6F25VC2A1S800","handle":"~/magic/nexus/nexus","state":"created"},
  {"id":"sb-06FZZX7V8XZM12YE7VTR7T8168","handle":"proof/uni5","state":"running"}
]}
```

Sandbox list: RETURNED TO BASELINE (3 pre-existing sandboxes only). ✓

**Disk audit after `sandbox rm`:**

```
$ ls -lah /home/newman/.local/state/nexus3/disks/

664  .next.shadow.ext4            10240.0M  (apparent; 69 MB actual — sparse)
664  dist.shadow.ext4             10240.0M  (apparent; 69 MB actual — sparse)
664  node_modules.shadow.ext4     10240.0M  (apparent; 69 MB actual — sparse)
600  sb-06FZZX7V8XZM12YE7VTR7T8168.raw   4096.0M  (uni5, pre-existing)
664  sb-06G07HKGMHTNSDD6J260KPAWBG-workspace.ext4  13165.0M  (parallel-a ORPHANED)
664  sb-06G07KBTE9RTS7CNQ2SB2SVA44-workspace.ext4  13165.0M  (parallel-b ORPHANED)
664  target.shadow.ext4           10240.0M  (apparent; 69 MB actual — sparse)
```

**`sandbox rm` does NOT reclaim disk files.** Two workspace disks (~13.4 GiB actual)
and four shadow disks (~276 MB actual, ~40 GiB apparent) were orphaned.

`nexus3 recover` examined only the 3 live sandbox records and had no visibility into
orphaned disks. No nexus3 command can reclaim these without a reaper (R1 slice).

Post-walkthrough: manually deleted the sandbox-ID-named workspace disks.

```
$ rm /home/newman/.local/state/nexus3/disks/sb-06G07HKGMHTNSDD6J260KPAWBG-workspace.ext4
$ rm /home/newman/.local/state/nexus3/disks/sb-06G07KBTE9RTS7CNQ2SB2SVA44-workspace.ext4

$ df -h /home
466G  399G   43G  91%   # near-baseline (was 44G; shadow disks account for ~1G actual)
```

**Disk: ~43 GiB free post-cleanup (baseline was 44 GiB).** Residual ~1 GiB from
4 shadow disks (each ~69 MB actual but sparse-sparse files consume some real blocks).
**Shadow disk files persist after `sandbox rm` with no reclaim path.**

---

## Ordered list of manual steps → slice ownership

Steps numbered in execution order. Each step that Waves 2–5 must replace with a verb.

| # | Manual step today | Owning slice |
|---|-------------------|--------------|
| 1 | Set `NEXUS3_KERNEL_PATH=/path/to/vmlinux-x86_64` before every `sandbox create` invocation (discovered by trial-and-error: first attempt wastes 28s capture) | **CHARTER GAP** — no slice owns this; M3 preflight should absorb a kernel-path pre-check |
| 2 | Stop the first sandbox before creating the second (parallel boot is impossible due to global shadow disk locking at `cli/shadowdisk.go:111`) | **M3** — `nexus3 up <motive> --count N` must namespace shadow disks per-sandbox |
| 3 | Rebuild `nexus3-agent-base` with the Aug-14 agent binary before any workspace-mount sandbox can auto-mount | **CHARTER GAP** — no slice owns image currency; new base images must be built after agent changes |
| 4 | After each sandbox starts: `exec <sb> /bin/sh -c 'mkdir -p /workspace/hanlun-lms && mount /dev/vdf /workspace/hanlun-lms'` | **G1** — seed-time setup must mount workspace disks (fix: rebuild image with current agent) |
| 5 | Per-sandbox: `exec <sb> git init` | **G1** — git repo initialization at seed time |
| 6 | Per-sandbox: `exec <sb> git config user.name "nexus3-<motive>-<id>"` | **G1** — deterministic bot identity |
| 7 | Per-sandbox: `exec <sb> git config user.email "nexus3@local"` | **G1** — bot identity |
| 8 | Per-sandbox: `exec <sb> git config safe.directory /workspace/...` | **G1** — safe.directory setup |
| 9 | Per-sandbox: `exec <sb> git branch -m nexus3/<motive>/<sandbox-short-id>` | **G1** — branch convention |
| 10 | Per-sandbox: establish tracking relationship to base ref (so bundle has proper ancestry for host repo application) | **G1** — recorded base ref |
| 11 | Per-sandbox: `exec <sb> git bundle create /tmp/<id>.bundle HEAD <branch>` | **G2** — `nexus3 bundle <motive\|sandbox> --out <dir>` |
| 12 | Per-sandbox: `nexus3 cp <sb> guest:/tmp/<id>.bundle <host-dir>/<id>.bundle` | **G2** — absorbed into `nexus3 bundle` |
| 13 | Per-sandbox: on host `git checkout -b nexus3/<motive>/<id>` from the right base | **P1** — `nexus3 pr <motive>` |
| 14 | Per-sandbox: apply the bundle diff to the host branch (bundle → cherry-pick or merge) | **P1** + **G1** (requires recorded base ref) |
| 15 | Per-sandbox: `git push origin nexus3/<motive>/<id>` | **P1** |
| 16 | Per-sandbox: `gh pr create --repo ... --head ... --base main --title ... --draft` | **P1** |
| 17 | After `sandbox rm`: manually `rm <id>-workspace.ext4` for orphaned workspace disks | **R1** (reaper) — `sandbox rm` must also reclaim disk files |

---

## Charter gaps (manual steps with no owning slice)

### GAP-1: NEXUS3_KERNEL_PATH discoverability

**Step 1 above.** Every invocation of `nexus3 sandbox create` via `go run` or any binary
not co-located with `images/kernel/` will silently proceed through a full workspace capture
(28 seconds, ~6.7 GiB) and then fail at boot time with a CloudHypervisor error that gives
no actionable hint about the kernel path.

The current help string (`cmd_sandbox.go:302`) does not mention `NEXUS3_KERNEL_PATH`.
The `kernelPathFor()` function (`cmd_sandbox.go:1189`) returns an empty string if resolution
fails, and the downstream error is opaque.

**Expected fix:** M3 preflight should validate the kernel path before starting any workspace
capture. The error message should name both the env var and the binary-relative fallback path.
No existing slice owns this. Recommend adding it to M3's acceptance criteria.

### GAP-2: Image currency — stale `nexus3-agent-base`

**Step 3 above.** There is no slice that specifies when base images are rebuilt after product
changes. The 2026-08-11 `nexus3-agent-base` images became stale when workspace mount
support landed 2026-08-13. No operator is alerted; no CI rebuilds the image. The result:
every sandbox using `--workspace --image nexus3-agent-base` silently captures the workspace
and attaches the disks but doesn't mount them — a subtle failure mode requiring `cat /proc/cmdline` to diagnose.

**Expected fix:** Either a CI step that rebuilds `nexus3-agent-base` when `cmd/nexus3-agent`
changes, or the image build (`nexus3 image build`) must be a required step documented in the
operator runbook. No existing slice owns this.

### GAP-3: `sandbox rm` does not reclaim workspace or shadow disks

**Step 17 above.** After `sandbox rm`, the workspace disk (e.g., `sb-<id>-workspace.ext4`,
~6.7 GiB actual) and shadow disks (`.next.shadow.ext4`, `dist.shadow.ext4`, etc., ~69 MB
each actual) are orphaned. `nexus3 recover` has no visibility into files that have no
corresponding live sandbox record.

The reaper slice (R1) is supposed to handle reclamation via an index, but R1 is not yet
built. More urgently: `sandbox rm` itself should delete the workspace disk it created — this
is basic resource ownership. The current gap means each create+rm cycle leaks ~6.7 GiB.

**Expected fix:** `sandbox rm` deletes associated workspace disk files as part of the same
operation. R1 handles the orphan case for disks that survive abnormal termination.

> **UPDATE 2026-08-18:** Workspace disks are now ULID-keyed (`<ULID>-workspace.ext4`) and are reclaimed by `nexus3 reap --apply` once the sandbox record is removed. The globally-named shadow disk files (`.next.shadow.ext4`, etc.) from this walkthrough are superseded: the new parallel-dev approach uses named volumes (`--mount-named`) which are intentionally user-owned and must be reclaimed explicitly with `nexus3 volume rm` or `nexus3 volume prune`. The legacy shadow disk files observed in Step 11 should be cleaned up with `nexus3 reap --apply` or `rm` manually.

### GAP-4: Harvest empty-placeholder behavior for stopped sandboxes

`nexus3 harvest <motive> <guest-path> <host-dir>` with a stopped sandbox creates a 0-byte
file (not a directory or a meaningful error) at the output path. The exit code is 1 and the
error message doesn't identify which sandbox failed. This makes harvest output unreliable
as input to G2 or P1.

No existing slice owns harvest's partial-failure semantics. M2 (`nexus3 exec --motive`)
should have an analogous discussion but is a different command. Recommend adding per-sandbox
outcome reporting to harvest's acceptance criteria.

---

## Resource envelope measurements

| Metric | Measured value |
|--------|----------------|
| Workspace capture time (hanlun-lms, ~6.7 GiB actual) | 27–28 seconds |
| Workspace disk: apparent size (sparse) | 13165 MiB = 12.9 GiB |
| Workspace disk: actual disk usage | ~6.7 GiB |
| Shadow disk per heavy dir: apparent | 10240 MiB = 10 GiB |
| Shadow disk per heavy dir: actual (after boot) | ~69 MB (sparse, mostly empty) |
| RAM per running sandbox (from memory notes) | ~85 MiB RSS |
| Shadow dirs excluded from workspace (4 total) | node_modules, .next, target, dist |
| Disk headroom at start | 44 GiB free |
| Sequential 2-sandbox run: peak disk additional | ~13.4 GiB (workspace disks only; shadow disks sparse) |
| Sandboxes possible before disk exhaustion (sequential) | ~6 (at 6.7 GiB actual per workspace) |
| Sandboxes possible concurrently today | **1** (shadow disk locking) |

---

## Top 5 friction points

### F1: Parallel boot is impossible — shadow disks are globally named
**File:** `internal/cli/shadowdisk.go:111`  
**Command:** `go run ./cmd/nexus3 sandbox create hanlun-lms/parallel-b ...` (while parallel-a running)  
**Error:** `Failed to get Write lock for disk image: /home/newman/.local/state/nexus3/disks/node_modules.shadow.ext4 — The file is already locked`  
Two sandboxes from the same workspace share shadow disk files with no sandbox ID in the path. CloudHypervisor holds exclusive write locks. The parallel-dev flow's primary use case is impossible without fixing this.  
**Blocks:** M3 (bulk create), the entire parallel-dev concept.

> **UPDATE 2026-08-18 (D-PD-82):** Named volumes (`--mount-named kind=disk`) replace shadow disks for the parallel-dev use case. Each sandbox gets its own named volume (e.g. `myapp-node_modules`) identified by project slug — no global lock contention. Two or more sandboxes from the same project can now run concurrently. The shadow disk mechanism still exists for backward compatibility but is no longer the recommended path.

### F2: `nexus3-agent-base` is stale — workspace not auto-mounted
**File:** `internal/cli/cmd_sandbox.go:1197`, agent image built 2026-08-11  
**Symptom:** Agent ignores `--workspace-mount` kernel cmdline args; disks attached as /dev/vdb–vdf but not mounted; `/workspace/hanlun-lms` doesn't exist in guest.  
**Diagnosis requires:** `cat /proc/cmdline` (to see the ignored args) + comparing agent binary sizes (15 MB image vs 35 MB host).  
**Blocks:** G1, G4, every workspace-based workflow.

### F3: NEXUS3_KERNEL_PATH not discoverable — first attempt always fails
**File:** `internal/cli/cmd_sandbox.go:1189` (`kernelPathFor()`)  
**Error (first attempt):** `cloudhypervisor: vm.boot: ... Cannot open kernel file ... No such file or directory`  
The error burns 28 seconds of workspace capture before failing. There's no mention of `NEXUS3_KERNEL_PATH` in any error message or help text. A full workspace capture is done before the kernel path is validated.  
**Blocks:** First-time operator experience, M3 preflight.

### F4: No `.git` in captured workspace — in-guest commits require manual git init
**Mechanism:** Workspace capture excludes `.git` by design  
**Symptom:** `git status` → `fatal: not a git repository`; no shared history with origin  
**Consequence:** In-guest bundles (when G2 is built) will be orphaned roots with no common ancestor with `oursky/hanlun-lms`. P1 cannot apply them without G1 establishing the base ref.  
**Blocks:** G1 (whole slice premise), G2, P1.

### F5: `sandbox rm` leaks workspace disks (~6.7 GiB each)
**Files leaked:** `/home/newman/.local/state/nexus3/disks/sb-<id>-workspace.ext4`  
**Symptom:** After `sandbox rm`, 6.7 GiB per sandbox is stranded on disk with no nexus3 command able to reclaim it. `nexus3 recover` only sees live records, not orphaned files.  
**Measured:** 2 sandboxes created and removed → 13.4 GiB stranded before manual delete.  
**Blocks:** R1 (reaper), operator trust. The motive objective quotes "11.2 GiB currently stranded on this host" — this walkthrough added to it.

> **UPDATE 2026-08-18 (D-PD-73, D-PD-74, D-PD-80(b)):** Workspace disk naming is now ULID-keyed (`<childULID>-workspace.ext4`), making orphaned workspace disks fully visible to and reclaimed by `nexus3 reap`. The create-window hazard is closed by flock leases (D-PD-73 for sandbox create, D-PD-74 for fork children). Named volume backing files remain user-owned and are never auto-deleted; use `nexus3 volume rm` or `nexus3 volume prune` to reclaim them.

---

## Appendix: commands attempted and their verdict

| Command | Verdict |
|---------|---------|
| `nexus3 sandbox create ... --motive ... --workspace ... --image nexus3-agent-base` | WORKS (with `NEXUS3_KERNEL_PATH` set) |
| Creating 2nd sandbox while 1st is running | BLOCKED (shadow disk lock) |
| `nexus3 exec <sb> /bin/sh -c '...'` | WORKS |
| Auto-mount of workspace disk in guest | BLOCKED (stale image agent) |
| `nexus3 cp <sb> guest:<path> <host-path>` | WORKS |
| `nexus3 harvest <motive> <file-path> <host-dir>` | PARTIAL (running sandboxes only; stopped → empty placeholder) |
| `nexus3 harvest <motive> <dir-path> <host-dir>` | BROKEN (creates empty file, not directory contents) |
| `nexus3 sandbox stop <sb>` | WORKS |
| `nexus3 sandbox rm <sb>` | WORKS (records only; disks NOT reclaimed) |
| `nexus3 recover` | WORKS (live records only; no orphaned disk visibility) |
| `git push origin <branch>` from host | WORKS (SSH auth, IniZio account) |
| `gh pr create --draft` | WORKS (PR #832 opened and closed) |
| `nexus3 motive status` | NOT BUILT (M1 slice, Wave 2) |
| `nexus3 exec --motive` | NOT BUILT (M2 slice, Wave 2) |
| `nexus3 up <motive> --count N` | NOT BUILT (M3 slice, Wave 2) |
| `nexus3 bundle <motive> --out <dir>` | NOT BUILT (G2 slice, Wave 3) |
| `nexus3 pr <motive>` | NOT BUILT (P1 slice, Wave 4) |

---

## S1-AC2 gate attempt — 2026-08-19 (live, KVM)

The gate was previously unattemptable: the image store held zero images. It was
unblocked by building the base image with `images/kernel/rebuild-base.sh --image`
(docker multi-stage build → 6 GiB ext4 → registered in the production cache).

**Condition (a) — named-volume mounts in ≥2 concurrent sandboxes: MET.**
Two volumes (`qa-vol1`, `qa-vol2`) were created and attached to two sandboxes
launched concurrently. `sandbox list --json` showed both in `state: running` at
the same instant, and each guest exposed its volume as `/dev/vdb`.

A same-volume variant was also exercised: two sandboxes attaching **one**
disk-kind volume. The second was refused —

```
rw conflict: sandbox sb-06G1FF2XVSR09CZ5KRP66GZ35M create is in flight (intent lease held)
```

This is the D-PD-93 attach guard behaving correctly: disk-kind volumes are
single-writer. "Two concurrent named-volume sandboxes" therefore means one
distinct volume per sandbox, not one shared volume. The refusal message is
however **misleading** — it attributes the conflict to an in-flight create and a
held intent lease, when the real reason is that the volume is already attached
read-write. See TBD-PD-21.

**Condition (b) — in-guest agent reaches the Anthropic API: NOT ATTEMPTED.**
Operator-gated. Seeding credentials requires an interactive
`nexus3 auth login --force` run from a dedicated Claude session (not the main
login, which rotates and logs the operator out). No agent obtained, fabricated,
or used credentials. This condition remains open and is the sole blocker on
declaring S1-AC2 met.

**Condition (c) — zero new orphans: MET.**
Verified independently of the executing agent, by mtime rather than by ID
prefix. After teardown (`sandbox list` → `0 sandbox(es)`, `volume ls` → `no
volumes`), `nexus3 reap` reported ~54 orphans at time of run, **all of kind `.raw`**. No file in
`~/.local/state/nexus3/disks/` or `/run/user/1003/nexus3/` has an mtime inside
the live-run window (the newest orphan predates it by ~12 minutes). Zero disk
and zero socket resources were stranded by this flow.

`ResourceIndex` enumerates `.iid` files as `KindSocketIID` (`resource_index.go:174-176`) and `clearState()` (`driver.go:484`) removes the `.iid` file on driver stop. Yet `/run/user/1003/nexus3/` still holds `.iid`/`.vsock`/`.sock` triples dated 08-14 through 08-16 for long-dead sandboxes with no store record, and `reap` reports zero socket-kind orphans. Why the reaper classifies no socket resources despite `ResourceIndex` enumerating them is unresolved and tracked as an open question for R1.

**Defect found — `sandbox rm` leaves a stale volume attachment.**
After both sandboxes were removed, `volume rm qa-vol1` failed with
`volume in use: attached to sb-06G1FF2XVSR09CZ5KRP66GZ35M` — a sandbox that no
longer exists. Removal required `volume prune --apply --include-detached`.
`prune`'s dry-run correctly classified the volumes as detached, so the stale
attachment record blocks `volume rm` specifically. See TBD-PD-22.

**Gate verdict: NOT MET.** Two of three conditions are met with recorded
evidence; (b) is operator-gated. TBR-PD-15 and TBR-PD-18 remain blocked.
