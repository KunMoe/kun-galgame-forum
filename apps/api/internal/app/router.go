package app

import (
	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	appreleaseapiv1 "kun-galgame-api/internal/apprelease/apiv1"
	authapiv1 "kun-galgame-api/internal/auth/apiv1"
	docapiv1 "kun-galgame-api/internal/doc/apiv1"
	friendlinkapiv1 "kun-galgame-api/internal/friendlink/apiv1"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	calendarapiv1 "kun-galgame-api/internal/galgame/calendarapiv1"
	galgameentityv1 "kun-galgame-api/internal/galgame/entityapiv1"
	ratingapiv1 "kun-galgame-api/internal/galgame/ratingapiv1"
	resourceapiv1 "kun-galgame-api/internal/galgame/resourceapiv1"
	imageapiv1 "kun-galgame-api/internal/image/apiv1"
	messageapiv1 "kun-galgame-api/internal/message/apiv1"
	"kun-galgame-api/internal/middleware"
	newsapiv1 "kun-galgame-api/internal/news/apiv1"
	overviewapiv1 "kun-galgame-api/internal/overview/apiv1"
	permissionapiv1 "kun-galgame-api/internal/permission/apiv1"
	quizapiv1 "kun-galgame-api/internal/quiz/apiv1"
	rankingapiv1 "kun-galgame-api/internal/ranking/apiv1"
	searchapiv1 "kun-galgame-api/internal/search/apiv1"
	sectionapiv1 "kun-galgame-api/internal/section/apiv1"
	toolsetapiv1 "kun-galgame-api/internal/toolset/apiv1"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	trustapiv1 "kun-galgame-api/internal/trust/apiv1"
	updateapiv1 "kun-galgame-api/internal/update/apiv1"
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
		resourceapiv1.Register(a.newGalgameResourceV1()),
		galgameentityv1.Register(a.GalgameEntityV1),
		calendarapiv1.Register(a.GalgameCalendarV1),
		ratingapiv1.Register(a.GalgameRatingV1),
		wallapiv1.Register(a.WallV1),
		userapiv1.Register(a.newUserV1()),
		messageapiv1.Register(a.newMessageV1()),
		websiteapiv1.Register(a.newWebsiteV1()),
		toolsetapiv1.Register(a.newToolsetV1()),
		quizapiv1.Register(a.newQuizV1()),
		updateapiv1.Register(a.newUpdateV1()),
		trustapiv1.Register(a.TrustV1),
		permissionapiv1.Register(a.newPermissionV1()),
		docapiv1.Register(a.newDocV1()),
		friendlinkapiv1.Register(a.newFriendLinkV1()),
		appreleaseapiv1.Register(a.newAppReleaseV1()),
		rankingapiv1.Register(a.RankingV1),
		searchapiv1.Register(a.SearchV1),
		activityapiv1.Register(a.ActivityV1),
		overviewapiv1.Register(a.OverviewV1),
		sectionapiv1.Register(a.newSectionV1()),
		authapiv1.Register(a.newAuthV1()),
		newsapiv1.Register(a.NewsV1),
		imageapiv1.Register(a.ImagesV1),
	)

	// Deliberately touches neither DB nor Redis: the container HEALTHCHECK reads
	// this, and a transient backing-store blip must not flap the container.
	a.Fiber.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := a.Fiber.Group("/api")

	api.Use(middleware.NamePreference)
	api.Use(middleware.ContentStance(a.Redis, a.BearerStance))

	api.Post("/trust/callback", a.TrustHandler.Callback)

	auth := api.Group("/auth")
	auth.Post("/oauth/callback", a.OAuthHandler.Callback)
	auth.Post("/logout", a.OAuthHandler.Logout)

	userAuth := a.Authn.Auth()

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
	api.Get("/galgame/:id/edit/diff", a.GalgameEditHandler.Diff)
	api.Get("/galgame/:id/edit/proposals", a.GalgameEditHandler.GameProposals)

	optAuth := api.Group("", a.Authn.OptionalAuth())
	// The rating and resource families sit here, not in the public group above:
	// leaving them there made optionalUID return 0 unconditionally and silently
	// broke the FindLikedSet batch fix, so every row rendered as not-liked for
	// logged-in viewers.

	// Both comment READ halves must mount before the auth boundary below, or
	// anonymous reads start demanding a session. Their writes mount after it.
	optAuth.Get("/galgame/:id/edit/revisions", a.GalgameEditHandler.Revisions)

	// THE AUTH BOUNDARY. This empty-prefix group registers Auth as Use() on
	// "/api", so it applies to EVERY route below this line. Nothing public or
	// optAuth may be registered after this point.
	authed := api.Group("", a.Authn.Auth())

	authed.Post("/galgame/submit", a.GalgameSubmissionHandler.Submit)
	authed.Post("/galgame/:id/resubmit", a.GalgameSubmissionHandler.Resubmit)
	authed.Delete("/galgame/:id", a.GalgameSubmissionHandler.Withdraw)
	authed.Delete("/galgame/:id/draft", a.GalgameSubmissionHandler.DeleteDraft)

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

	admin.Get("/admin/user/:id/content-stats", middleware.RequirePermission(perm.UserPurgeContent), a.AdminPurgeHandler.GetUserContentStats)
	admin.Delete("/admin/user/:id/content", middleware.RequirePermission(perm.UserPurgeContent), a.AdminPurgeHandler.PurgeUserContent)

	galgameAdmin := authed.Group("")
	galgameAdmin.Get("/admin/galgame/submissions", middleware.RequirePermission(perm.GalgameClaimReview), a.GalgameClaimReviewHandler.PendingQueue)
	galgameAdmin.Post(
		"/admin/galgame/:id/review",
		middleware.RequirePermission(perm.GalgameClaimReview),
		a.GalgameClaimReviewHandler.Review,
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
