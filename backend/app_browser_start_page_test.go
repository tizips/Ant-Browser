package backend

import (
	"ant-chrome/backend/internal/browser"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestShouldUseBrowserStartPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		startURLs            []string
		defaultStartURLs     []string
		skipDefaultStartURLs bool
		restoreLastSession   bool
		want                 bool
	}{
		{
			name: "plain launch without targets",
			want: true,
		},
		{
			name:      "explicit start urls win",
			startURLs: []string{"https://example.com"},
			want:      false,
		},
		{
			name:             "configured default urls win",
			defaultStartURLs: []string{"https://example.com"},
			want:             false,
		},
		{
			name:                 "skip default urls keeps blank launch behavior",
			skipDefaultStartURLs: true,
			want:                 false,
		},
		{
			name:               "restore session keeps chrome session behavior",
			restoreLastSession: true,
			want:               false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldUseBrowserStartPage(tt.startURLs, tt.defaultStartURLs, tt.skipDefaultStartURLs, tt.restoreLastSession)
			if got != tt.want {
				t.Fatalf("shouldUseBrowserStartPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrowserStartPageURLWritesProfileInfoPage(t *testing.T) {
	t.Parallel()

	app := NewApp(t.TempDir())
	app.browserMgr = &browser.Manager{
		GroupDAO: &startPageGroupDAOStub{
			groups: map[string]*browser.Group{
				"group-gmail": {
					GroupId:   "group-gmail",
					GroupName: "Gmail_替补",
				},
			},
		},
	}

	profile := &BrowserProfile{
		ID:           408,
		ProfileId:    "408",
		ProfileName:  "tanikajoe90@gmail.com",
		Username:     "tanikajoe90",
		Password:     "pass-408",
		Platform:     "google",
		PlatformName: "Google",
		PlatformURL:  "https://accounts.google.com/",
		GroupId:      "group-gmail",
		Tags:         []string{"Gmail", "替补"},
		Keywords:     []string{"buyer-408", "gmail"},
		TwoFASecret:  "JBSWY3DPEHPK3PXP",
		ProxyConfig:  "http://127.0.0.1:2260",
		LaunchCode:   "CODE408",
		LastStartAt:  "2026-05-24T16:55:57+08:00",
		FingerprintArgs: []string{
			"--lang=en,en-US;q=0.9",
			"--timezone=Asia/Tokyo",
			"--user-agent=Mozilla/5.0 Test",
		},
	}

	pageURL, err := app.browserStartPageURL(profile, time.Date(2026, 5, 24, 16, 55, 57, 0, time.FixedZone("CST", 8*60*60)))
	if err != nil {
		t.Fatalf("browserStartPageURL returned error: %v", err)
	}

	parsed, err := url.Parse(pageURL)
	if err != nil {
		t.Fatalf("invalid start page URL %q: %v", pageURL, err)
	}
	if parsed.Scheme != "http" || parsed.Host != "127.0.0.1:19876" || parsed.Path != "/start-pages/408.html" {
		t.Fatalf("start page URL = %q, want launch server URL", pageURL)
	}

	content, err := os.ReadFile(app.resolveAppPath("data/runtime/start-pages/408.html"))
	if err != nil {
		t.Fatalf("expected start page file to be readable: %v", err)
	}
	html := string(content)
	for _, want := range []string{
		"<title>408 tanikajoe90@gmail.com</title>",
		"序号:",
		"408",
		"实例名称:",
		"tanikajoe90@gmail.com",
		"用户名:",
		"tanikajoe90",
		"密码:",
		"pass-408",
		"平台:",
		"Google",
		"https://accounts.google.com/",
		"2FA验证码:",
		"data-secret=\"JBSWY3DPEHPK3PXP\"",
		"Gmail_替补",
		"Gmail, 替补",
		"关键词:",
		"buyer-408, gmail",
		"UserAgent:",
		"Mozilla/5.0 Test",
		"2026-05-24 16:55:57",
		"en,en-US;q=0.9",
		"Asia/Tokyo",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("start page HTML missing %q\n%s", want, html)
		}
	}
	for _, notWant := range []string{
		"快捷打开码:",
		"代理:",
		"备注:",
		"窗口名称:",
		"http://127.0.0.1:2260",
		"CODE408",
		"未设置2FA密钥",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("start page HTML should not contain %q\n%s", notWant, html)
		}
	}
}

func TestBrowserStartPageModelDefaultsUsernameToProfileName(t *testing.T) {
	t.Parallel()

	app := NewApp(t.TempDir())
	profile := &BrowserProfile{ProfileId: "profile-1", ProfileName: "buyer-001"}

	model := app.browserStartPageModel(profile, time.Now())

	if model.Username != "buyer-001" {
		t.Fatalf("Username = %q, want profile name fallback", model.Username)
	}
}

func TestBrowserStartPageFallsBackToNavigatorUserAgent(t *testing.T) {
	t.Parallel()

	html, err := renderBrowserStartPageHTML(browserStartPageModel{
		Title:       "buyer-001",
		Serial:      "1",
		ProfileName: "buyer-001",
		Username:    "buyer-001",
	})
	if err != nil {
		t.Fatalf("renderBrowserStartPageHTML() error = %v", err)
	}

	for _, want := range []string{
		`id="user-agent"`,
		`navigator.userAgent`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("start page HTML missing %q\n%s", want, html)
		}
	}
}

