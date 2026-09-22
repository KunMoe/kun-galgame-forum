package perm

type Permission string

const (
	TopicEditAny        Permission = "topic.edit_any"
	TopicHide           Permission = "topic.hide"
	TopicViewHidden     Permission = "topic.view_hidden"
	TopicViewRestricted Permission = "topic.view_restricted"
	TopicDeleteAny      Permission = "topic.delete_any"
	TopicSetBestAnswer  Permission = "topic.set_best_answer"

	ReplyEditAny   Permission = "reply.edit_any"
	ReplyDeleteAny Permission = "reply.delete_any"
	ReplyPin       Permission = "reply.pin"

	CommentTopicEdit      Permission = "comment.topic.edit"
	CommentTopicDelete    Permission = "comment.topic.delete"
	CommentGalgameEdit    Permission = "comment.galgame.edit"
	CommentGalgameDelete  Permission = "comment.galgame.delete"
	CommentRatingEdit     Permission = "comment.rating.edit"
	CommentRatingDelete   Permission = "comment.rating.delete"
	CommentWebsiteEdit    Permission = "comment.website.edit"
	CommentWebsiteDelete  Permission = "comment.website.delete"
	CommentToolsetEdit    Permission = "comment.toolset.edit"
	CommentToolsetDelete  Permission = "comment.toolset.delete"
	CommentResourceEdit   Permission = "comment.resource.edit"
	CommentResourceDelete Permission = "comment.resource.delete"
	CommentQuizEdit       Permission = "comment.quiz.edit"
	CommentQuizDelete     Permission = "comment.quiz.delete"

	PollCreateAny      Permission = "poll.create_any"
	PollEditAny        Permission = "poll.edit_any"
	PollDeleteAny      Permission = "poll.delete_any"
	PollViewRestricted Permission = "poll.view_restricted"

	LotteryCreateAny      Permission = "lottery.create_any"
	LotteryManageAny      Permission = "lottery.manage_any"
	LotteryViewRestricted Permission = "lottery.view_restricted"

	GalgameBanResourcePublish Permission = "galgame.ban_resource_publish"
	GalgameClaimReview        Permission = "galgame.claim.review"

	CollectionEditAny   Permission = "collection.edit_any"
	CollectionDeleteAny Permission = "collection.delete_any"

	QuizEditAny   Permission = "quiz.edit_any"
	QuizDeleteAny Permission = "quiz.delete_any"

	ResourceEditAny   Permission = "resource.edit_any"
	ResourceDeleteAny Permission = "resource.delete_any"

	RatingDeleteAny Permission = "rating.delete_any"

	ToolsetEditAny           Permission = "toolset.edit_any"
	ToolsetDeleteAny         Permission = "toolset.delete_any"
	ToolsetResourceEditAny   Permission = "toolset.resource.edit_any"
	ToolsetResourceDeleteAny Permission = "toolset.resource.delete_any"
	ToolsetUploadBypass      Permission = "toolset.upload_bypass"

	DocCreate Permission = "doc.create"
	DocEdit   Permission = "doc.edit"
	DocDelete Permission = "doc.delete"

	WebsiteCreate Permission = "website.create"
	WebsiteEdit   Permission = "website.edit"
	WebsiteDelete Permission = "website.delete"

	FriendLinkCreate Permission = "friend_link.create"
	FriendLinkEdit   Permission = "friend_link.edit"
	FriendLinkDelete Permission = "friend_link.delete"

	UpdateLogCreate Permission = "update_log.create"
	UpdateLogEdit   Permission = "update_log.edit"
	UpdateLogDelete Permission = "update_log.delete"
	UpdateLogReopen Permission = "update_log.reopen"

	TrustReview Permission = "trust.review"

	AdminDashboard   Permission = "admin.dashboard"
	UserPurgeContent Permission = "user.purge_content"
)

var moderatorPerms = []Permission{
	TopicEditAny,
	TopicHide,
	TopicViewHidden,
	TopicViewRestricted,
	TopicSetBestAnswer,
	ReplyEditAny,
	ReplyDeleteAny,
	ReplyPin,
	CommentTopicEdit,
	CommentTopicDelete,
	CommentGalgameEdit,
	CommentGalgameDelete,
	CommentRatingEdit,
	CommentRatingDelete,
	CommentWebsiteEdit,
	CommentWebsiteDelete,
	CommentToolsetEdit,
	CommentToolsetDelete,
	CommentResourceEdit,
	CommentResourceDelete,
	CommentQuizEdit,
	CommentQuizDelete,
	PollCreateAny,
	PollEditAny,
	PollDeleteAny,
	PollViewRestricted,
	LotteryCreateAny,
	LotteryManageAny,
	LotteryViewRestricted,
	GalgameBanResourcePublish,
	GalgameClaimReview,
	CollectionEditAny,
	CollectionDeleteAny,
	QuizEditAny,
	QuizDeleteAny,
	ResourceEditAny,
	ResourceDeleteAny,
	RatingDeleteAny,
	ToolsetEditAny,
	ToolsetDeleteAny,
	ToolsetResourceEditAny,
	ToolsetResourceDeleteAny,
	ToolsetUploadBypass,
	DocCreate,
	DocEdit,
	DocDelete,
	WebsiteCreate,
	WebsiteEdit,
	WebsiteDelete,
	FriendLinkCreate,
	FriendLinkEdit,
	FriendLinkDelete,
	UpdateLogCreate,
	UpdateLogEdit,
	UpdateLogDelete,
	TrustReview,
}

var adminPerms = append(append([]Permission{}, moderatorPerms...), AdminDashboard, UserPurgeContent, TopicDeleteAny, UpdateLogReopen)

var Bundles = map[string][]Permission{
	"moderator": moderatorPerms,
	"admin":     adminPerms,
	"ren":       adminPerms,
}

func Can(roles []string, p Permission) bool {
	return current.Load().can(roles, p)
}

type resolverT struct {
	grants map[string]map[Permission]struct{}
}

func (r *resolverT) can(roles []string, p Permission) bool {
	for _, roleName := range roles {
		if set, ok := r.grants[roleName]; ok {
			if _, ok := set[p]; ok {
				return true
			}
		}
	}
	return false
}
