package browser

import (
	"ant-chrome/backend/internal/database"
	"path/filepath"
	"testing"
)

func TestSQLiteProfileDAOPreservesRuntimeTimestamps(t *testing.T) {
	t.Parallel()

	db, err := database.NewDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	dao := NewSQLiteProfileDAO(db.GetConn())
	profile := &Profile{
		ProfileId:    "profile-runtime-time",
		ProfileName:  "Runtime Time",
		Username:     "runtime-user",
		Password:     "runtime-pass",
		Platform:     "google",
		PlatformName: "Google",
		PlatformURL:  "https://accounts.google.com/",
		UserDataDir:  "runtime-time",
		TwoFASecret:  "JBSWY3DPEHPK3PXP",
		IconColor:    "#123ABC",
		LastStartAt:  "2026-05-24T16:55:57+08:00",
		LastStopAt:   "2026-05-24T17:05:57+08:00",
	}
	if err := dao.Upsert(profile); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	got, err := dao.GetById(profile.ProfileId)
	if err != nil {
		t.Fatalf("GetById() error = %v", err)
	}
	if got.LastStartAt != profile.LastStartAt || got.LastStopAt != profile.LastStopAt {
		t.Fatalf("GetById() runtime timestamps = (%q, %q), want (%q, %q)", got.LastStartAt, got.LastStopAt, profile.LastStartAt, profile.LastStopAt)
	}
	if got.ID <= 0 {
		t.Fatalf("GetById() id = %d, want positive database id", got.ID)
	}
	if got.TwoFASecret != profile.TwoFASecret {
		t.Fatalf("GetById() twoFaSecret = %q, want %q", got.TwoFASecret, profile.TwoFASecret)
	}
	if got.Username != profile.Username {
		t.Fatalf("GetById() username = %q, want %q", got.Username, profile.Username)
	}
	if got.Password != profile.Password {
		t.Fatalf("GetById() password = %q, want %q", got.Password, profile.Password)
	}
	if got.Platform != profile.Platform || got.PlatformName != profile.PlatformName || got.PlatformURL != profile.PlatformURL {
		t.Fatalf("GetById() platform = (%q, %q, %q), want (%q, %q, %q)", got.Platform, got.PlatformName, got.PlatformURL, profile.Platform, profile.PlatformName, profile.PlatformURL)
	}
	if got.IconColor != profile.IconColor {
		t.Fatalf("GetById() iconColor = %q, want %q", got.IconColor, profile.IconColor)
	}

	list, err := dao.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List() length = %d, want 1", len(list))
	}
	if list[0].LastStartAt != profile.LastStartAt || list[0].LastStopAt != profile.LastStopAt {
		t.Fatalf("List() runtime timestamps = (%q, %q), want (%q, %q)", list[0].LastStartAt, list[0].LastStopAt, profile.LastStartAt, profile.LastStopAt)
	}
	if list[0].ID != got.ID {
		t.Fatalf("List() id = %d, want %d", list[0].ID, got.ID)
	}
	if list[0].TwoFASecret != profile.TwoFASecret {
		t.Fatalf("List() twoFaSecret = %q, want %q", list[0].TwoFASecret, profile.TwoFASecret)
	}
	if list[0].Username != profile.Username {
		t.Fatalf("List() username = %q, want %q", list[0].Username, profile.Username)
	}
	if list[0].Password != profile.Password {
		t.Fatalf("List() password = %q, want %q", list[0].Password, profile.Password)
	}
	if list[0].Platform != profile.Platform || list[0].PlatformName != profile.PlatformName || list[0].PlatformURL != profile.PlatformURL {
		t.Fatalf("List() platform = (%q, %q, %q), want (%q, %q, %q)", list[0].Platform, list[0].PlatformName, list[0].PlatformURL, profile.Platform, profile.PlatformName, profile.PlatformURL)
	}
	if list[0].IconColor != profile.IconColor {
		t.Fatalf("List() iconColor = %q, want %q", list[0].IconColor, profile.IconColor)
	}
}

func TestSQLiteProfileDAOListByGroupIncludesSecretsAndIconColor(t *testing.T) {
	t.Parallel()

	db, err := database.NewDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	dao := NewSQLiteProfileDAO(db.GetConn())
	profile := &Profile{
		ProfileId:    "profile-group-fields",
		ProfileName:  "Group Fields",
		Username:     "group-user",
		Password:     "group-pass",
		Platform:     "facebook",
		PlatformName: "Facebook",
		PlatformURL:  "https://www.facebook.com/",
		UserDataDir:  "group-fields",
		GroupId:      "group-a",
		TwoFASecret:  "JBSWY3DPEHPK3PXP",
		IconColor:    "#0D9488",
	}
	if err := dao.Upsert(profile); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	list, err := dao.ListByGroup("group-a", false, nil)
	if err != nil {
		t.Fatalf("ListByGroup() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByGroup() length = %d, want 1", len(list))
	}
	if list[0].TwoFASecret != profile.TwoFASecret {
		t.Fatalf("ListByGroup() twoFaSecret = %q, want %q", list[0].TwoFASecret, profile.TwoFASecret)
	}
	if list[0].Username != profile.Username {
		t.Fatalf("ListByGroup() username = %q, want %q", list[0].Username, profile.Username)
	}
	if list[0].Password != profile.Password {
		t.Fatalf("ListByGroup() password = %q, want %q", list[0].Password, profile.Password)
	}
	if list[0].Platform != profile.Platform || list[0].PlatformName != profile.PlatformName || list[0].PlatformURL != profile.PlatformURL {
		t.Fatalf("ListByGroup() platform = (%q, %q, %q), want (%q, %q, %q)", list[0].Platform, list[0].PlatformName, list[0].PlatformURL, profile.Platform, profile.PlatformName, profile.PlatformURL)
	}
	if list[0].IconColor != profile.IconColor {
		t.Fatalf("ListByGroup() iconColor = %q, want %q", list[0].IconColor, profile.IconColor)
	}
}
