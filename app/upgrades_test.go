package app

import (
	"testing"
)

func TestUpgradeV060PlanName(t *testing.T) {
	if UpgradeV060 != "v0.6.0" {
		t.Fatalf("expected upgrade plan name 'v0.6.0', got '%s'", UpgradeV060)
	}
}

func TestUpgradeHandlerRegistration(t *testing.T) {
	// Verify the upgrade handler function is well-formed (no nil panic).
	app := &OasyceApp{}
	handler := app.upgradeHandlerV060()
	if handler == nil {
		t.Fatal("upgradeHandlerV060 returned nil")
	}
}
