package backend

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ant-chrome/backend/internal/logger"
	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite"
)

type chromiumPasswordStoreEntry struct {
	originURL        string
	actionURL        string
	usernameElement  string
	usernameValue    string
	passwordElement  string
	passwordValue    string
	signonRealm      string
	dateCreated      time.Time
	dateLastModified time.Time
}

type chromiumPasswordEncryptor func(password string) ([]byte, error)

func (a *App) seedPlatformPasswordStore(profile *BrowserProfile, userDataDir string) {
	loginDataPath := filepath.Join(userDataDir, "Default", "Login Data")
	encrypt, err := chromiumPasswordEncryptorForOS()
	if err != nil {
		if shouldSeedChromiumLoginData(profile) {
			logger.New("Browser").Warn("平台账号密码未写入浏览器密码管理器",
				logger.F("profile_id", profile.ProfileId),
				logger.F("platform", profile.Platform),
				logger.F("error", err.Error()),
			)
		}
		return
	}
	if err := seedChromiumLoginData(loginDataPath, profile, encrypt, time.Now()); err != nil {
		logger.New("Browser").Warn("平台账号密码写入浏览器密码管理器失败",
			logger.F("profile_id", profile.ProfileId),
			logger.F("platform", profile.Platform),
			logger.F("path", loginDataPath),
			logger.F("error", err.Error()),
		)
	}
}

func shouldSeedChromiumLoginData(profile *BrowserProfile) bool {
	if profile == nil {
		return false
	}
	return browserProfilePlatformURL(profile) != "" &&
		strings.TrimSpace(profile.Username) != "" &&
		strings.TrimSpace(profile.Password) != ""
}

