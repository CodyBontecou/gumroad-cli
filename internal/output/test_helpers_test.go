package output

import "testing"

func setColorEnabledForTest(t *testing.T, enabled bool) {
	t.Helper()
	SetColorEnabledForTesting(enabled)
	t.Cleanup(ResetColorEnabledForTesting)
}

func resetColorEnabledForTest(t *testing.T) {
	t.Helper()
	ResetColorEnabledForTesting()
	t.Cleanup(ResetColorEnabledForTesting)
}
