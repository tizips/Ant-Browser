package backend

import (
	"strings"
	"testing"
)

func TestShouldInjectPlatformQuickInputRequiresPlatformURL(t *testing.T) {
	t.Parallel()

	if shouldInjectPlatformQuickInput(&BrowserProfile{Platform: "none", PlatformURL: "https://accounts.google.com/", Username: "user"}) {
		t.Fatal("无平台不应注入快捷输入")
	}
	if shouldInjectPlatformQuickInput(&BrowserProfile{Platform: "google", Username: "user"}) {
		t.Fatal("没有平台链接不应注入快捷输入")
	}
	if shouldInjectPlatformQuickInput(&BrowserProfile{Platform: "google", PlatformURL: "https://accounts.google.com/"}) {
		t.Fatal("没有账号、密码、2FA 内容不应注入快捷输入")
	}
	if !shouldInjectPlatformQuickInput(&BrowserProfile{Platform: "google", PlatformURL: "https://accounts.google.com/", Username: "alice@example.com"}) {
		t.Fatal("有平台链接和账号时应注入快捷输入")
	}
}

func TestPlatformQuickInputScriptIncludesCredentialActions(t *testing.T) {
	t.Parallel()

	script, err := renderPlatformQuickInputScript(&BrowserProfile{
		ProfileName:  "Alice Profile",
		Username:     "alice@example.com",
		Password:     "alice-pass",
		TwoFASecret:  "JBSWY3DPEHPK3PXP",
		Platform:     "google",
		PlatformName: "Google",
		PlatformURL:  "https://accounts.google.com/",
	})
	if err != nil {
		t.Fatalf("renderPlatformQuickInputScript() error = %v", err)
	}

	for _, want := range []string{
		`"username":"alice@example.com"`,
		`"password":"alice-pass"`,
		`"twoFaSecret":"JBSWY3DPEHPK3PXP"`,
		`账号`,
		`密码`,
		`2FA`,
		`fillCredential`,
		`one-time-code`,
		`input[type="password"]`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("quick input script missing %q\n%s", want, script)
		}
	}
}

func TestPlatformQuickInputMatchesPlatformTargetURL(t *testing.T) {
	t.Parallel()

	profile := &BrowserProfile{
		Platform:    "google",
		PlatformURL: "https://accounts.google.com/",
	}

	for _, targetURL := range []string{
		"https://accounts.google.com/v3/signin/identifier",
		"https://www.accounts.google.com/signin",
	} {
		if !platformQuickInputMatchesTargetURL(profile, targetURL) {
			t.Fatalf("platformQuickInputMatchesTargetURL(%q) = false, want true", targetURL)
		}
	}

	for _, targetURL := range []string{
		"chrome://password-manager/passwords",
		"http://127.0.0.1:19876/start-pages/376.html",
		"https://example.com/",
	} {
		if platformQuickInputMatchesTargetURL(profile, targetURL) {
			t.Fatalf("platformQuickInputMatchesTargetURL(%q) = true, want false", targetURL)
		}
	}
}
