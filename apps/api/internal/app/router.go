package app

import (
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	appreleaseapiv1 "kun-galgame-api/internal/apprelease/apiv1"
	docapiv1 "kun-galgame-api/internal/doc/apiv1"
	friendlinkapiv1 "kun-galgame-api/internal/friendlink/apiv1"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	galgameentityv1 "kun-galgame-api/internal/galgame/entityapiv1"
	messageapiv1 "kun-galgame-api/internal/message/apiv1"
	"kun-galgame-api/internal/middleware"
	permissionapiv1 "kun-galgame-api/internal/permission/apiv1"
	sectionapiv1 "kun-galgame-api/internal/section/apiv1"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	trustapiv1 "kun-galgame-api/internal/trust/apiv1"
	updateapiv1 "kun-galgame-api/internal/update/apiv1"
	toolsetapiv1 "kun-galgame-api/internal/toolset/apiv1"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	websiteapiv1 "kun-galgame-api/internal/website/apiv1"
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

	deps := apiv1.Deps{Redis: a.Redis}
	if a.Authn != nil {
		deps.Resolver = a.Authn
	}
	topicReads := a.newTopicV1()
	a.APIv1 = apiv1.Setup(a.Fiber, deps,
		topicapiv1.Register(topicReads),
		topicapiv1.RegisterWrites(a.newTopicV1Writes(topicReads)),
		topicapiv1.RegisterInteractions(a.newTopicV1Interactions(topicReads)),
		topicapiv1.RegisterPolls(a.newTopicV1Polls(topicReads)),
		topicapiv1.RegisterDrafts(a.newTopicV1Drafts()),
		topicapiv1.RegisterLotteries(a.newTopicV1Lotteries(topicReads)),
		topicapiv1.RegisterAdminTopics(a.newTopicV1Admin(topicReads)),
		galgameapiv1.Register(a.GalgameV1),
		galgameentityv1.Register(a.GalgameEntityV1),
		wallapiv1.Register(a.WallV1),
		userapiv1.Register(a.newUserV1()),
		messageapiv1.Register(a.newMessageV1()),
		websiteapiv1.Register(a.newWebsiteV1()),
		toolsetapiv1.Register(a.newToolsetV1()),
		updateapiv1.Register(a.newUpdateV1()),
		trustapiv1.Register(a.TrustV1),
		permissionapiv1.Register(a.newPermissionV1()),
		docapiv1.Register(a.newDocV1()),
		friendlinkapiv1.Register(a.newFriendLinkV1()),
		appreleaseapiv1.Register(a.newAppReleaseV1()),
		sectionapiv1.Register(a.newSectionV1()),
	)

	// Deliberately touches neither DB nor Redis: the container HEALTHCHECK reads
	// this, and a transient backing-store blip must not flap the container.
	a.Fiber.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := a.Fiber.Group("/api")

	api.Use(middleware.NamePreference)
	api.Use(middleware.ContentStance(a.Redis, a.BearerStance))

	api.Get("/home", a.HomeHandler.GetHome)

	api.Post("/trust/callback", a.TrustHandler.Callback)

	auth := api.Group("/auth")
	auth.Post("/oauth/callback", a.OAuthHandler.Callback)
	auth.Post("/logout", a.OAuthHandler.Logout)

	userAuth := a.Authn.Auth()
	api.Get("/user/:id/galgames", a.UserHandler.GetUserGalgames)
	api.Get("/user/:id/galgame-comments", a.UserHandler.GetUserGalgameComments)
	api.Get("/user/:id/topics", a.Authn.OptionalAuth(), a.UserHandler.GetUserTopics)
	api.Get("/user/:id/replies", a.UserHandler.GetUserReplies)
	api.Get("/user/:id/comments", a.UserHandler.GetUserComments)
	api.Get("/user/:id/resources", a.UserHandler.GetUserResources)
	api.Get("/user/:id/ratings", a.UserHandler.GetUserRatings)


	api.Get("/ranking/galgame", a.RankingHandler.GetGalgameRanking)
	api.Get("/ranking/topic", a.RankingHandler.GetTopicRanking)
	api.Get("/ranking/user", a.RankingHandler.GetUserRanking)

	api.Get("/activity", a.ActivityHandler.GetActivity)
	api.Get("/activity/tab", a.ActivityHandler.GetTab)
	api.Get("/activity/timeline", a.ActivityHandler.GetTimeline)

	api.Get("/news", a.NewsHandler.GetFeed)
	api.Get("/news/sources", a.NewsHandler.GetSources)
	api.Get("/news/archive", a.NewsHandler.GetArchive)
	api.Get("/news/month", a.NewsHandler.GetMonth)

	api.Get("/search", a.Authn.OptionalAuth(), a.SearchHandler.Search)
	api.Get("/search/quick", a.Authn.OptionalAuth(), a.SearchHandler.QuickSearch)
	api.Get("/search/overview", a.Authn.OptionalAuth(), a.SearchHandler.Overview)
	api.Get("/search/gal-comment", a.Authn.OptionalAuth(), a.SearchHandler.SearchGalComments)
	api.Get("/search/entity", a.Authn.OptionalAuth(), a.SearchHandler.SearchEntities)
	api.Get("/search/entity/resolve", a.Authn.OptionalAuth(), a.SearchHandler.ResolveEntities)

	api.Get("/rss/topic", a.RSSHandler.GetTopicRSS)
	api.Get("/rss/galgame", a.RSSHandler.GetGalgameRSS)

	api.Get("/galgame", a.GalgameHandler.GetList)
	// Every literal /galgame/<segment> route must precede /galgame/:id: the
	// catch-all binds "mine" / "calendar" / "drafts" as a work id and then fails
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
	api.Get("/galgame/:id/edit/diff", a.GalgameEditHandler.Diff)
	api.Get("/galgame/:id/edit/proposals", a.GalgameEditHandler.GameProposals)
	api.Get("/galgame-rating/all", a.GalgameRatingHandler.GetAllRatings)
	api.Get(
		"/galgame-quiz/:id/answers",
		a.Authn.OptionalAuth(),
		a.GalgameQuizHandler.GetQuizAnswers,
	)

	optAuth := api.Group("", a.Authn.OptionalAuth())
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

	optAuth.Get("/galgame/:id/resource/all", a.GalgameResourceHandler.GetGalgameResources)
	// Both comment READ halves must mount before the auth boundary below, or
	// anonymous reads start demanding a session. Their writes mount after it.
	optAuth.Get("/galgame/:id/link/all", a.GalgameProxyHandler.GetGalgameLinks)
	optAuth.Get("/galgame/:id/edit/revisions", a.GalgameEditHandler.Revisions)
	optAuth.Get("/galgame/:id", a.GalgameHandler.GetDetail)

	optAuth.Get("/galgame/collection/:cid", a.GalgameCollectionHandler.GetDetail)
	optAuth.Get("/user/:id/collections", a.GalgameCollectionHandler.GetUserCollections)



	// THE AUTH BOUNDARY. This empty-prefix group registers Auth as Use() on
	// "/api", so it applies to EVERY route below this line. Nothing public or
	// optAuth may be registered after this point.
	authed := api.Group("", a.Authn.Auth())
	authed.Get("/auth/me", a.OAuthHandler.Me)


	authed.Post("/image/topic", a.ImageHandler.UploadTopicImage)
	authed.Post("/image/cover", a.ImageHandler.UploadCoverImage)
	authed.Post("/image/message", a.ImageHandler.UploadMessageImage)
	authed.Post("/image/galgame", a.ImageHandler.UploadGalgameImage)

	authed.Post("/galgame/submit", a.GalgameSubmissionHandler.Submit)
	authed.Post("/galgame/:id/resubmit", a.GalgameSubmissionHandler.Resubmit)
	authed.Delete("/galgame/:id", a.GalgameSubmissionHandler.Withdraw)
	authed.Delete("/galgame/:id/draft", a.GalgameSubmissionHandler.DeleteDraft)

	authed.Get("/galgame/interactions/mine", a.GalgameHandler.MyInteractions)
	authed.Get("/galgame/playtime/mine", a.GalgamePlaytimeHandler.ListMine)
	authed.Put("/galgame/:id/like", a.GalgameHandler.ToggleLike)
	// Unlike every other catalog write here, this one travels as the USER: the
	// session's OAuth token goes out as a Bearer and the registry derives the
	// actor from it.
	authed.Put("/galgame/:id/playtime", a.GalgamePlaytimeHandler.Report)
	authed.Put("/galgame/:id/cover/:coverId/vote", a.GalgameCoverVoteHandler.Vote)
	authed.Delete("/galgame/:id/cover/:coverId/vote", a.GalgameCoverVoteHandler.Unvote)
	authed.Post("/galgame/collection", a.GalgameCollectionHandler.Create)
	authed.Patch("/galgame/collection/:cid", a.GalgameCollectionHandler.Update)
	authed.Delete("/galgame/collection/:cid", a.GalgameCollectionHandler.Delete)
	authed.Get("/galgame/:id/collections/mine", a.GalgameCollectionHandler.MyCollectionsForGalgame)
	authed.Put("/galgame/:id/collections", a.GalgameCollectionHandler.SetMembership)

	authed.Post("/galgame/:id/resource", a.GalgameResourceHandler.CreateResource)
	authed.Put("/galgame/:id/resource", a.GalgameResourceHandler.UpdateResource)
	authed.Delete("/galgame/:id/resource", a.GalgameResourceHandler.DeleteResource)
	authed.Put("/galgame/:id/resource/like", a.GalgameResourceHandler.ToggleLike)
	authed.Put("/galgame/:id/resource/valid", a.GalgameResourceHandler.MarkValid)
	authed.Put("/galgame/:id/resource/expired", a.GalgameResourceHandler.MarkExpired)

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

	authed.Get("/galgame/:id/edit/bootstrap", a.GalgameEditHandler.Bootstrap)
	authed.Post("/galgame/:id/edit/proposals", a.GalgameEditHandler.Submit)
	authed.Get("/galgame-edit/mine", a.GalgameEditHandler.Mine)
	authed.Post("/galgame-edit/proposals/:id/withdraw", a.GalgameEditHandler.Withdraw)
	authed.Get("/galgame-edit/queue", middleware.RequireModerator(), a.GalgameEditHandler.Queue)
	authed.Get("/galgame-edit/proposals/:id", a.GalgameEditHandler.ProposalDetail)
	authed.Post("/galgame-edit/proposals/:id/amend", a.GalgameEditHandler.Amend)
	authed.Post("/galgame-edit/proposals/:id/merge", a.GalgameEditHandler.Merge)
	authed.Post("/galgame-edit/proposals/:id/decline", a.GalgameEditHandler.Decline)
	authed.Post("/galgame/:id/edit/revert", a.GalgameEditHandler.Revert)

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

	galgameAdmin := authed.Group("")
	galgameAdmin.Get("/admin/galgame/submissions", middleware.RequirePermission(perm.GalgameClaimReview), a.GalgameClaimReviewHandler.PendingQueue)
	galgameAdmin.Post(
		"/admin/galgame/:id/review",
		middleware.RequirePermission(perm.GalgameClaimReview),
		a.GalgameClaimReviewHandler.Review,
	)
	galgameAdmin.Put(
		"/admin/galgame/:id/resource-publish-ban",
		middleware.RequirePermission(perm.GalgameBanResourcePublish),
		a.GalgameResourceHandler.SetResourcePublishBan,
	)

}

func (a *App) newTopicV1() *topicapiv1.Service {
	if a.DB == nil || a.UserClient == nil {
		return nil
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	convert := &content.Converter{
		CDNBase:  cdn,
		SiteBase: apiv1.SiteOrigin,
		Images:   a.ImageMeta,
		Users:    a.UserClient.Users,
	}
	return topicapiv1.New(
		topicRepo.NewTopicListRepository(a.DB),
		topicRepo.NewTopicRepository(a.DB),
		topicRepo.NewTopicTaxonomyRepository(a.DB),
		topicRepo.NewReplyRepository(a.DB),
		topicRepo.NewCommentRepository(a.DB),
		a.UserClient,
		convert,
		cdn,
	)
}
