package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type Permission string

func (Permission) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("permission", 64)
	s.Pattern = `^[a-z][a-z_]*(\.[a-z][a-z_]*)+$`
	s.Description = "Permission key: dot-separated snake_case segments. The catalog grows as the forum adds surfaces; treat a key you do not know as one you cannot act on."
	return s
}

type RankedRole string

func (RankedRole) Schema(huma.Registry) *huma.Schema {
	n := 9
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        []any{"creator", "moderator", "admin", "ren"},
		MaxLength:   &n,
		Description: "A role that carries permissions, lowest rank first. ren always holds every permission.",
	}
}

type PermissionOverride struct {
	Permission Permission `json:"permission" doc:"The permission the override is about."`
	Effect     string     `json:"effect" enum:"grant,revoke" maxLength:"6" doc:"grant adds a permission the baseline lacks; revoke removes one the baseline has."`
}

type PermissionSet struct {
	Object      string       `json:"object" enum:"permission_set" maxLength:"14" doc:"Type discriminant. Always permission_set."`
	Permissions []Permission `json:"permissions" maxItems:"256" doc:"Every permission the caller holds right now, in catalog order. Empty for a Bearer request: staff powers never reach the App channel."`
}

type RolePermissions struct {
	Object    string                `json:"object" enum:"role_permissions" maxLength:"16" doc:"Type discriminant. Always role_permissions."`
	Role      RankedRole            `json:"role" doc:"The role this layer belongs to."`
	Baseline  []Permission          `json:"baseline" maxItems:"256" doc:"Permissions the role holds by code, before overrides, in catalog order."`
	Overrides []PermissionOverride  `json:"overrides" maxItems:"256" doc:"The role's stored deviations from its baseline, in catalog order. Always empty for ren."`
	Effective []Permission          `json:"effective" maxItems:"256" doc:"Permissions the role holds after its overrides, in catalog order."`
	IsLocked  bool                  `json:"is_locked" doc:"Whether the role cannot be changed at all. True only for ren, which always holds every permission."`
	Viewer    RolePermissionsViewer `json:"viewer" doc:"The caller's own standing on this role."`
}

type RolePermissionsViewer struct {
	CanEdit bool `json:"can_edit" doc:"Whether the caller may replace this role's overrides: the role is not locked and ranks below the caller's highest role. Each key is still subject to the caller holding it."`
}

type RolePermissionMatrix struct {
	Object          string            `json:"object" enum:"role_permission_matrix" maxLength:"22" doc:"Type discriminant. Always role_permission_matrix."`
	Catalog         []Permission      `json:"catalog" maxItems:"256" doc:"Every permission key the forum knows, in catalog order."`
	RolePermissions []RolePermissions `json:"role_permissions" maxItems:"4" doc:"One layer per role: creator, moderator, admin, ren, in that order."`
}

type RoleOverrides struct {
	Role      RankedRole           `json:"role" doc:"The role whose overrides are replaced. ren is refused."`
	Overrides []PermissionOverride `json:"overrides" maxItems:"256" doc:"The role's new override set, replacing the stored one. An empty array resets the role to its baseline."`
}

type RolePermissionMatrixPatch struct {
	Changes []RoleOverrides `json:"changes" minItems:"1" maxItems:"4" doc:"The roles to replace. Roles not listed keep their overrides. All changes are judged together against the resulting state and written in one transaction, or none is."`
}

type UserPermissions struct {
	Object    string                `json:"object" enum:"user_permissions" maxLength:"16" doc:"Type discriminant. Always user_permissions."`
	ID        repr.DecimalID        `json:"id" doc:"The user's id. JSON string of a decimal integer."`
	Roles     []RankedRole          `json:"roles" maxItems:"4" doc:"The user's roles that carry permissions, from the account service's current record, lowest rank first."`
	Baseline  []Permission          `json:"baseline" maxItems:"256" doc:"Permissions the user's roles grant, in catalog order."`
	Overrides []PermissionOverride  `json:"overrides" maxItems:"256" doc:"The user's personal deviations from that baseline, in catalog order."`
	Effective []Permission          `json:"effective" maxItems:"256" doc:"Permissions the user holds after personal overrides, in catalog order. Every permission when the user holds ren."`
	IsLocked  bool                  `json:"is_locked" doc:"Whether the user's permissions cannot be changed: the user holds ren."`
	Viewer    UserPermissionsViewer `json:"viewer" doc:"The caller's own standing on this user."`
}

type UserPermissionsViewer struct {
	CanEdit bool `json:"can_edit" doc:"Whether the caller may replace this user's overrides: the user is not locked and ranks below the caller. Each key is still subject to the caller holding it."`
}

type UserPermissionsReplace struct {
	Overrides []PermissionOverride `json:"overrides" maxItems:"256" doc:"The user's new override set, replacing the stored one. An empty array resets the user to their roles."`
}

type PermissionChange struct {
	Object     string               `json:"object" enum:"permission_change" maxLength:"17" doc:"Type discriminant. Always permission_change."`
	ID         repr.DecimalID       `json:"id" doc:"Change id. JSON string of a decimal integer."`
	Actor      repr.UserRef         `json:"actor" doc:"Who made the change."`
	TargetRole *string              `json:"target_role" enum:"creator,moderator,admin" maxLength:"9" doc:"The role whose overrides changed. null when the change was to a user."`
	TargetUser *repr.UserRef        `json:"target_user" doc:"The user whose overrides changed. null when the change was to a role."`
	Before     []PermissionOverride `json:"before" maxItems:"256" doc:"The target's overrides before the change."`
	After      []PermissionOverride `json:"after" maxItems:"256" doc:"The target's overrides after the change. Empty when the change reset the target to its baseline."`
	CreatedAt  repr.DateTime        `json:"created_at" doc:"When the change was made."`
}
