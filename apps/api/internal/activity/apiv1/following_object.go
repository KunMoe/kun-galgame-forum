package apiv1

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/pkg/communityclient"
)

type objectRef struct {
	kind string
	id   int
}

var objectPaths = map[string]string{
	"work":    "/galgame/",
	"company": "/galgame/official/",
	"folder":  "/collection/",
	"user":    "/user/",
	"topic":   "/topic/",
}

var objectEntities = map[string]content.EntityType{
	"company": content.EntityCompany,
	"user":    content.EntityUser,
	"topic":   content.EntityTopic,
}

func parseObjectRef(raw *string) (objectRef, bool) {
	if raw == nil {
		return objectRef{}, false
	}
	kind, id, ok := strings.Cut(*raw, ":")
	if !ok || objectPaths[kind] == "" {
		return objectRef{}, false
	}
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		return objectRef{}, false
	}
	return objectRef{kind: kind, id: n}, true
}

func (r objectRef) path() string   { return objectPaths[r.kind] + strconv.Itoa(r.id) }
func (r objectRef) String() string { return r.kind + ":" + strconv.Itoa(r.id) }

// The target only names the object, so a failed lookup leaves the items on
// their stored titles instead of failing the feed.
func (s *Service) objectTargets(ctx context.Context, items []communityclient.ActivityItemView) map[objectRef]content.EntityTarget {
	var refs []objectRef
	var entityRefs []content.EntityRef
	seen := map[objectRef]bool{}
	for _, it := range items {
		ref, ok := parseObjectRef(it.ObjectRef)
		typ, named := objectEntities[ref.kind]
		if !ok || !named || seen[ref] {
			continue
		}
		seen[ref] = true
		refs = append(refs, ref)
		entityRefs = append(entityRefs, content.EntityRef{Type: typ, ID: ref.id})
	}
	if len(refs) == 0 || s.convert == nil {
		return nil
	}
	targets, err := s.convert.Targets(ctx, entityRefs)
	if err != nil {
		slog.Warn("activity: object targets unavailable", "err", err)
		return nil
	}
	out := make(map[objectRef]content.EntityTarget, len(refs))
	for i, ref := range refs {
		out[ref] = targets[i]
	}
	return out
}

func (x itemExtras) target(it communityclient.ActivityItemView) *content.EntityTarget {
	ref, ok := parseObjectRef(it.ObjectRef)
	if !ok {
		return nil
	}
	if t, ok := x.targets[ref]; ok {
		return &t
	}
	return nil
}

func itemWorkID(it communityclient.ActivityItemView) int {
	if it.WorkID != nil && *it.WorkID > 0 {
		return int(*it.WorkID)
	}
	if ref, ok := parseObjectRef(it.ObjectRef); ok && ref.kind == "work" {
		return ref.id
	}
	return 0
}
