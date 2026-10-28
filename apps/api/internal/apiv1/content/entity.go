package content

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
	"github.com/yuin/goldmark/ast"
)

type EntityType string

const (
	EntityTopic           EntityType = "topic"
	EntityWork            EntityType = "work"
	EntityGalgameResource EntityType = "galgame_resource"
	EntityUser            EntityType = "user"
	EntityCharacter       EntityType = "character"
	EntityCompany         EntityType = "company"
	EntityTag             EntityType = "tag"
	EntitySeries          EntityType = "series"
	EntityCreditName      EntityType = "credit_name"
	EntityRating          EntityType = "rating"
)

type EntityRef struct {
	Type EntityType
	ID   int
}

type EntityResolver func(ctx context.Context, refs []EntityRef) (map[EntityRef]EntityTarget, error)

var entityPathRe = regexp.MustCompile(`^/(topic|galgame/(?:resource|character|official|tag|series|staff)|galgame-rating|galgame|user)/([1-9][0-9]{0,18})/?$`)

var entityPaths = map[EntityType]string{
	EntityTopic:           "topic",
	EntityWork:            "galgame",
	EntityGalgameResource: "galgame/resource",
	EntityUser:            "user",
	EntityCharacter:       "galgame/character",
	EntityCompany:         "galgame/official",
	EntityTag:             "galgame/tag",
	EntitySeries:          "galgame/series",
	EntityCreditName:      "galgame/staff",
	EntityRating:          "galgame-rating",
}

var entityPathTypes = func() map[string]EntityType {
	out := make(map[string]EntityType, len(entityPaths))
	for t, p := range entityPaths {
		out[p] = t
	}
	return out
}()

func (c *converter) entityFromURL(dest string) (EntityRef, bool) {
	u, err := url.Parse(dest)
	if err != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.User != nil {
		return EntityRef{}, false
	}
	if u.Scheme != "" || u.Host != "" {
		scheme := strings.ToLower(u.Scheme)
		if (scheme != "https" && scheme != "http") || !c.isSiteHost(u.Host) {
			return EntityRef{}, false
		}
	} else if !strings.HasPrefix(dest, "/") || strings.HasPrefix(dest, "//") {
		return EntityRef{}, false
	}
	m := entityPathRe.FindStringSubmatch(u.EscapedPath())
	if m == nil {
		return EntityRef{}, false
	}
	id, ok := parsePositiveInt(m[2])
	if !ok {
		return EntityRef{}, false
	}
	return EntityRef{Type: entityPathTypes[m[1]], ID: id}, true
}

func (c *converter) isSiteHost(host string) bool {
	site, err := url.Parse(c.site)
	if err != nil || site.Host == "" {
		return false
	}
	host = strings.ToLower(host)
	apex := strings.TrimPrefix(strings.ToLower(site.Host), "www.")
	return host == apex || host == "www."+apex
}

func (c *converter) entityURL(ref EntityRef) string {
	return c.site + "/" + entityPaths[ref.Type] + "/" + strconv.Itoa(ref.ID)
}

