package backend

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestChromiumPasswordStoreCredentialForGoogle(t *testing.T) {
	profile := &BrowserProfile{
		Username:    "alice@example.com",
		Password:    "secret-pass",
		Platform:    "google",
		PlatformURL: "https://accounts.google.com/",
	}

	credential, ok := chromiumPasswordStoreCredential(profile)
	if !ok {
		t.Fatal("chromiumPasswordStoreCredential() ok = false, want true")
	}
	if credential.originURL != "https://accounts.google.com/" {
		t.Fatalf("originURL = %q, want google origin", credential.originURL)
	}
	if credential.signonRealm != "https://accounts.google.com/" {
		t.Fatalf("signonRealm = %q, want google realm", credential.signonRealm)
	}
	if credential.usernameElement != "identifier" {
		t.Fatalf("usernameElement = %q, want identifier", credential.usernameElement)
	}
	if credential.passwordElement != "Passwd" {
		t.Fatalf("passwordElement = %q, want Passwd", credential.passwordElement)
	}
}

func TestSeedChromiumLoginDataUpsertsPlatformPassword(t *testing.T) {
	loginDataPath := filepath.Join(t.TempDir(), "Default", "Login Data")
	profile := &BrowserProfile{
		Username:    "alice@example.com",
		Password:    "secret-pass",
		Platform:    "google",
		PlatformURL: "https://accounts.google.com/",
	}
	encrypt := func(password string) ([]byte, error) {
		return []byte("enc:" + password), nil
	}
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)

	if err := seedChromiumLoginData(loginDataPath, profile, encrypt, now); err != nil {
		t.Fatalf("seedChromiumLoginData() error = %v", err)
	}
	profile.Password = "changed-pass"
	if err := seedChromiumLoginData(loginDataPath, profile, encrypt, now.Add(time.Minute)); err != nil {
		t.Fatalf("seedChromiumLoginData() update error = %v", err)
	}

	db, err := sql.Open("sqlite", loginDataPath)
	if err != nil {
		t.Fatalf("open Login Data: %v", err)
	}
	defer db.Close()

	var count int
	var usernameValue, passwordElement string
	var passwordValue []byte
	var dateCreated, datePasswordModified int64
	if err := db.QueryRow(`SELECT COUNT(*), username_value, password_element, password_value, date_created, date_password_modified FROM logins`).Scan(&count, &usernameValue, &passwordElement, &passwordValue, &dateCreated, &datePasswordModified); err != nil {
		t.Fatalf("query logins: %v", err)
	}
	if count != 1 {
		t.Fatalf("logins count = %d, want 1", count)
	}
	if usernameValue != "alice@example.com" {
		t.Fatalf("username_value = %q", usernameValue)
	}
	if passwordElement != "Passwd" {
		t.Fatalf("password_element = %q", passwordElement)
	}
	if string(passwordValue) != "enc:changed-pass" {
		t.Fatalf("password_value = %q", string(passwordValue))
	}
	if dateCreated != chromeTime(now) {
		t.Fatalf("date_created = %d, want %d", dateCreated, chromeTime(now))
	}
	if datePasswordModified != chromeTime(now.Add(time.Minute)) {
		t.Fatalf("date_password_modified = %d, want %d", datePasswordModified, chromeTime(now.Add(time.Minute)))
	}
}

func TestChromiumMacOSPasswordEncryptionRoundTrip(t *testing.T) {
	safeStoragePassword := "YbEFI7H6HZlq/gBlV4EekQ=="
	encrypted, err := encryptChromiumMacOSPassword("secret-pass", safeStoragePassword)
	if err != nil {
		t.Fatalf("encryptChromiumMacOSPassword() error = %v", err)
	}
	if !bytes.HasPrefix(encrypted, []byte("v10")) {
		t.Fatalf("encrypted password missing v10 prefix: %x", encrypted[:min(len(encrypted), 3)])
	}

	decrypted, err := decryptChromiumMacOSPassword(encrypted, safeStoragePassword)
	if err != nil {
		t.Fatalf("decryptChromiumMacOSPassword() error = %v", err)
	}
	if decrypted != "secret-pass" {
		t.Fatalf("decrypted = %q, want secret-pass", decrypted)
	}
}
