package backend

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"ant-chrome/backend/internal/browser"
)

func TestIsBrowserBlankNewTabURL(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"about:blank",
		"chrome://newtab/",
		"chrome://new-tab-page/",
		"chrome://new-tab-page-third-party/",
	} {
		if !isBrowserBlankNewTabURL(value) {
			t.Fatalf("isBrowserBlankNewTabURL(%q) = false, want true", value)
		}
	}

	for _, value := range []string{
		"https://accounts.google.com/",
		"chrome://password-manager/passwords",
		"http://127.0.0.1:19876/start-pages/42.html",
	} {
		if isBrowserBlankNewTabURL(value) {
			t.Fatalf("isBrowserBlankNewTabURL(%q) = true, want false", value)
		}
	}
}

func TestBrowserNewTabRedirectURLWritesStartPageWithoutExtension(t *testing.T) {
	t.Parallel()

	app := NewApp(t.TempDir())
	app.browserMgr = &browser.Manager{}
	profile := &BrowserProfile{
		ID:          42,
		ProfileId:   "profile-42",
		ProfileName: "New Tab Profile",
	}

	redirectURL, err := app.browserNewTabRedirectURL(profile, time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("browserNewTabRedirectURL() error = %v", err)
	}
	if strings.Contains(redirectURL, "chrome-extension://") || strings.Contains(redirectURL, "--load-extension") {
		t.Fatalf("redirect URL should not use extension: %q", redirectURL)
	}
	parsed, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("redirect URL invalid: %v", err)
	}
	if parsed.Scheme != "http" || parsed.Host != "127.0.0.1:19876" || parsed.Path != "/start-pages/42.html" {
		t.Fatalf("redirect URL = %q, want launch server URL", redirectURL)
	}
	content, err := os.ReadFile(app.resolveAppPath("data/runtime/start-pages/42.html"))
	if err != nil {
		t.Fatalf("start page should be readable: %v", err)
	}
	if !strings.Contains(string(content), "New Tab Profile") {
		t.Fatalf("start page missing profile name:\n%s", content)
	}
}

func TestBrowserDefaultLaunchTargetsOpensPlatformAndStartPage(t *testing.T) {
	t.Parallel()

	app := NewApp(t.TempDir())
	app.config = DefaultConfig()
	app.browserMgr = &browser.Manager{}
	profile := &BrowserProfile{
		ProfileId:   "platform-profile",
		ProfileName: "Platform",
		Platform:    "google",
		PlatformURL: "https://accounts.google.com/",
	}

	targets, err := app.browserDefaultLaunchTargets(newBrowserStartInput(profile.ProfileId, nil, nil, false, false, false, "", ""), profile, false, time.Now())
	if err != nil {
		t.Fatalf("browserDefaultLaunchTargets() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets length = %d, want 2", len(targets))
	}
	if !strings.HasPrefix(targets[0], "http://127.0.0.1:19876/start-pages/") {
		t.Fatalf("first target should be generated start page, got %v", targets)
	}
	parsed, err := url.Parse(targets[1])
	if err != nil {
		t.Fatalf("target URL invalid: %v", err)
	}
	if parsed.Host != "accounts.google.com" {
		t.Fatalf("second target should remain platform URL, got %v", targets)
	}
}
