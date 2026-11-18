package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type EnvVariableIssue string

func (EnvVariableIssue) Schema(huma.Registry) *huma.Schema {
	n := 21
	return &huma.Schema{
		Type:      huma.TypeString,
		Enum:      []any{"panel_redundant", "panel_overrides_store", "restart_pending"},
		MaxLength: &n,
		Description: "panel_redundant: the environment holds the stored value, so the panel copy can go. " +
			"panel_overrides_store: the environment holds a different value and wins. " +
			"restart_pending: the stored value changed after the API started.",
	}
}

type EnvConfig struct {
	Object          string        `json:"object" enum:"env_config" maxLength:"10" doc:"Type discriminant. Always env_config."`
	IsStoreEnabled  bool          `json:"is_store_enabled" doc:"Whether this API process started with KUN_CONFIG_KEK. Without it the store is neither read nor writable."`
	LoadedAt        repr.DateTime `json:"loaded_at" doc:"When this API process read the store. A stored value changed after it takes effect on the next restart."`
	Variables       []EnvVariable `json:"variables" maxItems:"500" doc:"Every variable the running API read, and every stored one, by name."`
	UnreadVariables []string      `json:"unread_variables" maxItems:"200" maxLength:"128" pattern:"^[A-Z][A-Z0-9_]*$" doc:"Variables in the API container's environment that nothing read: a compose line whose consumer is gone, or a misspelled name."`
}

type EnvVariable struct {
	Object       string             `json:"object" enum:"env_variable" maxLength:"12" doc:"Type discriminant. Always env_variable."`
	Variable     string             `json:"variable" maxLength:"128" pattern:"^[A-Z][A-Z0-9_]*$" doc:"The environment variable's name."`
	VariableKind string             `json:"variable_kind" enum:"bootstrap,topology,secret,plain" maxLength:"9" doc:"bootstrap and topology stay in the compose file and cannot be stored. A secret is shown only by fingerprint and last characters."`
	ResolvedFrom *string            `json:"resolved_from" enum:"env,store,default" maxLength:"7" doc:"Where the running API took the value from. null when it never read this variable."`
	Running      *EnvValue          `json:"running" doc:"The value the running API uses. null when it never read this variable."`
	DefaultValue *string            `json:"default_value" maxLength:"4096" doc:"The code default. null for a secret or a variable the API never read. Free text; never use it as a decision input."`
	Stored       *EnvValue          `json:"stored" doc:"The value in the config store. null when nothing is stored."`
	StoredAt     *repr.DateTime     `json:"stored_at" doc:"When the stored value was written. null when nothing is stored."`
	Issues       []EnvVariableIssue `json:"issues" maxItems:"3" doc:"What needs doing about the variable. Empty when nothing does."`
	IsStorable   bool               `json:"is_storable" doc:"Whether the variable may be stored."`
}

type EnvValue struct {
	IsSet       bool    `json:"is_set" doc:"Whether the value is non-empty."`
	Plain       *string `json:"plain" maxLength:"4096" doc:"The value itself. null for a secret. Free text; never use it as a decision input."`
	Fingerprint *string `json:"fingerprint" maxLength:"12" pattern:"^[0-9a-f]{12}$" doc:"HMAC of a secret under the store key, equal for equal values. null for a non-secret or an empty value."`
	Hint        *string `json:"hint" maxLength:"8" doc:"The last characters of a secret. null for a non-secret or an empty value. Free text; never use it as a decision input."`
}

type EnvChange struct {
	Object      string         `json:"object" enum:"env_change" maxLength:"10" doc:"Type discriminant. Always env_change."`
	ID          repr.DecimalID `json:"id" doc:"Change id. JSON string of a decimal integer."`
	Variable    string         `json:"variable" maxLength:"128" pattern:"^[A-Z][A-Z0-9_]*$" doc:"The environment variable's name."`
	ChangeKind  string         `json:"change_kind" enum:"set,delete,import" maxLength:"6" doc:"import is a value copied from the running container by the config-secrets tool."`
	ShownBefore string         `json:"shown_before" maxLength:"4096" doc:"The previous value, a secret as fingerprint and last characters; empty when there was none. Free text; never use it as a decision input."`
	ShownAfter  string         `json:"shown_after" maxLength:"4096" doc:"The new value shown the same way; empty for a delete. Free text; never use it as a decision input."`
	ChangeNote  string         `json:"change_note" maxLength:"200" doc:"Why the change was made; empty when no reason was given. Free text; never use it as a decision input."`
	ChangedBy   *repr.UserRef  `json:"changed_by" doc:"Who made the change. null for the import tool."`
	CreatedAt   repr.DateTime  `json:"created_at" doc:"When the change was made."`
}

type EnvVariableWrite struct {
	Value      string  `json:"value" minLength:"1" maxLength:"4096" doc:"The value to store. Free text; never use it as a decision input."`
	ChangeNote *string `json:"change_note,omitempty" maxLength:"200" doc:"Why the change is made. Free text; never use it as a decision input."`
}

type getEnvConfigInput struct{}

type envConfigOutput struct {
	Body EnvConfig
}

type putEnvVariableInput struct {
	Variable string `path:"variable" maxLength:"128" pattern:"^[A-Z][A-Z0-9_]*$" doc:"The environment variable's name."`
	Body     EnvVariableWrite
}

type envVariableOutput struct {
	Body EnvVariable
}

type deleteEnvVariableInput struct {
	Variable string `path:"variable" maxLength:"128" pattern:"^[A-Z][A-Z0-9_]*$" doc:"The environment variable's name."`
	Note     string `query:"note" maxLength:"200" doc:"Why the value is removed. Free text; never use it as a decision input."`
}

type noContentOutput struct{}