func seedChromiumLoginData(loginDataPath string, profile *BrowserProfile, encrypt chromiumPasswordEncryptor, now time.Time) error {
	if !shouldSeedChromiumLoginData(profile) {
		return nil
	}
	if encrypt == nil {
		return errors.New("password encryptor is nil")
	}
	credential, ok := chromiumPasswordStoreCredential(profile)
	if !ok {
		return nil
	}
	encryptedPassword, err := encrypt(credential.passwordValue)
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}
	credential.dateLastModified = now

	if err := os.MkdirAll(filepath.Dir(loginDataPath), 0o755); err != nil {
		return fmt.Errorf("create Login Data dir: %w", err)
	}
	db, err := sql.Open("sqlite", loginDataPath)
	if err != nil {
		return fmt.Errorf("open Login Data: %w", err)
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA busy_timeout = 3000`); err != nil {
		return fmt.Errorf("set busy_timeout: %w", err)
	}
	if err := ensureChromiumLoginDataSchema(db); err != nil {
		return err
	}
	if credential.dateCreated.IsZero() {
		credential.dateCreated = now
	}

	_, err = db.Exec(`
INSERT INTO logins (
  origin_url, action_url, username_element, username_value, password_element, password_value,
  submit_element, signon_realm, date_created, blacklisted_by_user, scheme, password_type,
  times_used, form_data, display_name, icon_url, federation_url, skip_zero_click,
  generation_upload_status, possible_username_pairs, date_last_used, moving_blocked_for,
  date_password_modified, sender_email, sender_name, date_received, sharing_notification_displayed,
  keychain_identifier, sender_profile_image_url, date_last_filled, actor_login_approved
) VALUES (
  ?, ?, ?, ?, ?, ?,
  '', ?, ?, 0, 0, 0,
  0, NULL, '', '', '', 0,
  0, NULL, 0, NULL,
  ?, '', '', 0, 0,
  NULL, '', 0, 0
)
ON CONFLICT(origin_url, username_element, username_value, password_element, signon_realm)
DO UPDATE SET
  action_url = excluded.action_url,
  password_value = excluded.password_value,
  date_password_modified = excluded.date_password_modified
`, credential.originURL, credential.actionURL, credential.usernameElement, credential.usernameValue, credential.passwordElement, encryptedPassword,
		credential.signonRealm, chromeTime(credential.dateCreated), chromeTime(credential.dateLastModified))
	if err != nil {
		return fmt.Errorf("upsert Login Data row: %w", err)
	}
	return nil
}

func ensureChromiumLoginDataSchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS meta(key LONGVARCHAR NOT NULL UNIQUE PRIMARY KEY, value LONGVARCHAR)`); err != nil {
		return fmt.Errorf("create meta table: %w", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO meta(key, value) VALUES ('mmap_status', '-1'), ('version', '43'), ('last_compatible_version', '40')`); err != nil {
		return fmt.Errorf("seed meta table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS logins (
origin_url VARCHAR NOT NULL,
action_url VARCHAR,
username_element VARCHAR,
username_value VARCHAR,
password_element VARCHAR,
password_value BLOB,
submit_element VARCHAR,
signon_realm VARCHAR NOT NULL,
date_created INTEGER NOT NULL,
blacklisted_by_user INTEGER NOT NULL,
scheme INTEGER NOT NULL,
password_type INTEGER,
times_used INTEGER,
form_data BLOB,
display_name VARCHAR,
icon_url VARCHAR,
federation_url VARCHAR,
skip_zero_click INTEGER,
generation_upload_status INTEGER,
possible_username_pairs BLOB,
id INTEGER PRIMARY KEY AUTOINCREMENT,
date_last_used INTEGER NOT NULL DEFAULT 0,
moving_blocked_for BLOB,
date_password_modified INTEGER NOT NULL DEFAULT 0,
sender_email VARCHAR,
sender_name VARCHAR,
date_received INTEGER,
sharing_notification_displayed INTEGER NOT NULL DEFAULT 0,
keychain_identifier BLOB,
sender_profile_image_url VARCHAR,
date_last_filled INTEGER NOT NULL DEFAULT 0,
actor_login_approved INTEGER NOT NULL DEFAULT 0,
UNIQUE (origin_url, username_element, username_value, password_element, signon_realm)
)`); err != nil {
		return fmt.Errorf("create logins table: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS logins_signon ON logins (signon_realm)`); err != nil {
		return fmt.Errorf("create logins_signon index: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sync_entities_metadata (storage_key INTEGER PRIMARY KEY AUTOINCREMENT, metadata VARCHAR NOT NULL)`); err != nil {
		return fmt.Errorf("create sync_entities_metadata table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sync_model_metadata (id INTEGER PRIMARY KEY AUTOINCREMENT, model_metadata VARCHAR NOT NULL)`); err != nil {
		return fmt.Errorf("create sync_model_metadata table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS insecure_credentials (
parent_id INTEGER REFERENCES logins ON UPDATE CASCADE ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
insecurity_type INTEGER NOT NULL,
create_time INTEGER NOT NULL,
is_muted INTEGER NOT NULL DEFAULT 0,
trigger_notification_from_backend INTEGER NOT NULL DEFAULT 0,
UNIQUE (parent_id, insecurity_type)
)`); err != nil {
		return fmt.Errorf("create insecure_credentials table: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS foreign_key_index ON insecure_credentials (parent_id)`); err != nil {
		return fmt.Errorf("create insecure_credentials index: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS password_notes (
id INTEGER PRIMARY KEY AUTOINCREMENT,
parent_id INTEGER NOT NULL REFERENCES logins ON UPDATE CASCADE ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
key VARCHAR NOT NULL,
value BLOB,
date_created INTEGER NOT NULL,
confidential INTEGER,
UNIQUE (parent_id, key)
)`); err != nil {
		return fmt.Errorf("create password_notes table: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS foreign_key_index_notes ON password_notes (parent_id)`); err != nil {
		return fmt.Errorf("create password_notes index: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS stats (
origin_domain VARCHAR NOT NULL,
username_value VARCHAR,
dismissal_count INTEGER,
update_time INTEGER NOT NULL,
UNIQUE(origin_domain, username_value)
)`); err != nil {
		return fmt.Errorf("create stats table: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS stats_origin ON stats(origin_domain)`); err != nil {
		return fmt.Errorf("create stats_origin index: %w", err)
	}
	return nil
}

func chromiumPasswordStoreCredential(profile *BrowserProfile) (chromiumPasswordStoreEntry, bool) {
	if !shouldSeedChromiumLoginData(profile) {
		return chromiumPasswordStoreEntry{}, false
	}
	platformURL := browserProfilePlatformURL(profile)
	parsed, err := url.Parse(platformURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return chromiumPasswordStoreEntry{}, false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return chromiumPasswordStoreEntry{}, false
	}
	origin := scheme + "://" + strings.ToLower(parsed.Host) + "/"
	entry := chromiumPasswordStoreEntry{
		originURL:       origin,
		actionURL:       origin,
		usernameElement: "username",
		usernameValue:   strings.TrimSpace(profile.Username),
		passwordElement: "password",
		passwordValue:   strings.TrimSpace(profile.Password),
		signonRealm:     origin,
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	if strings.EqualFold(strings.TrimSpace(profile.Platform), "google") || host == "accounts.google.com" || strings.HasSuffix(host, ".google.com") {
		entry.usernameElement = "identifier"
		entry.passwordElement = "Passwd"
		entry.actionURL = "https://accounts.google.com/signin/v2/challenge/pwd"
	}
	return entry, true
}

func chromiumPasswordEncryptorForOS() (chromiumPasswordEncryptor, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("当前系统暂未支持写入 Chromium 原生密码库: %s", runtime.GOOS)
	}
	safeStoragePassword, err := macOSChromiumSafeStoragePassword()
	if err != nil {
		return nil, err
	}
	return func(password string) ([]byte, error) {
		return encryptChromiumMacOSPassword(password, safeStoragePassword)
	}, nil
}

func macOSChromiumSafeStoragePassword() (string, error) {
	services := []string{"Chromium Safe Storage", "Chrome Safe Storage"}
	for _, service := range services {
		out, err := exec.Command("security", "find-generic-password", "-w", "-s", service).Output()
		if err == nil {
			if password := strings.TrimSpace(string(out)); password != "" {
				return password, nil
			}
		}
	}
	return "", errors.New("未找到 Chromium Safe Storage 密钥")
}

func encryptChromiumMacOSPassword(password string, safeStoragePassword string) ([]byte, error) {
	key := pbkdf2.Key([]byte(safeStoragePassword), []byte("saltysalt"), 1003, 16, sha1.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad([]byte(password), aes.BlockSize)
	cipherText := make([]byte, len(padded))
	iv := bytes.Repeat([]byte(" "), aes.BlockSize)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(cipherText, padded)
	return append([]byte("v10"), cipherText...), nil
}

func decryptChromiumMacOSPassword(encrypted []byte, safeStoragePassword string) (string, error) {
	if !bytes.HasPrefix(encrypted, []byte("v10")) {
		return "", errors.New("encrypted password missing v10 prefix")
	}
	cipherText := encrypted[3:]
	if len(cipherText) == 0 || len(cipherText)%aes.BlockSize != 0 {
		return "", errors.New("invalid encrypted password length")
	}
	key := pbkdf2.Key([]byte(safeStoragePassword), []byte("saltysalt"), 1003, 16, sha1.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plain := make([]byte, len(cipherText))
	iv := bytes.Repeat([]byte(" "), aes.BlockSize)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, cipherText)
	unpadded, err := pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", err
	}
	return string(unpadded), nil
}

func pkcs7Pad(value []byte, blockSize int) []byte {
	padding := blockSize - len(value)%blockSize
	out := make([]byte, len(value)+padding)
	copy(out, value)
	for i := len(value); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, errors.New("invalid pkcs7 length")
	}
	padding := int(value[len(value)-1])
	if padding == 0 || padding > blockSize || padding > len(value) {
		return nil, errors.New("invalid pkcs7 padding")
	}
	for _, b := range value[len(value)-padding:] {
		if int(b) != padding {
			return nil, errors.New("invalid pkcs7 padding")
		}
	}
	return value[:len(value)-padding], nil
}

func chromeTime(t time.Time) int64 {
	chromeEpoch := time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)
	return t.UTC().Sub(chromeEpoch).Microseconds()
}
