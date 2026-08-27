// t6_path_policy_guard_test.go — T6 guard re-expression tests (D-PD-36 / D-PDE-16).
//
// Guards the asymmetry: a GitHub-host secret bind without a path policy (AllowedRepo
// shim or PathPolicies entry) is REFUSED; a non-GitHub-host bind without a path policy
// is ALLOWED. Each refusal case is mutation-proven: removing the guard causes the test
// to fail.
//
// Four cases (spec):
//   (a) github host, no policy       → REFUSED at create, Start, startSupervisor
//   (b) github host, AllowedRepo set → ALLOWED (shim covers it)
//   (c) github host, PathPolicies entry for that host → ALLOWED
//   (d) non-github host, no policy   → ALLOWED (host-only, D-PDE-16 asymmetry)
//
// Cases (a) and (b) are already exercised by pre-existing tests; they are
// re-stated here in the T6 context to make the three-site coverage explicit.
// Cases (c) and (d) are new (PathPolicies path not previously tested here).

package service

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/IniZio/nexus3/internal/core/domain"
	"github.com/IniZio/nexus3/internal/core/driver"
	"github.com/IniZio/nexus3/internal/core/driver/fake"
	"github.com/IniZio/nexus3/internal/core/image"
	"github.com/IniZio/nexus3/internal/core/lifecycle"
	"github.com/IniZio/nexus3/internal/core/store"
)

// ── internal NetworkHook stub ──────────────────────────────────────────────────

// t6NetHook wraps FakeDriver and satisfies driver.NetworkHook by returning one
// end of a net.Pipe as the guest fd. Used to drive startSupervisor directly.
type t6NetHook struct {
	*fake.FakeDriver
	conn net.Conn
}

func (h *t6NetHook) GuestNetworkFD(_ context.Context, _ domain.SandboxID) (io.ReadWriteCloser, error) {
	c := h.conn
	h.conn = nil // single-use
	return c, nil
}

// newT6NetHook returns a t6NetHook backed by net.Pipe, plus the host end of
// the pair. Closing the host end unblocks any read on the guest end.
func newT6NetHook() (*t6NetHook, net.Conn) {
	guestConn, hostConn := net.Pipe()
	return &t6NetHook{FakeDriver: fake.New(), conn: guestConn}, hostConn
}

// ── store seed helper ──────────────────────────────────────────────────────────

// t6SeedStore creates a FileStore pre-populated with sb, bypassing CreateAndBoot
// so the record does not pass the create-time guard. Simulates a record written
// before the policy guard existed.
func t6SeedStore(t *testing.T, sb domain.Sandbox) store.Store {
	t.Helper()
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := st.Create(context.Background(), sb); err != nil {
		t.Fatalf("store.Create: %v", err)
	}
	return st
}

// t6ImageOpts returns a CreateAndBootOptions with a valid image spec so that
// image resolution does not fail before the guard fires. Cache is created in a
// temp dir; a minimal fake ext4 blob is stored so the digest resolves.
func t6ImageOpts(t *testing.T) (opts CreateAndBootOptions, cacheRoot string) {
	t.Helper()
	cacheRoot = t.TempDir()
	cache, err := image.NewCache(cacheRoot)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	img := putFakeImage(t, context.Background(), cache)
	opts = CreateAndBootOptions{
		Image:     ImageSpec{Digest: string(img.Digest)},
		CacheRoot: cacheRoot,
	}
	return opts, cacheRoot
}

// ── (a) github host, no policy → REFUSED ─────────────────────────────────────

