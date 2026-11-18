package apiv1

import (
	"context"
	"errors"
	"sort"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/envconfig"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 env config: service is not configured")

type Service struct {
	store  *envconfig.Store
	people *userclient.Client
	cdn    string
}

func New(store *envconfig.Store, people *userclient.Client, cdn string) *Service {
	return &Service{store: store, people: people, cdn: cdn}
}

func (s *Service) admin(ctx context.Context) (*int, *problem.Problem) {
	if s == nil || s.store == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	if !user.Can(perm.SettingManage) {
		return nil, problem.New(problem.CodePermissionRequired, "This needs "+string(perm.SettingManage)+".")
	}
	id := user.ID
	return &id, nil
}

func kind(name string) string {
	switch {
	case config.IsBootstrap(name):
		return "bootstrap"
	case config.IsTopology(name):
		return "topology"
	case config.IsSecret(name):
		return "secret"
	default:
		return "plain"
	}
}

func valueOf(name, value string) *EnvValue {
	v := &EnvValue{IsSet: value != ""}
	if !config.IsSecret(name) {
		v.Plain = &value
	} else if value != "" {
		fp, hint := config.Fingerprint(value), config.Hint(value)
		v.Fingerprint, v.Hint = &fp, &hint
	}
	return v
}

func variableOf(running *config.Var, stored *envconfig.Stored) EnvVariable {
	name := ""
	if running != nil {
		name = running.Key
	} else {
		name = stored.Key
	}
	out := EnvVariable{
		Object: "env_variable", Variable: name, VariableKind: kind(name),
		Issues: []EnvVariableIssue{}, IsStorable: config.Storable(name),
	}
	if running != nil {
		from := string(running.Source)
		out.ResolvedFrom = &from
		out.Running = valueOf(name, running.Value)
		if !config.IsSecret(name) {
			def := running.Default
			out.DefaultValue = &def
		}
	}
	if stored != nil {
		out.Stored = valueOf(name, stored.Value)
		at := repr.Timestamp(stored.Updated)
		out.StoredAt = &at
	}
	out.Issues = issues(running, stored)
	return out
}

func issues(running *config.Var, stored *envconfig.Stored) []EnvVariableIssue {
	out := []EnvVariableIssue{}
	fromEnv := running != nil && running.Source == config.SourceEnv
	switch {
	case stored != nil && fromEnv && running.Value == stored.Value:
		out = append(out, "panel_redundant")
	case stored != nil && fromEnv:
		out = append(out, "panel_overrides_store")
	}
	switch {
	case stored != nil && !fromEnv && stored.Updated.After(config.LoadedAt()):
		out = append(out, "restart_pending")
	case stored == nil && running != nil && running.Source == config.SourceStore:
		out = append(out, "restart_pending")
	}
	return out
}

func (s *Service) snapshot(ctx context.Context) (EnvConfig, error) {
	stored, err := s.store.All(ctx)
	if err != nil {
		return EnvConfig{}, err
	}
	byName := map[string]*envconfig.Stored{}
	for i := range stored {
		byName[stored[i].Key] = &stored[i]
	}
	vars := config.Vars()
	out := EnvConfig{
		Object: "env_config", IsStoreEnabled: s.store.Enabled(), LoadedAt: repr.Timestamp(config.LoadedAt()),
		Variables: make([]EnvVariable, 0, len(vars)+len(stored)), UnreadVariables: config.UnreadEnv(),
	}
	for i := range vars {
		out.Variables = append(out.Variables, variableOf(&vars[i], byName[vars[i].Key]))
		delete(byName, vars[i].Key)
	}
	for _, st := range byName {
		out.Variables = append(out.Variables, variableOf(nil, st))
	}
	sort.Slice(out.Variables, func(i, j int) bool { return out.Variables[i].Variable < out.Variables[j].Variable })
	if out.UnreadVariables == nil {
		out.UnreadVariables = []string{}
	}
	return out, nil
}

