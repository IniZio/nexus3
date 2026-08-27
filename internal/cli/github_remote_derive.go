package cli

import (
	"net/url"
	"strings"
)

// deriveGitHubRepo parses a git remote URL and returns the GitHub owner and
// repository name when the remote is hosted on github.com. It handles three
// URL forms:
//
//   - HTTPS:   https://github.com/owner/name[.git]
//   - SSH SCP: git@github.com:owner/name[.git]
//   - SSH URL: ssh://git@github.com/owner/name[.git]
//
// ok is false (gracefully, never an error) for any non-github host, empty
// input, or URL that cannot be parsed into exactly two non-empty segments.
//
// SECURITY: each derived segment is validated with deriveSegmentOK (byte-
// identical behaviour to segmentOK in internal/core/perimeter/mitm/proxy.go)
// before being returned, so a crafted remote URL cannot smuggle path-
// injection characters past the caller (AC T3-AC3).
func deriveGitHubRepo(remoteURL string) (owner, name string, ok bool) {
	if remoteURL == "" {
		return "", "", false
	}

	var rawPath string

	switch {
	case strings.HasPrefix(remoteURL, "https://"), strings.HasPrefix(remoteURL, "http://"):
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", false
		}
		if !strings.EqualFold(u.Hostname(), "github.com") {
			return "", "", false
		}
		rawPath = u.Path

	case strings.HasPrefix(remoteURL, "ssh://"):
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", false
		}
		if !strings.EqualFold(u.Hostname(), "github.com") {
			return "", "", false
		}
		rawPath = u.Path

	default:
		// SCP-like form: [user@]host:path
		// Find the colon that separates host from path. The colon must not be
		// preceded by a slash (that would be a port, not SCP).
		colonIdx := strings.Index(remoteURL, ":")
		if colonIdx < 0 {
			return "", "", false
		}
		hostPart := remoteURL[:colonIdx]
		// Strip optional user@ prefix.
		if atIdx := strings.LastIndex(hostPart, "@"); atIdx >= 0 {
			hostPart = hostPart[atIdx+1:]
		}
		if !strings.EqualFold(hostPart, "github.com") {
			return "", "", false
		}
		rawPath = "/" + remoteURL[colonIdx+1:]
	}

	// Normalise: strip leading slash, trailing ".git", then split.
	rawPath = strings.TrimPrefix(rawPath, "/")
	rawPath = strings.TrimSuffix(rawPath, ".git")

	parts := strings.Split(rawPath, "/")
	if len(parts) != 2 {
		return "", "", false
	}

	o, n := parts[0], parts[1]
	if o == "" || n == "" {
		return "", "", false
	}
	if !deriveSegmentOK(o) || !deriveSegmentOK(n) {
		return "", "", false
	}

	return o, n, true
}

// deriveSegmentOK mirrors segmentOK from
// internal/core/perimeter/mitm/proxy.go (byte-identical behaviour).
// Allowed: ASCII letters, digits, dot, underscore, tilde, hyphen.
// Rejected: percent-encoding, slashes, "..", non-ASCII, and empty string.
func deriveSegmentOK(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '_' || c == '~' || c == '-' {
			continue
		}
		return false
	}
	return true
}
