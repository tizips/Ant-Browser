package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteProfileICNSCreatesIconFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "profile.icns")
	if err := writeProfileICNS(path, "12", "#0D9488"); err != nil {
		t.Fatalf("writeProfileICNS() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) <= 8 {
		t.Fatalf("icns file size = %d, want non-empty icon data", len(data))
	}
	if string(data[:4]) != "icns" {
		t.Fatalf("icns header = %q, want icns", string(data[:4]))
	}
}

func TestProfileBrowserLauncherScriptQuotesChromePath(t *testing.T) {
	t.Parallel()

	script := profileBrowserLauncherScript("/Applications/Google Chrome's Test.app/Contents/MacOS/Google Chrome")
	if !strings.Contains(script, `'/Applications/Google Chrome'\''s Test.app/Contents/MacOS/Google Chrome'`) {
		t.Fatalf("launcher script does not safely quote chrome path: %s", script)
	}
	if !strings.Contains(script, ` "$@"`) {
		t.Fatalf("launcher script does not forward arguments: %s", script)
	}
}

func TestIconDigitsUsesNumericSuffix(t *testing.T) {
	t.Parallel()

	if got := iconDigits("实例 12345"); got != "2345" {
		t.Fatalf("iconDigits() = %q, want 2345", got)
	}
	if got := iconDigits("no-number"); got != "0" {
		t.Fatalf("iconDigits() without number = %q, want 0", got)
	}
}

func TestEnsureProfileBrowserAppUsesInstanceNameForBundlePath(t *testing.T) {
	t.Parallel()

	app := NewApp(t.TempDir())
	profile := &BrowserProfile{
		ID:          376,
		ProfileId:   "54c603ec-9c53-4e93-bfa3-3afa948b644a",
		ProfileName: "tanikajoe90@gmail.com",
		IconColor:   "#5BAAAF",
	}

	appPath, err := app.ensureProfileBrowserApp(profile, "/bin/echo")
	if err != nil {
		t.Fatalf("ensureProfileBrowserApp() error = %v", err)
	}

	if got := filepath.Base(appPath); got != "376 tanikajoe90@gmail.com.app" {
		t.Fatalf("app bundle name = %q, want instance name", got)
	}
}
