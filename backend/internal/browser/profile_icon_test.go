package browser

import "testing"

func TestNormalizeProfileIconColor(t *testing.T) {
	t.Parallel()

	if got := NormalizeProfileIconColor("  1a2b3c "); got != "#1A2B3C" {
		t.Fatalf("NormalizeProfileIconColor() = %q, want #1A2B3C", got)
	}
	if got := NormalizeProfileIconColor("#GGGGGG"); got != "" {
		t.Fatalf("NormalizeProfileIconColor() invalid = %q, want empty", got)
	}
}

func TestResolveProfileIconColorReturnsHexColor(t *testing.T) {
	t.Parallel()

	got := ResolveProfileIconColor("", "profile-1")
	if NormalizeProfileIconColor(got) == "" {
		t.Fatalf("ResolveProfileIconColor() = %q, want hex color", got)
	}
}