// TestT6_CreateGuard_GitHubNoPolicy_Refused verifies case (a) at the CreateAndBoot
// site. Mutation evidence: remove the githubHostBoundByPolicy guard loop in
// CreateAndBoot → test fails (image resolves but guard is gone; sandbox is
// created instead of returning ErrUnboundGitHubSecret).
func TestT6_CreateGuard_GitHubNoPolicy_Refused(t *testing.T) {
	base, cacheRoot := t6ImageOpts(t)
	cache, _ := image.NewCache(cacheRoot)
	opts := base
	opts.Secrets = []SecretBind{{
		Env:   "GH_TOKEN",
		Hosts: []string{"github.com"},
		Token: "ghp_test",
	}}
	// AllowedRepo: "",     // omitted — triggers guard
	// PathPolicies: nil,   // omitted — triggers guard

	_, err := CreateAndBoot(
		context.Background(),
		newTestSvc(t, fake.New()),
		cache,
		func(_ string, _ []ExtraDisk) (driver.Driver, error) { return fake.New(), nil },
		noopProbe,
		"proj", "t6-gh-no-policy",
		opts,
	)
	if !errors.Is(err, ErrUnboundGitHubSecret) {
		t.Fatalf("CreateAndBoot: got %v, want ErrUnboundGitHubSecret", err)
	}
}

// TestT6_StartGuard_GitHubNoPolicy_Refused verifies case (a) at the Start site.
// Mutation evidence: remove the githubHostBoundByPolicy condition in Start →
// test fails (Start does not return ErrUnboundGitHubSecret).
func TestT6_StartGuard_GitHubNoPolicy_Refused(t *testing.T) {
	sb := domain.Sandbox{
		ID:      domain.NewSandboxID(),
		Name:    "t6-gh-no-policy",
		Project: "test",
		State:   domain.Stopped,
		Envelope: domain.Envelope{
			ImageDigest: "sha256:deadbeef",
			SecretHosts: []string{"github.com"},
			// AllowedRepo: "",   // omitted
			// PathPolicies: nil, // omitted
		},
	}
	svc := New(t6SeedStore(t, sb), fake.New(), lifecycle.New())
	_, err := svc.Start(context.Background(), sb.ID.String())
	if !errors.Is(err, ErrUnboundGitHubSecret) {
		t.Fatalf("Start: got %v, want ErrUnboundGitHubSecret", err)
	}
}

// TestT6_SupervisorGuard_GitHubNoPolicy_Refused verifies case (a) at the
// startSupervisor backstop site. The guard fires even when called directly,
// bypassing the Start guard — covering fork/restore children.
//
// Mutation evidence: remove the githubHostBoundByPolicy condition in
// startSupervisor → test fails (startSupervisor does not return
// ErrUnboundGitHubSecret).
func TestT6_SupervisorGuard_GitHubNoPolicy_Refused(t *testing.T) {
	hook, hostConn := newT6NetHook()
	defer hostConn.Close()

	svc := newTestSvc(t, hook)
	sb := domain.Sandbox{
		ID:      domain.NewSandboxID(),
		Project: "test",
		Envelope: domain.Envelope{
			SecretHosts: []string{"github.com"},
			// AllowedRepo: "",   // omitted
			// PathPolicies: nil, // omitted
		},
	}
	err := svc.startSupervisor(context.Background(), hook, sb)
	if !errors.Is(err, ErrUnboundGitHubSecret) {
		t.Fatalf("startSupervisor: got %v, want ErrUnboundGitHubSecret", err)
	}
}

// ── (b) github host, AllowedRepo set → ALLOWED ───────────────────────────────

// TestT6_StartGuard_GitHubWithAllowedRepo_Allowed verifies case (b) at Start:
// AllowedRepo satisfies the path-policy requirement, guard must not fire.
//
// Mutation evidence: remove the `allowedRepo != ""` early-return in
// githubHostBoundByPolicy → Start returns ErrUnboundGitHubSecret instead.
func TestT6_StartGuard_GitHubWithAllowedRepo_Allowed(t *testing.T) {
	sb := domain.Sandbox{
		ID:      domain.NewSandboxID(),
		Name:    "t6-gh-with-repo",
		Project: "test",
		State:   domain.Stopped,
		Envelope: domain.Envelope{
			ImageDigest: "sha256:deadbeef",
			SecretHosts: []string{"github.com"},
			AllowedRepo: "owner/repo", // satisfies the guard
		},
	}
	svc := New(t6SeedStore(t, sb), fake.New(), lifecycle.New())
	_, err := svc.Start(context.Background(), sb.ID.String())
	if err != nil {
		t.Fatalf("Start with AllowedRepo: got %v, want nil", err)
	}
}

