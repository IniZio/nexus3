package cli

import "testing"

func TestDeriveGitHubRepo(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantOwner string
		wantName  string
		wantOK    bool
	}{
		// HTTPS forms
		{
			name:      "https without .git",
			remoteURL: "https://github.com/octocat/Hello-World",
			wantOwner: "octocat",
			wantName:  "Hello-World",
			wantOK:    true,
		},
		{
			name:      "https with .git",
			remoteURL: "https://github.com/octocat/Hello-World.git",
			wantOwner: "octocat",
			wantName:  "Hello-World",
			wantOK:    true,
		},
		{
			name:      "https with trailing slash stripped",
			remoteURL: "https://github.com/owner/myrepo.git",
			wantOwner: "owner",
			wantName:  "myrepo",
			wantOK:    true,
		},

		// SSH SCP-like form
		{
			name:      "ssh scp with .git",
			remoteURL: "git@github.com:octocat/Hello-World.git",
			wantOwner: "octocat",
			wantName:  "Hello-World",
			wantOK:    true,
		},
		{
			name:      "ssh scp without .git",
			remoteURL: "git@github.com:octocat/Hello-World",
			wantOwner: "octocat",
			wantName:  "Hello-World",
			wantOK:    true,
		},

		// SSH URL form
		{
			name:      "ssh:// url with .git",
			remoteURL: "ssh://git@github.com/octocat/Hello-World.git",
			wantOwner: "octocat",
			wantName:  "Hello-World",
			wantOK:    true,
		},
		{
			name:      "ssh:// url without .git",
			remoteURL: "ssh://git@github.com/octocat/Hello-World",
			wantOwner: "octocat",
			wantName:  "Hello-World",
			wantOK:    true,
		},

		// Non-github hosts → ok=false
		{
			name:      "gitlab https",
			remoteURL: "https://gitlab.com/owner/repo.git",
			wantOK:    false,
		},
		{
			name:      "bitbucket https",
			remoteURL: "https://bitbucket.org/owner/repo.git",
			wantOK:    false,
		},
		{
			name:      "self-hosted ssh scp",
			remoteURL: "git@git.example.com:owner/repo.git",
			wantOK:    false,
		},
		{
			name:      "self-hosted ssh url",
			remoteURL: "ssh://git@git.corp.internal/owner/repo.git",
			wantOK:    false,
		},

		// Path injection attempts (AC T3-AC3)
		{
			name:      "https percent-encoded traversal in owner",
			remoteURL: "https://github.com/o/..%2f..%2fx",
			wantOK:    false,
		},
		{
			name:      "https double-dot traversal extra segments",
			remoteURL: "https://github.com/o/n/../../evil",
			wantOK:    false,
		},
		{
			name:      "ssh scp extra path segments",
			remoteURL: "git@github.com:o/n/../../evil",
			wantOK:    false,
		},
		{
			name:      "https owner with slash injection",
			remoteURL: "https://github.com/evil%2fowner/repo",
			wantOK:    false,
		},
		{
			name:      "https semicolon in name",
			remoteURL: "https://github.com/owner/re;po",
			wantOK:    false,
		},
		{
			name:      "https non-ascii in segment",
			remoteURL: "https://github.com/own\xc3\xa9r/repo",
			wantOK:    false,
		},

		// Empty / garbage
		{
			name:      "empty string",
			remoteURL: "",
			wantOK:    false,
		},
		{
			name:      "garbage string",
			remoteURL: "not-a-url-at-all",
			wantOK:    false,
		},
		{
			name:      "missing path",
			remoteURL: "https://github.com/",
			wantOK:    false,
		},
		{
			name:      "only owner no repo",
			remoteURL: "https://github.com/owner",
			wantOK:    false,
		},

		// Segment validation edge cases
		{
			name:      "underscore and tilde are allowed",
			remoteURL: "https://github.com/my_org/my~repo",
			wantOwner: "my_org",
			wantName:  "my~repo",
			wantOK:    true,
		},
		{
			name:      "dot in name (e.g. dotfiles)",
			remoteURL: "https://github.com/user/.dotfiles",
			wantOwner: "user",
			wantName:  ".dotfiles",
			wantOK:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			owner, name, ok := deriveGitHubRepo(tc.remoteURL)
			if ok != tc.wantOK {
				t.Fatalf("deriveGitHubRepo(%q) ok=%v, want %v", tc.remoteURL, ok, tc.wantOK)
			}
			if tc.wantOK {
				if owner != tc.wantOwner {
					t.Errorf("owner=%q, want %q", owner, tc.wantOwner)
				}
				if name != tc.wantName {
					t.Errorf("name=%q, want %q", name, tc.wantName)
				}
			}
		})
	}
}

func TestDeriveSegmentOK(t *testing.T) {
	tests := []struct {
		seg  string
		want bool
	}{
		{"abc", true},
		{"ABC", true},
		{"abc123", true},
		{"my-repo", true},
		{"my_repo", true},
		{"v1.0.0", true},
		{"my~tag", true},
		{"", false},
		{"..", true},   // mirrors segmentOK: dots are allowed; caller context must enforce no traversal
		{"%2e%2e", false}, // percent-encoded → rejected
		{"a/b", false},
		{"a b", false},
		{"a;b", false},
	}
	for _, tc := range tests {
		got := deriveSegmentOK(tc.seg)
		if got != tc.want {
			t.Errorf("deriveSegmentOK(%q)=%v, want %v", tc.seg, got, tc.want)
		}
	}
}