func TestBrowserStartPageTOTPCodeUsesRFCVector(t *testing.T) {
	t.Parallel()

	code, err := browserStartPageTOTPCode("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0))
	if err != nil {
		t.Fatalf("browserStartPageTOTPCode returned error: %v", err)
	}
	if code != "287082" {
		t.Fatalf("TOTP code = %q, want 287082", code)
	}
}

func TestBrowserStartPageTOTPCodeAcceptsGoogleAuthenticatorURI(t *testing.T) {
	t.Parallel()

	code, err := browserStartPageTOTPCode("otpauth://totp/demo?secret=GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ&issuer=Ant", time.Unix(59, 0))
	if err != nil {
		t.Fatalf("browserStartPageTOTPCode returned error: %v", err)
	}
	if code != "287082" {
		t.Fatalf("TOTP code = %q, want 287082", code)
	}
}

func TestBrowserStartPageTOTPCodeRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	if _, err := browserStartPageTOTPCode("not a base32 secret!", time.Unix(59, 0)); err == nil {
		t.Fatal("browserStartPageTOTPCode should reject invalid base32 secret")
	}
}

func TestBrowserDefaultLaunchTargetsInjectsStartPageOnlyForPlainLaunch(t *testing.T) {
	t.Parallel()

	app := NewApp(t.TempDir())
	app.config = DefaultConfig()
	app.browserMgr = &browser.Manager{}

	profile := &BrowserProfile{
		ProfileId:   "profile-408",
		ProfileName: "plain-launch",
	}
	launchTime := time.Date(2026, 5, 24, 16, 55, 57, 0, time.UTC)

	targets, err := app.browserDefaultLaunchTargets(newBrowserStartInput(profile.ProfileId, nil, nil, false, false, false, "", ""), profile, false, launchTime)
	if err != nil {
		t.Fatalf("browserDefaultLaunchTargets returned error: %v", err)
	}
	if len(targets) != 1 || !strings.HasPrefix(targets[0], "http://127.0.0.1:19876/start-pages/") {
		t.Fatalf("plain launch should use generated start page target, got %v", targets)
	}

	explicitTargets, err := app.browserDefaultLaunchTargets(newBrowserStartInput(profile.ProfileId, nil, []string{"https://example.com"}, false, false, false, "", ""), profile, false, launchTime)
	if err != nil {
		t.Fatalf("browserDefaultLaunchTargets explicit returned error: %v", err)
	}
	if len(explicitTargets) != 0 {
		t.Fatalf("explicit start URL should not inject default start page, got %v", explicitTargets)
	}

	restoreTargets, err := app.browserDefaultLaunchTargets(newBrowserStartInput(profile.ProfileId, nil, nil, false, false, false, "", ""), profile, true, launchTime)
	if err != nil {
		t.Fatalf("browserDefaultLaunchTargets restore returned error: %v", err)
	}
	if len(restoreTargets) != 0 {
		t.Fatalf("session restore should not inject default start page, got %v", restoreTargets)
	}
}

func TestBrowserDefaultLaunchTargetsUsesProfilePlatformURLForPlainLaunch(t *testing.T) {
	t.Parallel()

	app := NewApp(t.TempDir())
	app.config = DefaultConfig()
	app.browserMgr = &browser.Manager{}

	profile := &BrowserProfile{
		ProfileId:    "profile-platform",
		ProfileName:  "platform-launch",
		Platform:     "google",
		PlatformName: "Google",
		PlatformURL:  "accounts.google.com",
	}

	targets, err := app.browserDefaultLaunchTargets(newBrowserStartInput(profile.ProfileId, nil, nil, false, false, false, "", ""), profile, false, time.Now())
	if err != nil {
		t.Fatalf("browserDefaultLaunchTargets returned error: %v", err)
	}
	if len(targets) != 2 || !strings.HasPrefix(targets[0], "http://127.0.0.1:19876/start-pages/") || targets[1] != "https://accounts.google.com" {
		t.Fatalf("plain launch with platform should use generated start page plus platform URL, got %v", targets)
	}

	explicitTargets, err := app.browserDefaultLaunchTargets(newBrowserStartInput(profile.ProfileId, nil, []string{"https://example.com"}, false, false, false, "", ""), profile, false, time.Now())
	if err != nil {
		t.Fatalf("browserDefaultLaunchTargets explicit returned error: %v", err)
	}
	if len(explicitTargets) != 0 {
		t.Fatalf("explicit launch should not inject platform URL, got %v", explicitTargets)
	}
}

type startPageGroupDAOStub struct {
	groups map[string]*browser.Group
}

func (s *startPageGroupDAOStub) List() ([]*browser.Group, error) {
	groups := make([]*browser.Group, 0, len(s.groups))
	for _, group := range s.groups {
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *startPageGroupDAOStub) GetById(groupId string) (*browser.Group, error) {
	if group, ok := s.groups[groupId]; ok {
		return group, nil
	}
	return nil, os.ErrNotExist
}

func (s *startPageGroupDAOStub) Create(input browser.GroupInput) (*browser.Group, error) {
	return nil, os.ErrPermission
}

func (s *startPageGroupDAOStub) Update(groupId string, input browser.GroupInput) (*browser.Group, error) {
	return nil, os.ErrPermission
}

func (s *startPageGroupDAOStub) Delete(groupId string) error {
	return os.ErrPermission
}

func (s *startPageGroupDAOStub) GetChildren(parentId string) ([]*browser.Group, error) {
	return nil, nil
}

func (s *startPageGroupDAOStub) MoveChildren(fromGroupId, toGroupId string) error {
	return nil
}
