package workspace

import "testing"

func TestPresetCapabilities(t *testing.T) {
	tests := []struct {
		role         Role
		wantContains []Capability
		wantLacks    []Capability
	}{
		{RoleViewer, []Capability{CapRead}, []Capability{CapCreate, CapDelete, CapManageWorkspace}},
		{RoleEditor, []Capability{CapRead, CapCreate, CapPublish}, []Capability{CapDelete, CapManageMembers}},
		{RoleAdmin, []Capability{CapDelete, CapManageChannels, CapManageMembers}, []Capability{CapManageWorkspace}},
		{RoleOwner, []Capability{CapManageWorkspace, CapManageMembers, CapRead}, nil},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			caps := PresetCapabilities(tt.role)
			for _, want := range tt.wantContains {
				if !Has(caps, want) {
					t.Errorf("role %s missing expected capability %s", tt.role, want)
				}
			}
			for _, lack := range tt.wantLacks {
				if Has(caps, lack) {
					t.Errorf("role %s should NOT have capability %s", tt.role, lack)
				}
			}
		})
	}
}

func TestPresetCapabilities_UnknownRole(t *testing.T) {
	if caps := PresetCapabilities("superuser"); caps != nil {
		t.Errorf("unknown role should yield nil, got %v", caps)
	}
}

func TestValidCapabilityAndRole(t *testing.T) {
	if !ValidCapability(CapPublish) {
		t.Error("publish should be valid")
	}
	if ValidCapability("teleport") {
		t.Error("teleport should be invalid")
	}
	if !ValidRole(RoleAdmin) {
		t.Error("admin should be valid")
	}
	if ValidRole("wizard") {
		t.Error("wizard should be invalid")
	}
}

func TestNormalizeCapabilities(t *testing.T) {
	clean, unknown := NormalizeCapabilities([]string{"read", "upload", "read"})
	if unknown != "" {
		t.Fatalf("unexpected unknown: %s", unknown)
	}
	if len(clean) != 2 {
		t.Errorf("expected 2 deduped caps, got %v", clean)
	}

	if _, unknown := NormalizeCapabilities([]string{"read", "fly"}); unknown != "fly" {
		t.Errorf("expected unknown=fly, got %q", unknown)
	}
}

func TestCanGrant_NoPrivilegeEscalation(t *testing.T) {
	actor := []string{"read", "upload", "create"}

	// Granting a subset the actor holds is allowed.
	if ok, missing := CanGrant(actor, []string{"read", "upload"}); !ok {
		t.Errorf("granting held subset should succeed, missing=%s", missing)
	}
	// Granting a capability the actor lacks is rejected.
	ok, missing := CanGrant(actor, []string{"read", "delete"})
	if ok {
		t.Error("granting unheld capability should fail")
	}
	if missing != "delete" {
		t.Errorf("missing = %q, want delete", missing)
	}
}
