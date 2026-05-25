package browser

import (
	"crypto/rand"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

var defaultProfileIconPalette = []string{
	"#2563EB", "#059669", "#DC2626", "#D97706", "#7C3AED", "#0D9488",
	"#DB2777", "#4F46E5", "#0891B2", "#65A30D", "#EA580C", "#475569",
}

func NormalizeProfileIconColor(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	if len(value) != 7 {
		return ""
	}
	for _, r := range value[1:] {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return ""
		}
	}
	return strings.ToUpper(value)
}

func ResolveProfileIconColor(input string, seed string) string {
	if color := NormalizeProfileIconColor(input); color != "" {
		return color
	}
	var b [3]byte
	if _, err := rand.Read(b[:]); err == nil {
		return fmt.Sprintf("#%02X%02X%02X", b[0], b[1], b[2])
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(seed)))
	return defaultProfileIconPalette[int(h.Sum32())%len(defaultProfileIconPalette)]
}

func ProfileIconColorRGB(value string) (uint8, uint8, uint8, bool) {
	color := NormalizeProfileIconColor(value)
	if color == "" {
		return 0, 0, 0, false
	}
	r, errR := strconv.ParseUint(color[1:3], 16, 8)
	g, errG := strconv.ParseUint(color[3:5], 16, 8)
	b, errB := strconv.ParseUint(color[5:7], 16, 8)
	if errR != nil || errG != nil || errB != nil {
		return 0, 0, 0, false
	}
	return uint8(r), uint8(g), uint8(b), true
}
