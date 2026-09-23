package apiv1

import (
	"strconv"
	"strings"

	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/perm"

	"github.com/danielgtaylor/huma/v2"
)

type SubjectType string

var subjectTypes = []any{"galgame", "galgame_rating", "galgame_resource", "galgame_quiz", "toolset", "website"}

func (SubjectType) Schema(huma.Registry) *huma.Schema {
	n := 16
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        subjectTypes,
		MaxLength:   &n,
		Description: "Kind of page whose comment wall this is.",
	}
}

type subjectSpec struct {
	typ        SubjectType
	anchorKind int32
	// site_resource packs the page kind into the anchor id ("rating:12"); a
	// site_game anchor id is the bare galgame id.
	anchorPrefix string
	editPerm     perm.Permission
	deletePerm   perm.Permission
	maxLength    int
	feedType     string
	// The table whose comment_count this wall maintains. galgame_rating has a
	// column nobody reads and the legacy wall never maintained it.
	counterTable string
	// Owners of these walls are notified of new top-level comments and may
	// delete any comment on their wall.
	hasOwner   bool
	linkPrefix string
	legacyKey  string
}

var subjects = map[SubjectType]subjectSpec{
	"galgame": {
		typ: "galgame", anchorKind: communityclient.AnchorSiteGame,
		editPerm: perm.CommentGalgameEdit, deletePerm: perm.CommentGalgameDelete,
		maxLength: 5000, feedType: "GALGAME_COMMENT_CREATION", linkPrefix: "/galgame/",
	},
	"galgame_rating": {
		typ: "galgame_rating", anchorKind: communityclient.AnchorSiteResource, anchorPrefix: "rating:",
		editPerm: perm.CommentRatingEdit, deletePerm: perm.CommentRatingDelete,
		maxLength: 1314, feedType: "GALGAME_RATING_COMMENT_CREATION",
		hasOwner: true, linkPrefix: "/galgame-rating/", legacyKey: "rating",
	},
	"galgame_resource": {
		typ: "galgame_resource", anchorKind: communityclient.AnchorSiteResource, anchorPrefix: "resource:",
		editPerm: perm.CommentResourceEdit, deletePerm: perm.CommentResourceDelete,
		maxLength: 1007, feedType: "GALGAME_RESOURCE_COMMENT_CREATION", counterTable: "galgame_resource",
		hasOwner: true, linkPrefix: "/galgame/resource/", legacyKey: "resource",
	},
	"galgame_quiz": {
		typ: "galgame_quiz", anchorKind: communityclient.AnchorSiteResource, anchorPrefix: "quiz:",
		editPerm: perm.CommentQuizEdit, deletePerm: perm.CommentQuizDelete,
		maxLength: 1007, feedType: "GALGAME_QUIZ_COMMENT_CREATION", counterTable: "galgame_quiz",
		hasOwner: true, linkPrefix: "/galgame-quiz/", legacyKey: "quiz",
	},
	"toolset": {
		typ: "toolset", anchorKind: communityclient.AnchorSiteResource, anchorPrefix: "toolset:",
		editPerm: perm.CommentToolsetEdit, deletePerm: perm.CommentToolsetDelete,
		maxLength: 1007, feedType: "TOOLSET_COMMENT_CREATION", counterTable: "galgame_toolset",
		hasOwner: true, linkPrefix: "/toolset/", legacyKey: "toolset",
	},
	"website": {
		typ: "website", anchorKind: communityclient.AnchorSiteResource, anchorPrefix: "website:",
		editPerm: perm.CommentWebsiteEdit, deletePerm: perm.CommentWebsiteDelete,
		maxLength: 1007, feedType: "GALGAME_WEBSITE_COMMENT_CREATION", counterTable: "galgame_website",
		linkPrefix: "/website/", legacyKey: "website",
	},
}

func (sp subjectSpec) anchorID(id int) string {
	return sp.anchorPrefix + strconv.Itoa(id)
}

func subjectFromAnchor(kind int32, anchorID string) (subjectSpec, int, bool) {
	for _, sp := range subjects {
		if sp.anchorKind != kind {
			continue
		}
		raw := anchorID
		if sp.anchorPrefix != "" {
			var ok bool
			if raw, ok = strings.CutPrefix(anchorID, sp.anchorPrefix); !ok {
				continue
			}
		}
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 || strconv.Itoa(id) != raw {
			return subjectSpec{}, 0, false
		}
		return sp, id, true
	}
	return subjectSpec{}, 0, false
}
