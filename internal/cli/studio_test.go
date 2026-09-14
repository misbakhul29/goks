package cli

import (
	"testing"
)

func TestStudioCmd_Flags(t *testing.T) {
	cmd := StudioCmd()
	if cmd.Use != "studio" {
		t.Errorf("Expected command use 'studio', got %s", cmd.Use)
	}

	pFlag := cmd.Flags().Lookup("port")
	if pFlag == nil || pFlag.DefValue != "4983" {
		t.Errorf("Expected default port 4983, got %v", pFlag)
	}

	noOpenFlag := cmd.Flags().Lookup("no-open")
	if noOpenFlag == nil {
		t.Errorf("Expected no-open flag to exist")
	}
}
