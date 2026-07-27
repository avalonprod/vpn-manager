package bot

import "testing"

func TestAutoImportPlatforms(t *testing.T) {
	for _, os := range []string{"ios", "macos", "android"} {
		if !supportsAutoImport(os) {
			t.Errorf("os %q must offer auto import", os)
		}
	}
}

func TestWindowsFallsBackToManualSetup(t *testing.T) {
	if supportsAutoImport("windows") {
		t.Error("windows must not offer auto import: v2rayN registers no URL scheme")
	}
}

func TestUnknownPlatformFallsBackToManualSetup(t *testing.T) {
	for _, os := range []string{"", "linux", "IOS"} {
		if supportsAutoImport(os) {
			t.Errorf("os %q must not offer auto import", os)
		}
	}
}
