package content

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/yuin/goldmark/ast"
)

// A cold catalog read of the series card index took over 20s on 2026-10-27,
// and every page carrying a series card waited for it. Past this budget the
// page renders its entity links as plain links instead.
const entityLookupTimeout = 2500 * time.Millisecond

type Converter struct {
	CDNBase  string
	SiteBase string
	Images   func(hashes []string) map[string]imageclient.ImageMeta
	Users    func(ctx context.Context, ids []int) (map[int]userclient.User, error)
	Entities EntityResolver
}

func (c *Converter) Convert(ctx context.Context, sources []string) ([]ContentDocument, error) {
	type parsed struct {
		root  ast.Node
		src   []byte
		empty bool
	}
	items := make([]parsed, len(sources))
	var hashes []string
	seenH := map[string]struct{}{}
	var userIDs []int
	seenU := map[int]struct{}{}
	var entityRefs []EntityRef
	seenE := map[EntityRef]struct{}{}
	site := strings.TrimRight(c.SiteBase, "/")
	addHash := func(h string) {
		if h == "" {
			return
		}
		if _, ok := seenH[h]; ok {
			return
		}
		seenH[h] = struct{}{}
		hashes = append(hashes, h)
	}
	addUser := func(id int) {
		if id < 1 {
			return
		}
		if _, ok := seenU[id]; ok {
			return
		}
		seenU[id] = struct{}{}
		userIDs = append(userIDs, id)
	}
	addEntity := func(ref EntityRef) {
		if ref.Type == EntityUser {
			addUser(ref.ID)
			return
		}
		if _, ok := seenE[ref]; ok {
			return
		}
		seenE[ref] = struct{}{}
		entityRefs = append(entityRefs, ref)
	}
	for i, s := range sources {
		if strings.TrimSpace(s) == "" {
			items[i].empty = true
			continue
		}
		root, src := markdown.ParseStored(s)
		items[i].root = root
		items[i].src = src
		collectRefs(root, src, addHash, addUser)
		if c.Entities != nil {
			(&converter{site: site, src: src}).collectEntityRefs(root, addEntity)
		}
	}

	var images map[string]imageclient.ImageMeta
	if len(hashes) > 0 && c.Images != nil {
		images = c.Images(hashes)
	}
	var users map[int]userclient.User
	if len(userIDs) > 0 && c.Users != nil {
		var err error
		users, err = c.Users(ctx, userIDs)
		if err != nil {
			return nil, fmt.Errorf("content: lookup users: %w", err)
		}
	}

	entitiesOn := c.Entities != nil
	var targets map[EntityRef]EntityTarget
	if len(entityRefs) > 0 && entitiesOn {
		lookupCtx, cancel := context.WithTimeout(ctx, entityLookupTimeout)
		var err error
		targets, err = c.Entities(lookupCtx, entityRefs)
		cancel()
		if err != nil {
			slog.Warn("content: entity previews unavailable, rendering links", "error", err)
			entitiesOn = false
		}
	}

	out := make([]ContentDocument, len(sources))
	for i, it := range items {
		if it.empty {
			out[i] = NewDocument(nil)
			continue
		}
		cv := &converter{
			cdn:        c.CDNBase,
			site:       site,
			src:        it.src,
			images:     images,
			users:      users,
			entitiesOn: entitiesOn,
			targets:    targets,
		}
		out[i] = NewDocument(cv.blocks(it.root))
	}
	return out, nil
}

type converter struct {
	cdn        string
	site       string
	src        []byte
	images     map[string]imageclient.ImageMeta
	users      map[int]userclient.User
	untrusted  bool // raw HTML stays literal, and kungal-user/kungal-reply are not mentions
	entitiesOn bool
	targets    map[EntityRef]EntityTarget
}

func (c *converter) userRef(id int) repr.UserRef {
	if u, ok := c.users[id]; ok {
		return repr.NewUserRef(c.cdn, u)
	}
	return repr.DeletedUserRef(id)
}

func (c *converter) imageMeta(hash string) *imageclient.ImageMeta {
	if c.images == nil {
		return nil
	}
	m, ok := c.images[hash]
	if !ok {
		return nil
	}
	return &m
}

func collectRefs(root ast.Node, src []byte, addHash func(string), addUser func(int)) {
	if root == nil {
		return
	}
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Image:
			if h, _, ok := markdown.ParseContentImageRef(string(t.Destination)); ok {
				addHash(h)
			}
		case *ast.Link:
			if id, ok := parseUserMention(string(t.Destination)); ok {
				addUser(id)
			}
		case *ast.RawHTML:
			collectHTMLImageHashes(t.Segments.Value(src), addHash)
		case *ast.HTMLBlock:
			collectHTMLImageHashes([]byte(htmlBlockSource(t, src)), addHash)
		}
		return ast.WalkContinue, nil
	})
}
