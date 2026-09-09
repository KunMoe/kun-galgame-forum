package app

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/perm"

	"github.com/gofiber/fiber/v3"
	fiberCors "github.com/gofiber/fiber/v3/middleware/cors"
)

// Fiber matches routes in REGISTRATION ORDER, and an empty-prefix Group
// registers its middleware as Use() on the parent path, applying it to every
// route below. Both facts are load-bearing throughout this file; see
// router_gate_test.go for the outage that proved it.
func (a *App) setupRoutes() {
	middleware.SecureCookies = a.Config.Server.Mode == "prod"

	a.Fiber.Use(fiberCors.New(middleware.CORS(a.Config.CORS.AllowOrigins)))

	// Deliberately touches neither DB nor Redis: the container HEALTHCHECK reads
	// this, and a transient backing-store blip must not flap the container.
	a.Fiber.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := a.Fiber.Group("/api")

	api.Use(middleware.NamePreference)

	api.Get("/home", a.HomeHandler.GetHome)

	api.Post("/trust/callback", a.TrustHandler.Callback)

	auth := api.Group("/auth")
	auth.Post("/oauth/callback", a.OAuthHandler.Callback)
	auth.Post("/logout", a.OAuthHandler.Logout)

	userAuth := middleware.Auth(a.Redis, a.OAuthClient)
	// No rate limiter on purpose. The once-per-day gate is the `daily_check_in`
	// flag reset at calendar midnight by the daily cron. A 24h-rolling limiter
	// spilled past midnight, blocked legitimate next-day check-ins, and masked
	// "已签到" with a generic "操作过于频繁" 400.
	api.Post("/user/check-in", userAuth, a.UserHandler.CheckIn)
	api.Get("/user/status", userAuth, a.UserHandler.GetStatus)
	// Every fixed /user/* path must stay ahead of /user/:id, or the literal
	// segment binds as :id.
	api.Get("/user/notification-preferences", userAuth, a.UserHandler.GetNotificationPreferences)
	api.Put("/user/notification-preferences", userAuth, a.UserHandler.UpdateNotificationPreferences)
	api.Get("/user/moemoepoint/log", userAuth, a.UserHandler.GetMoemoepointLog)
	api.Get("/user/search", userAuth, a.UserHandler.SearchMention)

	api.Get("/user/creator/status", userAuth, a.CreatorHandler.Status)
	api.Post("/user/creator/apply", userAuth, a.CreatorHandler.Apply)

	api.Put("/user/bio", userAuth, a.UserProfileHandler.UpdateBio)
	api.Put("/user/username", userAuth, a.UserProfileHandler.UpdateUsername)
	api.Post("/user/avatar", userAuth, a.UserProfileHandler.UploadAvatar)

	api.Get("/user/:id/floating", a.UserHandler.GetFloatingCard)
	api.Get("/user/:id", a.UserHandler.GetProfile)
	api.Get("/user/:id/galgames", a.UserHandler.GetUserGalgames)
	api.Get("/user/:id/galgame-comments", a.UserHandler.GetUserGalgameComments)
	api.Get("/user/:id/topics", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.UserHandler.GetUserTopics)
	api.Get("/user/:id/replies", a.UserHandler.GetUserReplies)
	api.Get("/user/:id/comments", a.UserHandler.GetUserComments)
	api.Get("/user/:id/resources", a.UserHandler.GetUserResources)
	api.Get("/user/:id/ratings", a.UserHandler.GetUserRatings)
	api.Get("/user/:id/toolsets", a.ToolsetHandler.GetUserToolsets)

	api.Get("/ranking/galgame", a.RankingHandler.GetGalgameRanking)
	api.Get("/ranking/topic", a.RankingHandler.GetTopicRanking)
	api.Get("/ranking/user", a.RankingHandler.GetUserRanking)

	api.Get("/section", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.SectionHandler.GetSectionTopics)
	api.Get("/category", a.SectionHandler.GetCategories)

	api.Get("/doc/article", a.DocArticleHandler.GetArticles)
	api.Get("/doc/article/:slug", a.DocArticleHandler.GetArticleBySlug)
	api.Get("/doc/category", a.DocCategoryHandler.GetCategories)
	api.Get("/doc/tag", a.DocTagHandler.GetTags)

	api.Get("/website-category", a.WebsiteCategoryHandler.GetWebsiteCategories)
	api.Get("/website-category/:name", a.WebsiteCategoryHandler.GetWebsiteCategory)
	api.Get("/website-tag", a.WebsiteTagHandler.GetWebsiteTags)
	api.Get("/website-tag-group", a.WebsiteTagGroupHandler.GetWebsiteTagGroups)
	api.Get("/website-tag/:name", a.WebsiteTagHandler.GetWebsiteTagDetail)

	api.Get("/update/history", a.UpdateHandler.GetHistory)
	api.Get("/update/todo", a.UpdateHandler.GetTodos)

	api.Get("/friend-link", a.FriendLinkHandler.List)

	api.Get("/perm/bundles", a.AdminRolePermissionHandler.GetBundles)

	api.Get("/activity", a.ActivityHandler.GetActivity)
	api.Get("/activity/tab", a.ActivityHandler.GetTab)
	api.Get("/activity/timeline", a.ActivityHandler.GetTimeline)

	api.Get("/news", a.NewsHandler.GetFeed)
	api.Get("/news/sources", a.NewsHandler.GetSources)
	api.Get("/news/archive", a.NewsHandler.GetArchive)
	api.Get("/news/month", a.NewsHandler.GetMonth)

	api.Get("/resource", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.TopicHandler.GetResourceList)

	api.Get("/search", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.SearchHandler.Search)
	api.Get("/search/quick", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.SearchHandler.QuickSearch)
	api.Get("/search/overview", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.SearchHandler.Overview)
	api.Get("/search/entity", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.SearchHandler.SearchEntities)
	api.Get("/search/entity/resolve", middleware.OptionalAuth(a.Redis, a.OAuthClient), a.SearchHandler.ResolveEntities)

	api.Get("/rss/topic", a.RSSHandler.GetTopicRSS)
	api.Get("/rss/galgame", a.RSSHandler.GetGalgameRSS)

	api.Get("/toolset/:id/resource/detail", a.ToolsetResourceHandler.GetResourceDetail)

	api.Get("/galgame", a.GalgameHandler.GetList)
	// Every literal /galgame/<segment> route must precede /galgame/:gid: the
	// catch-all binds "mine" / "calendar" / "drafts" as a gid and then fails
	// inside GetDetail with Atoi("mine").
	api.Get("/galgame/mine", userAuth, a.GalgameSubmissionHandler.ListMine)
	api.Get(
		"/galgame/audited",
		userAuth,
		middleware.RequirePermission(perm.GalgameClaimReview),
		a.GalgameSubmissionHandler.ListAudit,
	)
	api.Get(
		"/galgame/search/wizard",
		userAuth,
		a.GalgameSubmissionHandler.SearchWithPending,
	)
	api.Get("/galgame/search/picker", a.GalgameQuizHandler.SearchGalgames)
	api.Get("/galgame/calendar", a.GalgameCalendarHandler.GetMonth)
	api.Get("/galgame/collected-calendar", a.GalgameHandler.CollectedCalendar)
	api.Get("/galgame/calendar/today", a.GalgameCalendarHandler.GetToday)
	api.Get("/galgame/calendar/pending", a.GalgameCalendarHandler.GetPending)
	api.Get("/galgame/calendar/tba", a.GalgameCalendarHandler.GetTBA)
	api.Get("/galgame/calendar/upcoming", a.GalgameCalendarHandler.GetUpcoming)
	api.Get("/galgame/drafts", a.GalgameDraftsHandler.GetDrafts)
	api.Get("/galgame/:gid/edit/diff", a.GalgameEditHandler.Diff)
	api.Get("/galgame/:gid/edit/proposals", a.GalgameEditHandler.GameProposals)
	api.Get("/galgame-tag", a.GalgameEntityHandler.GetTagList)
	api.Get("/galgame-tag/search", a.GalgameEntityHandler.SearchTags)
	api.Get("/galgame-tag/multi", a.GalgameEntityHandler.GetMultiTagGalgames)
	api.Get("/galgame-tag/:id", a.GalgameEntityHandler.GetTagDetail)
	api.Get("/galgame-official", a.GalgameEntityHandler.GetOfficialList)
	api.Get("/galgame-official/search", a.GalgameEntityHandler.SearchOfficials)
	api.Get("/galgame-official/legacy/:id", a.GalgameEntityHandler.ResolveLegacyOfficial)
	api.Get("/galgame-official/:id", a.GalgameEntityHandler.GetOfficialDetail)
	api.Get("/galgame-official/:id/relation-graph", a.GalgameEntityHandler.GetOfficialRelationGraph)
	api.Get("/galgame-engine", a.GalgameEntityHandler.GetEngineList)
	api.Get("/galgame-engine/:id", a.GalgameEntityHandler.GetEngineDetail)
	api.Get("/galgame-staff/search", a.GalgameEntityHandler.SearchStaff)
	api.Get("/galgame-staff/:id", a.GalgameEntityHandler.GetStaffDetail)
	api.Get("/galgame-character/search", a.GalgameEntityHandler.SearchCharacters)
	api.Get("/galgame-character/:id", a.GalgameEntityHandler.GetCharacterDetail)
	api.Get("/galgame-series", a.GalgameEntityHandler.GetSeriesList)
	api.Get("/galgame-series/cards", a.GalgameEntityHandler.GetSeriesCards)
	api.Get("/galgame-series/:id", a.GalgameEntityHandler.GetSeriesDetail)
	api.Get("/galgame-rating/all", a.GalgameRatingHandler.GetAllRatings)
	api.Get(
		"/galgame-quiz/:id/answers",
		middleware.OptionalAuth(a.Redis, a.OAuthClient),
		a.GalgameQuizHandler.GetQuizAnswers,
	)

	optAuth := api.Group("", middleware.OptionalAuth(a.Redis, a.OAuthClient))
	// The rating and resource families sit here, not in the public group above:
	// leaving them there made optionalUID return 0 unconditionally and silently
	// broke the FindLikedSet batch fix, so every row rendered as not-liked for
	// logged-in viewers.
	optAuth.Get("/galgame-resource", a.GalgameResourceHandler.GetResourceList)
	optAuth.Get("/galgame-resource/:id/detail", a.GalgameResourceHandler.GetResourceDownloadDetail)
	optAuth.Get("/galgame-resource/:id", a.GalgameResourceHandler.GetResourceDetail)

	optAuth.Get("/galgame-rating/:id", a.GalgameRatingHandler.GetRatingDetail)

	optAuth.Get("/galgame-quiz/all", a.GalgameQuizHandler.GetAllQuizzes)
	optAuth.Get("/galgame-quiz/:id", a.GalgameQuizHandler.GetQuizPlay)

	// On `api` with an explicit middleware, and BEFORE /topic/:tid: a later
	// static /topic/draft is captured by the earlier param route (tid="draft").
	topicDraftAuth := middleware.Auth(a.Redis, a.OAuthClient)
	api.Get("/topic/draft", topicDraftAuth, a.TopicDraftHandler.List)
	api.Post("/topic/draft", topicDraftAuth, a.TopicDraftHandler.Save)
	api.Get("/topic/draft/:id", topicDraftAuth, a.TopicDraftHandler.Get)
	api.Delete("/topic/draft/:id", topicDraftAuth, a.TopicDraftHandler.Delete)

	optAuth.Get("/topic", a.TopicHandler.GetList)
	optAuth.Get("/topic/:tid", a.TopicHandler.GetDetail)
	optAuth.Get("/topic/:tid/upvotes", a.TopicHandler.GetUpvotes)
	optAuth.Get("/topic/:tid/reaction/history", a.TopicHandler.GetTopicReactionHistory)
	optAuth.Get("/topic/:tid/reply", a.ReplyHandler.GetReplies)
	optAuth.Get("/topic/:tid/reply/detail", a.ReplyHandler.GetReplyDetail)
	optAuth.Get("/topic/:tid/reply/locate", a.ReplyHandler.GetReplyLocate)
	optAuth.Get("/topic/:tid/reply/reaction/history", a.ReplyHandler.GetReplyReactionHistory)
	optAuth.Get("/topic/:tid/poll/topic", a.PollHandler.GetPollsByTopic)
	optAuth.Get("/topic/:tid/poll/log", a.PollHandler.GetVoteLog)
	optAuth.Get("/topic/:tid/lottery/topic", a.LotteryHandler.GetLotteriesByTopic)
	optAuth.Get("/topic/:tid/lottery/entrants", a.LotteryHandler.GetEntrants)

	optAuth.Get("/galgame/:gid/resource/all", a.GalgameResourceHandler.GetGalgameResources)
	// Both comment READ halves must mount before the auth boundary below, or
	// anonymous reads start demanding a session. Their writes mount after it.
	a.GalgameCommunityCommentHandler.RegisterReads(optAuth)
	a.ResourceCommentHandler.RegisterReads(optAuth)
	optAuth.Get("/galgame/:gid/link/all", a.GalgameProxyHandler.GetGalgameLinks)
	optAuth.Get("/galgame/:gid/edit/revisions", a.GalgameEditHandler.Revisions)
	optAuth.Get("/galgame/:gid", a.GalgameHandler.GetDetail)

	optAuth.Get("/galgame/collection/:cid", a.GalgameCollectionHandler.GetDetail)
	optAuth.Get("/user/:id/collections", a.GalgameCollectionHandler.GetUserCollections)

	optAuth.Get("/website", a.WebsiteHandler.GetWebsites)
	optAuth.Get("/website/:domain", a.WebsiteHandler.GetWebsiteDetail)

	optAuth.Get("/toolset", a.ToolsetHandler.GetList)
	optAuth.Get("/toolset/:id", a.ToolsetHandler.GetDetail)
	optAuth.Get("/toolset/:id/practicality", a.ToolsetPracticalityHandler.GetPracticality)

	// THE AUTH BOUNDARY. This empty-prefix group registers Auth as Use() on
	// "/api", so it applies to EVERY route below this line. Nothing public or
	// optAuth may be registered after this point.
	authed := api.Group("", middleware.Auth(a.Redis, a.OAuthClient))
	authed.Get("/auth/me", a.OAuthHandler.Me)

	authed.Get("/perm/mine", a.AdminUserPermissionHandler.GetMine)

	authed.Get("/topic/interactions/mine", a.TopicHandler.MyInteractions)
	authed.Post("/topic", a.TopicHandler.Create)
	authed.Put("/topic/:tid", a.TopicHandler.Update)
	authed.Put("/topic/:tid/like", a.TopicHandler.ToggleLike)
	authed.Put("/topic/:tid/dislike", a.TopicHandler.ToggleDislike)
	authed.Put("/topic/:tid/upvote", a.TopicHandler.Upvote)
	authed.Put("/topic/:tid/favorite", a.TopicHandler.ToggleFavorite)
	authed.Put("/topic/:tid/reaction", a.TopicHandler.ToggleReaction)
	authed.Put("/topic/:tid/hide", a.TopicHandler.ToggleHide)
	authed.Put("/topic/:tid/best-answer", a.TopicHandler.SetBestAnswer)

	authed.Post("/topic/:tid/reply", a.ReplyHandler.CreateReply)
	authed.Put("/topic/:tid/reply", a.ReplyHandler.UpdateReply)
	authed.Delete("/topic/:tid/reply", a.ReplyHandler.DeleteReply)
	authed.Put("/topic/:tid/reply/like", a.ReplyHandler.ToggleReplyLike)
	authed.Put("/topic/:tid/reply/dislike", a.ReplyHandler.ToggleReplyDislike)
	authed.Put("/topic/:tid/reply/reaction", a.ReplyHandler.ToggleReplyReaction)
	authed.Put("/topic/:tid/reply/pin", a.ReplyHandler.PinReply)

	authed.Post("/topic/:tid/comment", a.TopicCommentHandler.CreateComment)
	authed.Put("/topic/:tid/comment", a.TopicCommentHandler.UpdateComment)
	authed.Put("/topic/:tid/comment/like", a.TopicCommentHandler.ToggleCommentLike)
	authed.Delete("/topic/:tid/comment", a.TopicCommentHandler.DeleteComment)

	authed.Post("/topic/:tid/poll", a.PollHandler.CreatePoll)
	authed.Put("/topic/:tid/poll", a.PollHandler.UpdatePoll)
	authed.Delete("/topic/:tid/poll", a.PollHandler.DeletePoll)
	authed.Post("/topic/:tid/poll/vote", a.PollHandler.Vote)

	authed.Post("/topic/:tid/lottery", a.LotteryHandler.CreateLottery)
	authed.Put("/topic/:tid/lottery", a.LotteryHandler.UpdateLottery)
	authed.Delete("/topic/:tid/lottery", a.LotteryHandler.DeleteLottery)
	authed.Post("/topic/:tid/lottery/enter", a.LotteryHandler.Enter)
	authed.Post("/topic/:tid/lottery/withdraw", a.LotteryHandler.Withdraw)
	authed.Post("/topic/:tid/lottery/draw", a.LotteryHandler.Draw)
	authed.Post("/topic/:tid/lottery/cancel", a.LotteryHandler.Cancel)
	// POST, never GET — see LotteryHandler.Claim.
	authed.Post("/topic/:tid/lottery/claim", a.LotteryHandler.Claim)
	authed.Put("/topic/:tid/lottery/fulfillment", a.LotteryHandler.SetFulfillment)

	authed.Get("/message", a.MessageHandler.GetMessages)
	authed.Get("/message/muted", a.MessageHandler.GetMutedMessages)
	authed.Delete("/message/:id", a.MessageHandler.DeleteMessage)
	authed.Put("/message/system/read", a.MessageHandler.MarkAllRead)
	authed.Get("/message/admin", a.MessageHandler.GetSystemMessages)
	authed.Put("/message/admin/read", a.MessageHandler.MarkAdminRead)
	authed.Get("/message/nav/system", a.MessageHandler.GetNavSummary)
	authed.Get("/message/nav/contact", a.MessageChatHandler.GetNavContact)
	authed.Get("/message/chat/history", a.MessageChatHandler.GetChatHistory)
	authed.Post("/message/chat/send", a.MessageChatHandler.SendChatMessage)
	authed.Post("/message/chat/recall", a.MessageChatHandler.RecallChatMessage)

	authed.Post("/image/topic", a.ImageHandler.UploadTopicImage)
	authed.Post("/image/cover", a.ImageHandler.UploadCoverImage)
	authed.Post("/image/message", a.ImageHandler.UploadMessageImage)
	authed.Post("/image/galgame", a.ImageHandler.UploadGalgameImage)

	authed.Get("/report/reasons", a.TrustHandler.GetReasons)
	authed.Post("/report/submit", a.TrustHandler.SubmitReport)

	authed.Put("/website/:domain/like", a.WebsiteHandler.ToggleLike)
	authed.Put("/website/:domain/favorite", a.WebsiteHandler.ToggleFavorite)

	authed.Post("/galgame/submit", a.GalgameSubmissionHandler.Submit)
	authed.Post("/galgame/:gid/resubmit", a.GalgameSubmissionHandler.Resubmit)
	authed.Delete("/galgame/:gid", a.GalgameSubmissionHandler.Withdraw)
	authed.Delete("/galgame/:gid/draft", a.GalgameSubmissionHandler.DeleteDraft)

	authed.Get("/galgame/interactions/mine", a.GalgameHandler.MyInteractions)
	authed.Get("/galgame/playtime/mine", a.GalgamePlaytimeHandler.ListMine)
	authed.Put("/galgame/:gid/like", a.GalgameHandler.ToggleLike)
	// Unlike every other catalog write here, this one travels as the USER: the
	// session's OAuth token goes out as a Bearer and the registry derives the
	// actor from it.
	authed.Put("/galgame/:gid/playtime", a.GalgamePlaytimeHandler.Report)
	authed.Put("/galgame/:gid/cover/:coverId/vote", a.GalgameCoverVoteHandler.Vote)
	authed.Delete("/galgame/:gid/cover/:coverId/vote", a.GalgameCoverVoteHandler.Unvote)
	authed.Post("/galgame/collection", a.GalgameCollectionHandler.Create)
	authed.Patch("/galgame/collection/:cid", a.GalgameCollectionHandler.Update)
	authed.Delete("/galgame/collection/:cid", a.GalgameCollectionHandler.Delete)
	authed.Get("/galgame/:gid/collections/mine", a.GalgameCollectionHandler.MyCollectionsForGalgame)
	authed.Put("/galgame/:gid/collections", a.GalgameCollectionHandler.SetMembership)

	a.GalgameCommunityCommentHandler.RegisterWrites(authed)

	a.ResourceCommentHandler.RegisterWrites(authed)

	authed.Post("/galgame/:gid/resource", a.GalgameResourceHandler.CreateResource)
	authed.Put("/galgame/:gid/resource", a.GalgameResourceHandler.UpdateResource)
	authed.Delete("/galgame/:gid/resource", a.GalgameResourceHandler.DeleteResource)
	authed.Put("/galgame/:gid/resource/like", a.GalgameResourceHandler.ToggleLike)
	authed.Put("/galgame/:gid/resource/valid", a.GalgameResourceHandler.MarkValid)
	authed.Put("/galgame/:gid/resource/expired", a.GalgameResourceHandler.MarkExpired)

	authed.Post("/galgame-rating", a.GalgameRatingHandler.CreateRating)
	authed.Put("/galgame-rating/:id", a.GalgameRatingHandler.UpdateRating)
	authed.Delete("/galgame-rating/:id", a.GalgameRatingHandler.DeleteRating)
	authed.Put("/galgame-rating/:id/like", a.GalgameRatingHandler.ToggleLike)

	authed.Get("/galgame-quiz/mine/answered", a.GalgameQuizHandler.GetMyAnswered)
	authed.Get("/galgame-quiz/mine/favorites", a.GalgameQuizHandler.GetMyFavorites)
	authed.Post("/galgame-quiz", a.GalgameQuizHandler.CreateQuiz)
	authed.Delete("/galgame-quiz/:id", a.GalgameQuizHandler.DeleteQuiz)
	authed.Post("/galgame-quiz/:id/answer", a.GalgameQuizHandler.AnswerQuiz)
	authed.Put("/galgame-quiz/:id/quality", a.GalgameQuizHandler.RateQuizQuality)
	authed.Put("/galgame-quiz/:id/favorite", a.GalgameQuizHandler.ToggleQuizFavorite)
	authed.Get("/galgame-quiz/:id/edit", a.GalgameQuizHandler.GetQuizForEdit)
	authed.Put("/galgame-quiz/:id", a.GalgameQuizHandler.UpdateQuiz)

	authed.Get("/galgame/:gid/edit/bootstrap", a.GalgameEditHandler.Bootstrap)
	authed.Post("/galgame/:gid/edit/proposals", a.GalgameEditHandler.Submit)
	authed.Get("/galgame-edit/mine", a.GalgameEditHandler.Mine)
	authed.Post("/galgame-edit/proposals/:id/withdraw", a.GalgameEditHandler.Withdraw)
	authed.Get("/galgame-edit/queue", middleware.RequireModerator(), a.GalgameEditHandler.Queue)
	authed.Get("/galgame-edit/proposals/:id", a.GalgameEditHandler.ProposalDetail)
	authed.Post("/galgame-edit/proposals/:id/amend", a.GalgameEditHandler.Amend)
	authed.Post("/galgame-edit/proposals/:id/merge", a.GalgameEditHandler.Merge)
	authed.Post("/galgame-edit/proposals/:id/decline", a.GalgameEditHandler.Decline)
	authed.Post("/galgame/:gid/edit/revert", a.GalgameEditHandler.Revert)

	authed.Post("/toolset", a.ToolsetHandler.Create)
	authed.Put("/toolset/:id", a.ToolsetHandler.Update)
	authed.Delete("/toolset/:id", a.ToolsetHandler.Delete)
	authed.Put("/toolset/:id/practicality", a.ToolsetPracticalityHandler.UpsertPracticality)
	authed.Post("/toolset/:id/resource", a.ToolsetResourceHandler.CreateResource)
	authed.Put("/toolset/:id/resource", a.ToolsetResourceHandler.UpdateResource)
	authed.Delete("/toolset/:id/resource", a.ToolsetResourceHandler.DeleteResource)
	authed.Post("/toolset/:id/upload/init", a.ToolsetUploadHandler.UploadInit)
	authed.Post("/toolset/:id/upload/complete", a.ToolsetUploadHandler.UploadComplete)
	authed.Post("/toolset/:id/upload/resume", a.ToolsetUploadHandler.UploadResume)
	authed.Post("/toolset/:id/upload/abort", a.ToolsetUploadHandler.UploadAbort)

	// Every admin gate below is PER-ROUTE, never Group("", middleware.X()) — see
	// router_gate_test.go for the 2026-07-21..2026-08-07 outage that rule
	// encodes. Where a route proxies infra, the local Require* is a VIEW gate
	// deciding which page opens; infra re-checks and owns the outcome. Never
	// tighten one into a second answer that can disagree with the engine.
	admin := authed.Group("")
	admin.Get("/admin/overview/all", middleware.RequirePermission(perm.AdminDashboard), a.AdminOverviewHandler.GetOverview)
	admin.Get("/admin/overview/stats", middleware.RequirePermission(perm.AdminDashboard), a.AdminOverviewHandler.GetStats)

	admin.Get("/admin/user/:id/content-stats", middleware.RequirePermission(perm.UserPurgeContent), a.AdminPurgeHandler.GetUserContentStats)
	admin.Delete("/admin/user/:id/content", middleware.RequirePermission(perm.UserPurgeContent), a.AdminPurgeHandler.PurgeUserContent)
	admin.Get("/admin/topic/hidden", middleware.RequirePermission(perm.TopicViewHidden), a.AdminTopicHandler.ListHidden)
	admin.Get("/admin/topic/:tid/purge-stats", middleware.RequirePermission(perm.TopicDeleteAny), a.AdminTopicHandler.PurgeStats)
	admin.Delete("/admin/topic/:tid", middleware.RequirePermission(perm.TopicDeleteAny), a.AdminTopicHandler.Delete)

	// RequireAdmin, not RequirePermission: overrides must never be able to lock
	// admins out of the surface that repairs overrides.
	rolePermAdmin := authed.Group("")
	rolePermAdmin.Get("/admin/role-permissions", middleware.RequireAdmin(), a.AdminRolePermissionHandler.GetMatrix)
	rolePermAdmin.Put("/admin/role-permissions/:role", middleware.RequireAdmin(), a.AdminRolePermissionHandler.Replace)
	rolePermAdmin.Get("/admin/user-permissions/:uid", middleware.RequireAdmin(), a.AdminUserPermissionHandler.GetView)
	rolePermAdmin.Put("/admin/user-permissions/:uid", middleware.RequireAdmin(), a.AdminUserPermissionHandler.Replace)
	rolePermAdmin.Get("/admin/permission-audit", middleware.RequireAdmin(), a.AdminPermissionAuditHandler.List)

	trustAdmin := authed.Group("")
	trustAdmin.Get("/admin/trust/review-items", middleware.RequirePermission(perm.TrustReview), a.TrustHandler.ListReviewItems)
	trustAdmin.Get("/admin/trust/review-items/:id", middleware.RequirePermission(perm.TrustReview), a.TrustHandler.GetReviewItem)
	trustAdmin.Post("/admin/trust/review-items/:id/claim", middleware.RequirePermission(perm.TrustReview), a.TrustHandler.ClaimReviewItem)
	trustAdmin.Post("/admin/trust/review-items/:id/decide", middleware.RequirePermission(perm.TrustReview), a.TrustHandler.DecideReviewItem)

	galgameAdmin := authed.Group("")
	galgameAdmin.Get("/admin/galgame/submissions", middleware.RequirePermission(perm.GalgameClaimReview), a.GalgameClaimReviewHandler.PendingQueue)
	galgameAdmin.Post(
		"/admin/galgame/:gid/review",
		middleware.RequirePermission(perm.GalgameClaimReview),
		a.GalgameClaimReviewHandler.Review,
	)
	galgameAdmin.Put(
		"/admin/galgame/:gid/resource-publish-ban",
		middleware.RequirePermission(perm.GalgameBanResourcePublish),
		a.GalgameResourceHandler.SetResourcePublishBan,
	)

	docAdmin := authed.Group("")
	docAdmin.Get("/admin/doc/article", middleware.RequirePermission(perm.DocEdit), a.DocArticleHandler.GetAdminArticles)
	docAdmin.Post("/doc/article", middleware.RequirePermission(perm.DocCreate), a.DocArticleHandler.CreateArticle)
	docAdmin.Put("/doc/article", middleware.RequirePermission(perm.DocEdit), a.DocArticleHandler.UpdateArticle)
	docAdmin.Put("/doc/article/reorder", middleware.RequirePermission(perm.DocEdit), a.DocArticleHandler.ReorderArticles)
	docAdmin.Put("/doc/article/pin", middleware.RequirePermission(perm.DocEdit), a.DocArticleHandler.SetArticlePin)
	docAdmin.Delete("/doc/article", middleware.RequirePermission(perm.DocDelete), a.DocArticleHandler.DeleteArticle)
	docAdmin.Post("/doc/category", middleware.RequirePermission(perm.DocCreate), a.DocCategoryHandler.CreateCategory)
	docAdmin.Put("/doc/category", middleware.RequirePermission(perm.DocEdit), a.DocCategoryHandler.UpdateCategory)
	docAdmin.Delete("/doc/category", middleware.RequirePermission(perm.DocDelete), a.DocCategoryHandler.DeleteCategory)
	docAdmin.Post("/doc/tag", middleware.RequirePermission(perm.DocCreate), a.DocTagHandler.CreateTag)
	docAdmin.Put("/doc/tag", middleware.RequirePermission(perm.DocEdit), a.DocTagHandler.UpdateTag)
	docAdmin.Delete("/doc/tag", middleware.RequirePermission(perm.DocDelete), a.DocTagHandler.DeleteTag)

	wsAdmin := authed.Group("")
	wsAdmin.Post("/website", middleware.RequirePermission(perm.WebsiteCreate), a.WebsiteHandler.CreateWebsite)
	wsAdmin.Put("/website/:domain", middleware.RequirePermission(perm.WebsiteEdit), a.WebsiteHandler.UpdateWebsite)
	wsAdmin.Delete("/website/:domain", middleware.RequirePermission(perm.WebsiteDelete), a.WebsiteHandler.DeleteWebsite)
	wsAdmin.Post("/website-category", middleware.RequirePermission(perm.WebsiteCreate), a.WebsiteCategoryHandler.CreateWebsiteCategory)
	wsAdmin.Put("/website-category", middleware.RequirePermission(perm.WebsiteEdit), a.WebsiteCategoryHandler.UpdateWebsiteCategory)
	wsAdmin.Delete("/website-category", middleware.RequirePermission(perm.WebsiteDelete), a.WebsiteCategoryHandler.DeleteWebsiteCategory)
	wsAdmin.Post("/website-tag", middleware.RequirePermission(perm.WebsiteCreate), a.WebsiteTagHandler.CreateWebsiteTag)
	wsAdmin.Put("/website-tag", middleware.RequirePermission(perm.WebsiteEdit), a.WebsiteTagHandler.UpdateWebsiteTag)
	wsAdmin.Delete("/website-tag", middleware.RequirePermission(perm.WebsiteDelete), a.WebsiteTagHandler.DeleteWebsiteTag)
	wsAdmin.Post("/website-tag-group", middleware.RequirePermission(perm.WebsiteCreate), a.WebsiteTagGroupHandler.CreateWebsiteTagGroup)
	wsAdmin.Put("/website-tag-group", middleware.RequirePermission(perm.WebsiteEdit), a.WebsiteTagGroupHandler.UpdateWebsiteTagGroup)
	wsAdmin.Delete("/website-tag-group", middleware.RequirePermission(perm.WebsiteDelete), a.WebsiteTagGroupHandler.DeleteWebsiteTagGroup)

	updateAdmin := authed.Group("")
	updateAdmin.Post("/update/history", middleware.RequirePermission(perm.UpdateLogCreate), a.UpdateHandler.CreateHistory)
	updateAdmin.Put("/update/history", middleware.RequirePermission(perm.UpdateLogEdit), a.UpdateHandler.UpdateHistory)
	updateAdmin.Delete("/update/history", middleware.RequirePermission(perm.UpdateLogDelete), a.UpdateHandler.DeleteHistory)
	updateAdmin.Post("/update/todo/claim", middleware.RequirePermission(perm.UpdateLogEdit), a.UpdateHandler.ClaimTodo)
	updateAdmin.Delete("/update/todo", middleware.RequirePermission(perm.UpdateLogDelete), a.UpdateHandler.DeleteTodo)

	// Creating, editing, completing and discarding a todo are open to any
	// logged-in user; ownership is enforced in the handlers.
	authed.Post("/update/todo", a.UpdateHandler.CreateTodo)
	authed.Put("/update/todo", a.UpdateHandler.UpdateTodo)
	authed.Post("/update/todo/complete", a.UpdateHandler.CompleteTodo)
	authed.Post("/update/todo/discard", a.UpdateHandler.DiscardTodo)

	friendAdmin := authed.Group("")
	friendAdmin.Post("/admin/friend-link", middleware.RequirePermission(perm.FriendLinkCreate), a.FriendLinkHandler.Create)
	friendAdmin.Put("/admin/friend-link", middleware.RequirePermission(perm.FriendLinkEdit), a.FriendLinkHandler.Update)
	friendAdmin.Delete("/admin/friend-link", middleware.RequirePermission(perm.FriendLinkDelete), a.FriendLinkHandler.Delete)
	friendAdmin.Put("/admin/friend-link/reorder", middleware.RequirePermission(perm.FriendLinkEdit), a.FriendLinkHandler.Reorder)
}