// ── (c) github host, PathPolicies entry → ALLOWED ────────────────────────────

// TestT6_CreateGuard_GitHubWithPathPolicy_Allowed verifies case (c) at
// CreateAndBoot: a PathPolicies entry for the github host satisfies the guard
// without AllowedRepo being set.
//
// Mutation evidence: remove the PathPolicies loop in githubHostBoundByPolicy →
// CreateAndBoot returns ErrUnboundGitHubSecret even though PathPolicies is set.
func TestT6_CreateGuard_GitHubWithPathPolicy_Allowed(t *testing.T) {
	base, cacheRoot := t6ImageOpts(t)
	cache, _ := image.NewCache(cacheRoot)
	pp := domain.EgressPathPolicies{
		"": {"github.com": domain.EgressHostPolicy{Paths: []string{"/repos/owner/repo/**"}}},
	}
	opts := base
	opts.Secrets = []SecretBind{{
		Env:   "GH_TOKEN",
		Hosts: []string{"github.com"},
		Token: "ghp_test",
	}}
	opts.PathPolicies = pp
	// AllowedRepo: "",   // not set — PathPolicies must carry the guard

	_, err := CreateAndBoot(
		context.Background(),
		newTestSvc(t, fake.New()),
		cache,
		func(_ string, _ []ExtraDisk) (driver.Driver, error) { return fake.New(), nil },
		noopProbe,
		"proj", "t6-gh-paths-policy",
		opts,
	)
	if err != nil {
		t.Fatalf("CreateAndBoot with PathPolicies: got %v, want nil", err)
	}
}

// TestT6_StartGuard_GitHubWithPathPolicy_Allowed verifies case (c) at Start.
//
// Mutation evidence: same as TestT6_CreateGuard_GitHubWithPathPolicy_Allowed.
func TestT6_StartGuard_GitHubWithPathPolicy_Allowed(t *testing.T) {
	pp := domain.EgressPathPolicies{
		"": {"github.com": domain.EgressHostPolicy{Paths: []string{"/repos/owner/repo/**"}}},
	}
	sb := domain.Sandbox{
		ID:      domain.NewSandboxID(),
		Name:    "t6-gh-paths-start",
		Project: "test",
		State:   domain.Stopped,
		Envelope: domain.Envelope{
			ImageDigest:  "sha256:deadbeef",
			SecretHosts:  []string{"github.com"},
			PathPolicies: pp, // satisfies the guard; AllowedRepo deliberately empty
		},
	}
	svc := New(t6SeedStore(t, sb), fake.New(), lifecycle.New())
	_, err := svc.Start(context.Background(), sb.ID.String())
	if err != nil {
		t.Fatalf("Start with PathPolicies: got %v, want nil", err)
	}
}

// TestT6_SupervisorGuard_GitHubWithPathPolicy_Allowed verifies case (c) at
// startSupervisor.
//
// Mutation evidence: same as TestT6_CreateGuard_GitHubWithPathPolicy_Allowed.
func TestT6_SupervisorGuard_GitHubWithPathPolicy_Allowed(t *testing.T) {
	pp := domain.EgressPathPolicies{
		"": {"github.com": domain.EgressHostPolicy{Paths: []string{"/repos/owner/repo/**"}}},
	}
	hook, hostConn := newT6NetHook()
	defer hostConn.Close()

	svc := newTestSvc(t, hook)
	sb := domain.Sandbox{
		ID:      domain.NewSandboxID(),
		Project: "test",
		Envelope: domain.Envelope{
			SecretHosts:  []string{"github.com"},
			PathPolicies: pp, // satisfies the guard; AllowedRepo deliberately empty
		},
	}
	err := svc.startSupervisor(context.Background(), hook, sb)
	// Guard must not return ErrUnboundGitHubSecret. Other errors (e.g. from
	// netstack/MITM setup without a real broker) are acceptable — only the
	// path-policy guard is under test.
	if errors.Is(err, ErrUnboundGitHubSecret) {
		t.Fatalf("startSupervisor with PathPolicies: got ErrUnboundGitHubSecret, want guard to pass")
	}
}

