package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"ant-chrome/backend/internal/browser"
)

func (a *App) newBrowserLaunchCommand(plan *browserStartPlan) (*exec.Cmd, error) {
	if runtime.GOOS == "darwin" {
		if appPath, err := a.ensureProfileBrowserApp(plan.profile, plan.chromeBinaryPath); err == nil && appPath != "" {
			args := append([]string{"-W", "-n", appPath, "--args"}, plan.args...)
			return exec.Command("/usr/bin/open", args...), nil
		}
	}
	cmd := exec.Command(plan.chromeBinaryPath, plan.args...)
	cmd.Dir = filepath.Dir(plan.chromeBinaryPath)
	return cmd, nil
}

func (a *App) ensureProfileBrowserApp(profile *BrowserProfile, chromeBinaryPath string) (string, error) {
	if profile == nil || strings.TrimSpace(chromeBinaryPath) == "" {
		return "", nil
	}
	serial := browserStartPageSerial(profile)
	if serial == "" {
		serial = safeStartPageFileName(profile.ProfileId)
	}
	colorValue := browser.ResolveProfileIconColor(profile.IconColor, profile.ProfileId)
	profile.IconColor = colorValue

	displayName := profileBrowserDisplayName(profile, serial)
	appDir := a.resolveAppPath(filepath.ToSlash(filepath.Join("data", "runtime", "profile-apps", safeProfileAppBundleName(displayName)+".app")))
	contentsDir := filepath.Join(appDir, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	resourcesDir := filepath.Join(contentsDir, "Resources")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		return "", err
	}

	iconPath := filepath.Join(resourcesDir, "profile.icns")
	if err := writeProfileICNS(iconPath, serial, colorValue); err != nil {
		return "", err
	}
	executablePath := filepath.Join(macOSDir, "profile-browser")
	if err := os.WriteFile(executablePath, []byte(profileBrowserLauncherScript(chromeBinaryPath)), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(contentsDir, "Info.plist"), []byte(profileBrowserInfoPlist(profile, serial)), 0o644); err != nil {
		return "", err
	}
	return appDir, nil
}

func profileBrowserLauncherScript(chromeBinaryPath string) string {
	return "#!/bin/sh\nexec " + shellQuote(chromeBinaryPath) + " \"$@\"\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func profileBrowserDisplayName(profile *BrowserProfile, serial string) string {
	profileName := ""
	if profile != nil {
		profileName = strings.TrimSpace(profile.ProfileName)
	}
	displayName := strings.TrimSpace(strings.TrimSpace(serial) + " " + profileName)
	if displayName == "" {
		displayName = "Ant Browser"
	}
	return displayName
}

func safeProfileAppBundleName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Ant Browser"
	}
	var b strings.Builder
	for _, r := range value {
		switch r {
		case '/', ':':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	safe := strings.Trim(strings.TrimSpace(b.String()), ".")
	if safe == "" {
		return "Ant Browser"
	}
	return safe
}

func profileBrowserInfoPlist(profile *BrowserProfile, serial string) string {
	bundleID := "cn.reelix.antbrowser.profile." + safeStartPageFileName(profile.ProfileId)
	displayName := profileBrowserDisplayName(profile, serial)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>zh_CN</string>
  <key>CFBundleExecutable</key>
  <string>profile-browser</string>
  <key>CFBundleIconFile</key>
  <string>profile.icns</string>
  <key>CFBundleIdentifier</key>
  <string>%s</string>
  <key>CFBundleName</key>
  <string>%s</string>
  <key>CFBundleDisplayName</key>
  <string>%s</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>1.0</string>
  <key>CFBundleVersion</key>
  <string>1</string>
  <key>LSMinimumSystemVersion</key>
  <string>10.13</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
`, templateEscape(bundleID), templateEscape(displayName), templateEscape(displayName))
}

func templateEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return replacer.Replace(value)
}

func writeProfileICNS(path string, label string, colorValue string) error {
	var chunks bytes.Buffer
	for _, item := range []struct {
		typ  string
		size int
	}{
		{"ic07", 128},
		{"ic08", 256},
		{"ic09", 512},
		{"ic10", 1024},
	} {
		pngData, err := renderProfileIconPNG(item.size, label, colorValue)
		if err != nil {
			return err
		}
		chunks.WriteString(item.typ)
		_ = binary.Write(&chunks, binary.BigEndian, uint32(len(pngData)+8))
		chunks.Write(pngData)
	}

	var out bytes.Buffer
	out.WriteString("icns")
	_ = binary.Write(&out, binary.BigEndian, uint32(chunks.Len()+8))
	out.Write(chunks.Bytes())
	return os.WriteFile(path, out.Bytes(), 0o644)
}

func renderProfileIconPNG(size int, label string, colorValue string) ([]byte, error) {
	r, g, b, ok := browser.ProfileIconColorRGB(colorValue)
	if !ok {
		r, g, b = 37, 99, 235
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: r, G: g, B: b, A: 255}}, image.Point{}, draw.Src)

	digits := iconDigits(label)
	scale := max(1, size/170)
	width := len(digits)*digitWidth(scale) + max(0, len(digits)-1)*digitGap(scale)
	height := digitHeight(scale)
	x := (size - width) / 2
	y := (size - height) / 2
	for _, digit := range digits {
		drawDigit(img, digit, x, y, scale, color.RGBA{255, 255, 255, 255})
		x += digitWidth(scale) + digitGap(scale)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func iconDigits(label string) string {
	var digits strings.Builder
	for _, r := range label {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	value := digits.String()
	if value == "" {
		return "0"
	}
	if len(value) > 4 {
		return value[len(value)-4:]
	}
	return value
}

func digitWidth(scale int) int  { return 18 * scale }
func digitHeight(scale int) int { return 34 * scale }
func digitGap(scale int) int    { return 5 * scale }

var digitSegments = map[rune]string{
	'0': "abcedf",
	'1': "bc",
	'2': "abged",
	'3': "abgcd",
	'4': "fgbc",
	'5': "afgcd",
	'6': "afgecd",
	'7': "abc",
	'8': "abcdefg",
	'9': "abfgcd",
}

func drawDigit(img *image.RGBA, digit rune, x int, y int, scale int, c color.Color) {
	segments := digitSegments[digit]
	for _, segment := range segments {
		drawSegment(img, segment, x, y, scale, c)
	}
}

func drawSegment(img *image.RGBA, segment rune, x int, y int, scale int, c color.Color) {
	thick := 4 * scale
	long := 14 * scale
	short := 13 * scale
	switch segment {
	case 'a':
		fillRect(img, x+2*scale, y, long, thick, c)
	case 'b':
		fillRect(img, x+16*scale-thick, y+2*scale, thick, short, c)
	case 'c':
		fillRect(img, x+16*scale-thick, y+18*scale, thick, short, c)
	case 'd':
		fillRect(img, x+2*scale, y+30*scale, long, thick, c)
	case 'e':
		fillRect(img, x, y+18*scale, thick, short, c)
	case 'f':
		fillRect(img, x, y+2*scale, thick, short, c)
	case 'g':
		fillRect(img, x+2*scale, y+15*scale, long, thick, c)
	}
}

func fillRect(img *image.RGBA, x int, y int, w int, h int, c color.Color) {
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{C: c}, image.Point{}, draw.Src)
}
