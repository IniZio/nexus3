# Changelog

## Unreleased

### Breaking changes

**`--no-builtin-gh` flag removed** (D-PDE-02)

The flag was a no-op since the built-in GitHub token auto-injection was removed. It is now rejected as an unknown flag.

Migration: `nexus3 sandbox create --repo X` no longer auto-attaches a GitHub credential. To wire one, add `--secret GH_TOKEN@github.com,api.github.com,uploads.github.com` explicitly — one mechanism for all credential injection.

```sh
# Before (--no-builtin-gh was a deprecated no-op):
nexus3 create p/n --repo owner/repo

# After:
nexus3 create p/n --repo owner/repo --secret GH_TOKEN@github.com,api.github.com,uploads.github.com
```

Sandboxes created without `--repo` receive no GitHub credential (fail-closed default unchanged).
