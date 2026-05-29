// Package workspace implements multi-tenant workspaces, capability-based
// membership, and the authorization model described in docs/MASTER_PLAN.md 5.1.
//
// Authorization is capability-based, not a fixed role hierarchy: a member holds
// an explicit set of capabilities. Roles are named presets that expand to a
// capability set on assignment; an admin may then grant or revoke individual
// capabilities, so the stored permission set is the source of truth at authz
// time and may diverge from any preset.
package workspace

// Capability is a single permission a member may hold within a workspace.
type Capability string

// The capability registry. Extend as features land; keep in sync with
// docs/MASTER_PLAN.md 5.1 and the SECURITY checklist.
const (
	// CapRead allows viewing workspace resources.
	CapRead Capability = "read"
	// CapCreate allows creating posts/drafts.
	CapCreate Capability = "create"
	// CapUpdate allows editing existing posts/drafts.
	CapUpdate Capability = "update"
	// CapDelete allows deleting posts/drafts/media.
	CapDelete Capability = "delete"
	// CapUpload allows uploading media assets.
	CapUpload Capability = "upload"
	// CapPublish allows scheduling and publishing posts.
	CapPublish Capability = "publish"
	// CapManageChannels allows connecting/disconnecting social channels.
	CapManageChannels Capability = "manage_channels"
	// CapManageMembers allows inviting/removing members and changing their capabilities.
	CapManageMembers Capability = "manage_members"
	// CapManageWorkspace allows renaming/deleting the workspace (owner-level).
	CapManageWorkspace Capability = "manage_workspace"
)

// allCapabilities is the full registry, used to validate input and build the
// owner preset.
var allCapabilities = []Capability{
	CapRead, CapCreate, CapUpdate, CapDelete, CapUpload,
	CapPublish, CapManageChannels, CapManageMembers, CapManageWorkspace,
}

// Role is a named preset over capabilities. Roles are a convenience default;
// the stored capability set is authoritative.
type Role string

// The preset roles.
const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// rolePresets maps each role to the capabilities it grants on assignment.
var rolePresets = map[Role][]Capability{
	RoleViewer: {CapRead},
	RoleEditor: {CapRead, CapCreate, CapUpdate, CapUpload, CapPublish},
	RoleAdmin: {
		CapRead, CapCreate, CapUpdate, CapUpload, CapPublish,
		CapDelete, CapManageChannels, CapManageMembers,
	},
	RoleOwner: allCapabilities,
}

// ValidCapability reports whether c is a known capability.
func ValidCapability(c Capability) bool {
	for _, known := range allCapabilities {
		if c == known {
			return true
		}
	}
	return false
}

// ValidRole reports whether r is a known preset role.
func ValidRole(r Role) bool {
	_, ok := rolePresets[r]
	return ok
}

// PresetCapabilities returns the capability strings a role grants. The result is
// a fresh slice the caller may store; an unknown role yields nil.
func PresetCapabilities(r Role) []string {
	preset, ok := rolePresets[r]
	if !ok {
		return nil
	}
	out := make([]string, len(preset))
	for i, c := range preset {
		out[i] = string(c)
	}
	return out
}

// Has reports whether the held capability set includes c.
func Has(held []string, c Capability) bool {
	for _, h := range held {
		if Capability(h) == c {
			return true
		}
	}
	return false
}

// NormalizeCapabilities validates and de-duplicates a requested capability set,
// returning the cleaned list. It reports the first unknown capability, if any,
// so callers can reject invalid input.
func NormalizeCapabilities(requested []string) (clean []string, unknown string) {
	seen := make(map[string]struct{}, len(requested))
	for _, r := range requested {
		if !ValidCapability(Capability(r)) {
			return nil, r
		}
		if _, dup := seen[r]; dup {
			continue
		}
		seen[r] = struct{}{}
		clean = append(clean, r)
	}
	return clean, ""
}

// CanGrant enforces the no-privilege-escalation invariant: an actor may grant
// only capabilities it itself holds. It returns the first capability the actor
// lacks, if any.
func CanGrant(actorHeld, requested []string) (ok bool, missing string) {
	for _, r := range requested {
		if !Has(actorHeld, Capability(r)) {
			return false, r
		}
	}
	return true, ""
}
