package content

import (
	"context"
	"fmt"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/yuin/goldmark/ast"
)

type Converter struct {
	CDNBase  string
	SiteBase string
	Images   func(hashes []string) map[string]imageclient.ImageMeta
	Users    func(ctx context.Context, ids []int) (map[int]userclient.User, error)
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
	for i, s := range sources {
		if strings.TrimSpace(s) == "" {
			items[i].empty = true
			continue
		}
		root, src := markdown.ParseStored(s)
		items[i].root = root
		items[i].src = src
		collectRefs(root, src, addHash, addUser)
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

	out := make([]ContentDocument, len(sources))
	for i, it := range items {
		if it.empty {
			out[i] = NewDocument(nil)
			continue
		}
		cv := &converter{
			cdn:    c.CDNBase,
			site:   strings.TrimRight(c.SiteBase, "/"),
			src:    it.src,
			images: images,
			users:  users,
		}
		out[i] = NewDocument(cv.blocks(it.root))
	}
	return out, nil
}

type converter struct {
	cdn       string
	site      string
	src       []byte
	images    map[string]imageclient.ImageMeta
	users     map[int]userclient.User
	untrusted bool // raw HTML stays literal, and kungal-user/kungal-reply are not mentions
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
