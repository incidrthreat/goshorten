package data

import "strings"

// ParseUserAgent extracts device type, browser, and OS from a user-agent string.
// This is a lightweight parser — no external dependency needed for Shlink-level analytics.
func ParseUserAgent(ua string) (deviceType, browser, os string) {
	lower := strings.ToLower(ua)

	// --- OS detection ---
	switch {
	case strings.Contains(lower, "windows"):
		os = "Windows"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os"):
		os = "macOS"
	case strings.Contains(lower, "iphone"):
		os = "iOS"
	case strings.Contains(lower, "ipad"):
		os = "iPadOS"
	case strings.Contains(lower, "android"):
		os = "Android"
	case strings.Contains(lower, "linux"):
		os = "Linux"
	case strings.Contains(lower, "chromeos") || strings.Contains(lower, "cros"):
		os = "ChromeOS"
	default:
		os = "Other"
	}

	// --- Browser detection (order matters: more specific first) ---
	switch {
	case strings.Contains(lower, "edg/") || strings.Contains(lower, "edge/"):
		browser = "Edge"
	case strings.Contains(lower, "opr/") || strings.Contains(lower, "opera"):
		browser = "Opera"
	case strings.Contains(lower, "vivaldi"):
		browser = "Vivaldi"
	case strings.Contains(lower, "brave"):
		browser = "Brave"
	case strings.Contains(lower, "chrome") && !strings.Contains(lower, "chromium"):
		browser = "Chrome"
	case strings.Contains(lower, "chromium"):
		browser = "Chromium"
	case strings.Contains(lower, "firefox") || strings.Contains(lower, "fxios"):
		browser = "Firefox"
	case strings.Contains(lower, "safari") && !strings.Contains(lower, "chrome"):
		browser = "Safari"
	case strings.Contains(lower, "msie") || strings.Contains(lower, "trident"):
		browser = "IE"
	default:
		browser = "Other"
	}

	// --- Device type detection ---
	switch {
	case strings.Contains(lower, "mobile") || strings.Contains(lower, "iphone") ||
		strings.Contains(lower, "android") && !strings.Contains(lower, "tablet"):
		deviceType = "mobile"
	case strings.Contains(lower, "tablet") || strings.Contains(lower, "ipad"):
		deviceType = "tablet"
	default:
		deviceType = "desktop"
	}

	return deviceType, browser, os
}

// botPatterns contains substrings commonly found in bot/crawler user-agents.
var botPatterns = []string{
	"bot", "crawl", "spider", "slurp", "bingpreview",
	"mediapartners", "adsbot", "feedfetcher", "facebookexternalhit",
	"twitterbot", "linkedinbot", "whatsapp", "telegrambot",
	"discordbot", "applebot", "yandex", "baiduspider",
	"sogou", "exabot", "ia_archiver", "archive.org_bot",
	"semrush", "ahrefs", "mj12bot", "dotbot", "petalbot",
	"bytespider", "gptbot", "chatgpt", "claudebot",
	"curl", "wget", "python-requests", "httpx", "go-http-client",
	"java/", "libwww", "lwp-trivial", "php/", "ruby",
	"scrapy", "httpclient", "okhttp",
}

// IsBot returns true if the user-agent looks like a bot/crawler.
func IsBot(ua string) bool {
	lower := strings.ToLower(ua)
	for _, p := range botPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