func (s *Service) getEnvConfig(ctx context.Context, _ *getEnvConfigInput) (*envConfigOutput, error) {
	if _, p := s.admin(ctx); p != nil {
		return nil, p
	}
	body, err := s.snapshot(ctx)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &envConfigOutput{Body: body}, nil
}

func (s *Service) storable(name string) *problem.Problem {
	if config.Storable(name) {
		return nil
	}
	return problem.New(problem.CodeValidationFailed, "This variable stays in the compose file.",
		problem.AtParameter("variable", problem.ReasonNotAllowedValue, "bootstrap and topology variables cannot be stored", nil))
}

func (s *Service) storeOff() *problem.Problem {
	return problem.Unavailable(envconfig.ErrStoreOff)
}

func (s *Service) find(ctx context.Context, name string) (EnvVariable, error) {
	body, err := s.snapshot(ctx)
	if err != nil {
		return EnvVariable{}, err
	}
	for _, v := range body.Variables {
		if v.Variable == name {
			return v, nil
		}
	}
	return variableOf(&config.Var{Key: name, Source: config.SourceDefault}, nil), nil
}

func (s *Service) putEnvVariable(ctx context.Context, in *putEnvVariableInput) (*envVariableOutput, error) {
	uid, p := s.admin(ctx)
	if p != nil {
		return nil, p
	}
	if p := s.storable(in.Variable); p != nil {
		return nil, p
	}
	if !s.store.Enabled() {
		return nil, s.storeOff()
	}
	note := ""
	if in.Body.ChangeNote != nil {
		note = *in.Body.ChangeNote
	}
	if _, err := s.store.Set(ctx, in.Variable, in.Body.Value, note, envconfig.ActionSet, uid); err != nil {
		return nil, problem.Internal(err)
	}
	v, err := s.find(ctx, in.Variable)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &envVariableOutput{Body: v}, nil
}

func (s *Service) deleteEnvVariable(ctx context.Context, in *deleteEnvVariableInput) (*noContentOutput, error) {
	uid, p := s.admin(ctx)
	if p != nil {
		return nil, p
	}
	if !s.store.Enabled() {
		return nil, s.storeOff()
	}
	deleted, err := s.store.Delete(ctx, in.Variable, in.Note, uid)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !deleted {
		return nil, problem.New(problem.CodeNotFound, "Nothing is stored for this variable.")
	}
	return &noContentOutput{}, nil
}

type listEnvChangesInput struct {
	collect.PageNumber
}

type listEnvChangesOutput struct {
	Body repr.PageList[EnvChange]
}

func (s *Service) listEnvChanges(ctx context.Context, in *listEnvChangesInput) (*listEnvChangesOutput, error) {
	if _, p := s.admin(ctx); p != nil {
		return nil, p
	}
	if p := in.CheckDepth(); p != nil {
		return nil, p
	}
	rows, count, err := s.store.Changes(ctx, in.Offset(), in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		if r.UserID != nil {
			ids = append(ids, *r.UserID)
		}
	}
	users := map[int]userclient.User{}
	if len(ids) > 0 && s.people != nil {
		if users, err = s.people.Users(ctx, ids); err != nil {
			return nil, problem.Unavailable(err)
		}
	}
	items := make([]EnvChange, 0, len(rows))
	for _, r := range rows {
		change := EnvChange{
			Object: "env_change", ID: repr.ID(int(r.ID)), Variable: r.Key, ChangeKind: r.Action,
			ShownBefore: r.OldShown, ShownAfter: r.NewShown, ChangeNote: r.Note, CreatedAt: repr.Timestamp(r.Created),
		}
		if r.UserID != nil {
			ref := repr.DeletedUserRef(*r.UserID)
			if u, ok := users[*r.UserID]; ok {
				ref = repr.NewUserRef(s.cdn, u)
			}
			change.ChangedBy = &ref
		}
		items = append(items, change)
	}
	return &listEnvChangesOutput{Body: repr.NewPageList(items, int(count), "eq")}, nil
}
