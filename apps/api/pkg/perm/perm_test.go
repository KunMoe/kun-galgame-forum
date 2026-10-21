package perm

import "testing"

var allPerms = []Permission{
	TopicEditAny, TopicHide, TopicViewHidden, TopicViewRestricted, TopicDeleteAny, TopicSetBestAnswer,
	ReplyEditAny, ReplyDeleteAny, ReplyPin,
	CommentTopicEdit, CommentTopicDelete,
	CommentGalgameEdit, CommentGalgameDelete,
	CommentRatingEdit, CommentRatingDelete,
	CommentWebsiteEdit, CommentWebsiteDelete,
	CommentToolsetEdit, CommentToolsetDelete,
	CommentResourceEdit, CommentResourceDelete,
	CommentQuizEdit, CommentQuizDelete,
	PollCreateAny, PollEditAny, PollDeleteAny, PollViewRestricted,
	LotteryCreateAny, LotteryManageAny, LotteryViewRestricted,
	GalgameBanResourcePublish, GalgameClaimReview,
	CollectionEditAny, CollectionDeleteAny,
	QuizEditAny, QuizDeleteAny,
	ResourceEditAny, ResourceDeleteAny,
	RatingDeleteAny,
	ToolsetEditAny, ToolsetDeleteAny,
	ToolsetResourceEditAny, ToolsetResourceDeleteAny, ToolsetUploadBypass,
	DocCreate, DocEdit, DocDelete,
	WebsiteCreate, WebsiteEdit, WebsiteDelete,
	FriendLinkCreate, FriendLinkEdit, FriendLinkDelete,
	UpdateLogCreate, UpdateLogEdit, UpdateLogDelete, UpdateLogReopen,
	TrustReview,
	AdminDashboard, UserPurgeContent,
}

var adminOnly = map[Permission]bool{
	AdminDashboard:   true,
	UserPurgeContent: true,
	TopicDeleteAny:   true,
	UpdateLogReopen:  true,
}

const (
	totalPerms = 60
	modPerms   = 56
)

func isAdminOnly(p Permission) bool { return adminOnly[p] }

func TestVocabularySize(t *testing.T) {
	if len(allPerms) != totalPerms {
		t.Fatalf("allPerms has %d keys, want %d", len(allPerms), totalPerms)
	}
	seen := make(map[Permission]bool, len(allPerms))
	for _, p := range allPerms {
		if p == "" {
			t.Errorf("empty permission string in allPerms")
		}
		if seen[p] {
			t.Errorf("duplicate permission %q", p)
		}
		seen[p] = true
	}
	if len(adminOnly) != totalPerms-modPerms {
		t.Fatalf("adminOnly has %d keys, want %d", len(adminOnly), totalPerms-modPerms)
	}
}

func bundleSet(t *testing.T, roleName string) map[Permission]bool {
	t.Helper()
	set := make(map[Permission]bool)
	for _, p := range Bundles[roleName] {
		if set[p] {
			t.Errorf("role %q bundle lists %q twice", roleName, p)
		}
		set[p] = true
	}
	return set
}

func TestBundleSizes(t *testing.T) {
	cases := []struct {
		role string
		size int
	}{
		{"moderator", modPerms},
		{"admin", totalPerms},
		{"ren", totalPerms},
	}
	for _, tc := range cases {
		if got := len(bundleSet(t, tc.role)); got != tc.size {
			t.Errorf("role %q grants %d perms, want %d", tc.role, got, tc.size)
		}
	}
}

func TestContainment(t *testing.T) {
	mod := bundleSet(t, "moderator")
	admin := bundleSet(t, "admin")
	ren := bundleSet(t, "ren")

	for p := range mod {
		if !admin[p] {
			t.Errorf("moderator holds %q but admin does not (containment broken)", p)
		}
	}
	if len(mod) >= len(admin) {
		t.Errorf("moderator (%d) must be a STRICT subset of admin (%d)", len(mod), len(admin))
	}
	if len(admin) != len(ren) {
		t.Fatalf("admin (%d) and ren (%d) differ in size", len(admin), len(ren))
	}
	for p := range admin {
		if !ren[p] {
			t.Errorf("admin holds %q but ren does not (admin != ren)", p)
		}
	}
}

func wantGranted(roleName string, p Permission) bool {
	switch roleName {
	case "moderator":
		return !isAdminOnly(p)
	case "admin", "ren":
		return true
	default:
		return false
	}
}

func TestMatrix(t *testing.T) {
	singleRoles := []string{"user", "creator", "moderator", "admin", "ren", "unknown"}
	for _, roleName := range singleRoles {
		for _, p := range allPerms {
			got := Can([]string{roleName}, p)
			want := wantGranted(roleName, p)
			if got != want {
				t.Errorf("Can([%s], %q) = %v, want %v", roleName, p, got, want)
			}
		}
	}

	for _, p := range allPerms {
		if Can(nil, p) {
			t.Errorf("Can(nil, %q) = true, want false (empty roles grant nothing)", p)
		}
		if Can([]string{}, p) {
			t.Errorf("Can([], %q) = true, want false (empty roles grant nothing)", p)
		}
	}
}

func TestUnknownPermission(t *testing.T) {
	for _, roleName := range []string{"moderator", "admin", "ren"} {
		if Can([]string{roleName}, Permission("does.not.exist")) {
			t.Errorf("role %q granted an unknown permission", roleName)
		}
	}
}

func TestMultiRoleUnion(t *testing.T) {
	roles := []string{"creator", "admin"}
	if !Can(roles, UserPurgeContent) {
		t.Errorf("Can([creator, admin], user.purge_content) = false, want true")
	}
	if !Can([]string{"user", "moderator"}, TopicHide) {
		t.Errorf("Can([user, moderator], topic.hide) = false, want true")
	}
	if Can([]string{"user", "creator"}, TopicHide) {
		t.Errorf("Can([user, creator], topic.hide) = true, want false")
	}
}