type TopicPreview struct {
	Object      string         `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID          repr.DecimalID `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title       string         `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	CoverImages []repr.Image   `json:"cover_images" maxItems:"9" doc:"Cover images in stored token order, stickers left out. Empty array if none."`
	Author      repr.UserRef   `json:"author" doc:"Topic author."`
	ViewCount   int            `json:"view_count" minimum:"0" doc:"Lifetime view count."`
	LikeCount   int            `json:"like_count" minimum:"0" doc:"Like count."`
	ReplyCount  int            `json:"reply_count" minimum:"0" doc:"Reply count."`
	IsNSFW      bool           `json:"is_nsfw" doc:"Whether the topic is marked NSFW. Mask its covers unless the reader asked for NSFW content."`
	CreatedAt   repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type GalgameResourcePreview struct {
	Object            string                      `json:"object" enum:"galgame_resource" maxLength:"16" doc:"Type discriminant. Always galgame_resource."`
	ID                repr.DecimalID              `json:"id" doc:"Resource id."`
	Work              *repr.WorkRef               `json:"work" doc:"The work this resource belongs to. Never null here; nullable to match WorkRef elsewhere."`
	Author            repr.UserRef                `json:"author" doc:"Uploader of the resource."`
	ResourceType      string                      `json:"resource_type" enum:"game,patch,collection,crack_fix,mod,tool,walkthrough,ost,voice,cg,wallpaper,artbook,video,other" maxLength:"11" doc:"Kind of download."`
	ResourceLanguages []workrepr.ResourceLanguage `json:"resource_languages" doc:"Languages of the resource. Never null."`
	ResourcePlatforms []workrepr.ResourcePlatform `json:"resource_platforms" doc:"Platforms of the resource. Empty array when only runtimes apply. Never null."`
	Title             string                      `json:"title" maxLength:"200" doc:"Optional title. Empty string when none. Free text; never use it as a decision input."`
	Size              string                      `json:"size" maxLength:"64" doc:"Size as the uploader wrote it, such as 1.5 GB. Free text; never use it as a decision input."`
	State             string                      `json:"state" enum:"valid,expired" maxLength:"7" doc:"Whether the links are currently treated as working."`
	DownloadCount     int                         `json:"download_count" minimum:"0" doc:"Times download secrets were issued."`
	LikeCount         int                         `json:"like_count" minimum:"0" doc:"Number of likes."`
	CreatedAt         repr.DateTime               `json:"created_at" doc:"Creation time."`
}

type UnavailableEntity struct {
	Object     string         `json:"object" enum:"unavailable_entity" maxLength:"18" doc:"Type discriminant. Always unavailable_entity."`
	EntityType string         `json:"entity_type" enum:"topic,work,galgame_resource,character,company,tag,series,credit_name,rating" maxLength:"16" doc:"What the link pointed at."`
	EntityID   repr.DecimalID `json:"entity_id" doc:"Id of the entity the link pointed at. Its page may still open for a reader allowed to see it."`
}

func NewUnavailableEntity(ref EntityRef) UnavailableEntity {
	return UnavailableEntity{Object: "unavailable_entity", EntityType: string(ref.Type), EntityID: repr.ID(ref.ID)}
}

type EntityTarget struct{ v any }

func TopicTarget(p TopicPreview) EntityTarget { return EntityTarget{p} }

func GalgameResourceTarget(p GalgameResourcePreview) EntityTarget { return EntityTarget{p} }

func UserTarget(u repr.UserRef) EntityTarget { return EntityTarget{u} }

func UnavailableTarget(ref EntityRef) EntityTarget { return EntityTarget{NewUnavailableEntity(ref)} }

func (t EntityTarget) MarshalJSON() ([]byte, error) { return json.Marshal(t.v) }

var entityTargetMembers = []reflect.Type{
	reflect.TypeFor[TopicPreview](),
	reflect.TypeFor[WorkPreview](),
	reflect.TypeFor[GalgameResourcePreview](),
	reflect.TypeFor[repr.UserRef](),
	reflect.TypeFor[CharacterPreview](),
	reflect.TypeFor[CompanyPreview](),
	reflect.TypeFor[TagPreview](),
	reflect.TypeFor[SeriesPreview](),
	reflect.TypeFor[CreditNamePreview](),
	reflect.TypeFor[RatingPreview](),
	reflect.TypeFor[UnavailableEntity](),
}

func (EntityTarget) Schema(r huma.Registry) *huma.Schema {
	return unionRef(r, "EntityTarget",
		"What an entity link points at. Switch on object; for an unknown type, render the node's children.", entityTargetMembers)
}

type EntityCardNode struct {
	Object   string       `json:"object" enum:"entity_card" maxLength:"11" doc:"Type discriminant. Always entity_card."`
	Target   EntityTarget `json:"target" doc:"The entity the card shows, as of this read."`
	Children Blocks       `json:"children" doc:"One paragraph holding the plain link the author wrote. Render target instead; the link is for a client that does not know entity_card."`
}

type EntityLinkNode struct {
	Object   string       `json:"object" enum:"entity_link" maxLength:"11" doc:"Type discriminant. Always entity_link."`
	Target   EntityTarget `json:"target" doc:"The entity the link points at, as of this read."`
	Children Inlines      `json:"children" doc:"The plain link the author wrote. Render target instead; the link is for a client that does not know entity_link."`
}

func NewEntityCard(target EntityTarget, link LinkNode) EntityCardNode {
	return EntityCardNode{Object: "entity_card", Target: target, Children: Blocks{NewParagraph(Inlines{link})}}
}

func NewEntityLink(target EntityTarget, link LinkNode) EntityLinkNode {
	return EntityLinkNode{Object: "entity_link", Target: target, Children: Inlines{link}}
}

func (c *converter) bareEntity(n ast.Node) (EntityRef, string, bool) {
	var dest, text string
	switch t := n.(type) {
	case *ast.AutoLink:
		if t.AutoLinkType != ast.AutoLinkURL {
			return EntityRef{}, "", false
		}
		dest = string(t.URL(c.src))
		text = string(t.Label(c.src))
	case *ast.Link:
		dest = string(t.Destination)
		text = c.plainText(t)
	default:
		return EntityRef{}, "", false
	}
	if text != dest {
		return EntityRef{}, "", false
	}
	ref, ok := c.entityFromURL(dest)
	return ref, text, ok
}

func (c *converter) collectEntityRefs(root ast.Node, add func(EntityRef)) {
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if ref, _, ok := c.bareEntity(n); ok {
				add(ref)
			}
		}
		return ast.WalkContinue, nil
	})
}

func (c *converter) entityLink(n ast.Node) (EntityLinkNode, bool) {
	if !c.entitiesOn || c.untrusted {
		return EntityLinkNode{}, false
	}
	ref, text, ok := c.bareEntity(n)
	if !ok {
		return EntityLinkNode{}, false
	}
	link := NewLink(c.entityURL(ref), Inlines{NewText(text)})
	return NewEntityLink(c.entityTarget(ref), link), true
}

func (c *converter) entityTarget(ref EntityRef) EntityTarget {
	if ref.Type == EntityUser {
		return UserTarget(c.userRef(ref.ID))
	}
	if t, ok := c.targets[ref]; ok {
		return t
	}
	return UnavailableTarget(ref)
}

func entityCard(in Inlines) (EntityCardNode, bool) {
	var el EntityLinkNode
	found := false
	for _, n := range in {
		if t, ok := n.(TextNode); ok && strings.TrimSpace(t.Value) == "" {
			continue
		}
		e, ok := n.(EntityLinkNode)
		if !ok || found {
			return EntityCardNode{}, false
		}
		el, found = e, true
	}
	if !found {
		return EntityCardNode{}, false
	}
	link, ok := el.Children[0].(LinkNode)
	if !ok {
		return EntityCardNode{}, false
	}
	return NewEntityCard(el.Target, link), true
}

func (c *Converter) Targets(ctx context.Context, refs []EntityRef) ([]EntityTarget, error) {
	var userIDs []int
	var others []EntityRef
	for _, r := range refs {
		if r.Type == EntityUser {
			userIDs = append(userIDs, r.ID)
		} else {
			others = append(others, r)
		}
	}
	cv := &converter{cdn: c.CDNBase}
	if len(userIDs) > 0 && c.Users != nil {
		users, err := c.Users(ctx, userIDs)
		if err != nil {
			return nil, err
		}
		cv.users = users
	}
	if len(others) > 0 && c.Entities != nil {
		targets, err := c.Entities(ctx, others)
		if err != nil {
			return nil, err
		}
		cv.targets = targets
	}
	out := make([]EntityTarget, len(refs))
	for i, r := range refs {
		out[i] = cv.entityTarget(r)
	}
	return out, nil
}
