package shortener

import (
	"net/url"
	"strings"
)

// NormalizeURL validates and normalizes a URL.
// Adds scheme if missing, lowercases the host, removes trailing slash from path.
func NormalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errNoURL
	}

	// Add scheme if missing
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", errInvalidURL
	}

	if u.Host == "" {
		return "", errInvalidURL
	}

	// Lowercase the host
	u.Host = strings.ToLower(u.Host)

	// Remove trailing slash from path (except root)
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	return u.String(), nil
}

// ValidateRedirectType checks that the redirect type is one of the valid HTTP redirect codes.
func ValidateRedirectType(rt int32) int32 {
	switch rt {
	case 301, 302, 307, 308:
		return rt
	default:
		return 302
	}
}

// ValidateCustomSlug checks that a custom slug is safe and within bounds.
func ValidateCustomSlug(slug string) error {
	if len(slug) < 3 {
		return errSlugTooShort
	}
	if len(slug) > 100 {
		return errSlugTooLong
	}
	for _, c := range slug {
		if !isValidSlugChar(c) {
			return errSlugInvalidChars
		}
	}
	return nil
}

func isValidSlugChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_'
}