// ── (d) non-github host, no policy → ALLOWED (asymmetry) ─────────────────────

// TestT6_CreateGuard_NonGitHubNoPolicy_Allowed verifies case (d) at
// CreateAndBoot: a non-GitHub secret bind without any path policy must be
// permitted — only GitHub hosts trigger the path-policy requirement (D-PDE-16).
//
// Mutation evidence: change the `isGitHubHost(h)` condition to always-true in
// the create guard → CreateAndBoot returns ErrUnboundGitHubSecret for a
// non-GitHub bind that has no path policy.
func TestT6_CreateGuard_NonGitHubNoPolicy_Allowed(t *testing.T) {
	base, cacheRoot := t6ImageOpts(t)
	cache, _ := image.NewCache(cacheRoot)
	opts := base
	opts.Secrets = []SecretBind{{
		Env:   "REGISTRY_TOKEN",
		Hosts: []string{"registry.example.com"},
		Token: "tok_test",
	}}
	// No AllowedRepo, no PathPolicies — permitted for non-GitHub

	_, err := CreateAndBoot(
		context.Background(),
		newTestSvc(t, fake.New()),
		cache,
		func(_ string, _ []ExtraDisk) (driver.Driver, error) { return fake.New(), nil },
		noopProbe,
		"proj", "t6-nongithub-no-policy",
		opts,
	)
	if errors.Is(err, ErrUnboundGitHubSecret) {
		t.Fatalf("CreateAndBoot non-github host: got ErrUnboundGitHubSecret, want no guard error (asymmetry)")
	}
}

// TestT6_StartGuard_NonGitHubNoPolicy_Allowed verifies case (d) at Start.
//
// Mutation evidence: same as TestT6_CreateGuard_NonGitHubNoPolicy_Allowed.
func TestT6_StartGuard_NonGitHubNoPolicy_Allowed(t *testing.T) {
	sb := domain.Sandbox{
		ID:      domain.NewSandboxID(),
		Name:    "t6-nongithub-start",
		Project: "test",
		State:   domain.Stopped,
		Envelope: domain.Envelope{
			ImageDigest: "sha256:deadbeef",
			SecretHosts: []string{"registry.example.com"},
			// No AllowedRepo, no PathPolicies — permitted for non-GitHub
		},
	}
	svc := New(t6SeedStore(t, sb), fake.New(), lifecycle.New())
	_, err := svc.Start(context.Background(), sb.ID.String())
	if errors.Is(err, ErrUnboundGitHubSecret) {
		t.Fatalf("Start non-github host: got ErrUnboundGitHubSecret, want no guard error (asymmetry)")
	}
}

// TestT6_SupervisorGuard_NonGitHubNoPolicy_Allowed verifies case (d) at
// startSupervisor.
//
// Mutation evidence: same as TestT6_CreateGuard_NonGitHubNoPolicy_Allowed.
func TestT6_SupervisorGuard_NonGitHubNoPolicy_Allowed(t *testing.T) {
	hook, hostConn := newT6NetHook()
	defer hostConn.Close()

	svc := newTestSvc(t, hook)
	sb := domain.Sandbox{
		ID:      domain.NewSandboxID(),
		Project: "test",
		Envelope: domain.Envelope{
			SecretHosts: []string{"registry.example.com"},
			// No AllowedRepo, no PathPolicies — permitted for non-GitHub
		},
	}
	err := svc.startSupervisor(context.Background(), hook, sb)
	if errors.Is(err, ErrUnboundGitHubSecret) {
		t.Fatalf("startSupervisor non-github host: got ErrUnboundGitHubSecret, want no guard error (asymmetry)")
	}
}
