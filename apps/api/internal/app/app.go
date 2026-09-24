package app

import (
	"context"
	"log/slog"
	"time"

	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/community/anchor"
	communitynotify "kun-galgame-api/internal/community/notify"
	communitytrust "kun-galgame-api/internal/community/trust"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	calendarapiv1 "kun-galgame-api/internal/galgame/calendarapiv1"
	"kun-galgame-api/internal/galgame/client"
	galgameentityv1 "kun-galgame-api/internal/galgame/entityapiv1"
	ratingapiv1 "kun-galgame-api/internal/galgame/ratingapiv1"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	resourceapiv1 "kun-galgame-api/internal/galgame/resourceapiv1"
	galgameService "kun-galgame-api/internal/galgame/service"
	imageapiv1 "kun-galgame-api/internal/image/apiv1"
	"kun-galgame-api/internal/infrastructure/cache"
	cronPkg "kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/infrastructure/storelink"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	newsapiv1 "kun-galgame-api/internal/news/apiv1"
	overviewapiv1 "kun-galgame-api/internal/overview/apiv1"
	quizapiv1 "kun-galgame-api/internal/quiz/apiv1"
	rankingapiv1 "kun-galgame-api/internal/ranking/apiv1"
	rankingRepo "kun-galgame-api/internal/ranking/repository"
	searchapiv1 "kun-galgame-api/internal/search/apiv1"
	searchRepo "kun-galgame-api/internal/search/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	topicService "kun-galgame-api/internal/topic/service"
	trustapiv1 "kun-galgame-api/internal/trust/apiv1"
	"kun-galgame-api/internal/trust/enforce"
	"kun-galgame-api/internal/trust/gate"
	trustHandler "kun-galgame-api/internal/trust/handler"
	"kun-galgame-api/internal/user/handler"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/internal/user/service"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	"kun-galgame-api/pkg/artifactclient"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/dlsite"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/moyuclient"
	"kun-galgame-api/pkg/newsclient"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/secretbox"
	"kun-galgame-api/pkg/storeclient"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Fiber              *fiber.App
	DB                 *gorm.DB
	Redis              *redis.Client
	Config             *config.Config
	OAuthClient        *oauth.Client
	UserState          *repository.StateRepository
	TopicAward         topicapiv1.AwardFunc
	TrustCheck         *gate.CheckService
	TrustScan          *gate.ScanService
	Notifier           msgService.Notifier
	Messages           *msgService.MessageService
	UserClient         *userclient.Client
	UserService        *service.UserService
	CreatorService     *galgameService.CreatorService
	Authn              *middleware.Authenticator
	BearerStance       *middleware.BearerStance
	ImageMeta          func(hashes []string) map[string]imageclient.ImageMeta
	GalgameV1          *galgameapiv1.Service
	GalgameEntityV1    *galgameentityv1.Service
	GalgameCalendarV1  *calendarapiv1.Service
	GalgameRatingV1    *ratingapiv1.Service
	QuizCatalog        quizapiv1.Catalog
	ResourceCatalog    resourceapiv1.Catalog
	ResourceClaim      resourceapiv1.ClaimFunc
	ResourceChecker    resourceapiv1.ShareChecker
	StoreLinks         *storelink.Resolver
	WallV1             *wallapiv1.Service
	Community          *communityclient.Client
	ContributedWorkIDs func(context.Context, int64) ([]int, error)
	OverviewV1         *overviewapiv1.Service
	RankingV1          *rankingapiv1.Service
	SearchV1           *searchapiv1.Service
	ActivityV1         *activityapiv1.Service
	TrustV1            *trustapiv1.Service

	OAuthHandler        *handler.OAuthHandler
	LotteryService      *topicService.LotteryService
	AdminPurge          *adminService.PurgeService
	TrustHandler        *trustHandler.TrustHandler
	NewsV1              *newsapiv1.Service
	ImagesV1            *imageapiv1.Service
	Artifact            *artifactclient.Client
	FileStorage         *storage.S3Client
	CronStop            func()
	RolePermStop        func()
	StoreLinkStop       func()
	CommunityNotifyStop func()
	APIv1               huma.API
}

func New(cfg *config.Config) *App {
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	rdb := cache.NewRedis(cfg.Redis)
	fileStorageClient := storage.NewS3(cfg.FileStorage)
	if fileStorageClient == nil {
		slog.Warn("FILE_STORAGE_* 未配置, 删除历史 s3 工具集资源将不可用")
	}

	markdown.SetContentImageCDNBase(cfg.NextMoeAPI.ImageCDNBase)

	userStateRepo := repository.NewStateRepository(db)
	userStatsRepo := repository.NewUserStatsRepository(db)
	messageRepository := msgRepo.NewMessageRepository(db)

	gc := client.New(
		cfg.NextMoeAPI.BaseURL,
		cfg.NextMoeAPI.APIKey,
		cfg.NextMoeAPI.ImageCDNBase,
	).WithRedis(rdb)

	newsCli := newsclient.New(newsclient.Config{
		BaseURL: cfg.NewsAPI.BaseURL,
		APIKey:  cfg.NewsAPI.APIKey,
	})
	if newsCli.Configured() {
		slog.Info("news face client configured", "base_url", cfg.NewsAPI.BaseURL)
	} else {
		slog.Warn("news face client NOT configured; /news returns 503 — set KUN_NEWS_API_KEY (scope news:read)")
	}

	moyuCli := moyuclient.New(moyuclient.Config{
		BaseURL: cfg.MoyuAPI.BaseURL,
		APIKey:  cfg.MoyuAPI.APIKey,
	})
	if moyuCli.Configured() {
		slog.Info("moyu patch face client configured", "base_url", cfg.MoyuAPI.BaseURL)
	} else {
		slog.Warn("moyu patch face client NOT configured; the galgame patch tab stays hidden — set KUN_NEXTMOE_API_KEY or KUN_MOYU_API_KEY")
	}

	oauthClient := oauth.NewClient(cfg.OAuth)

	uc := userclient.New(userclient.Config{
		BaseURL:      cfg.OAuth.ServerURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		ImageCDNBase: cfg.NextMoeAPI.ImageCDNBase,
	})

	moemoepoint.SetDefault(moemoepoint.NewAwarder(uc, db))

	var imgCli *imageclient.Client
	var imageMeta *imageclient.MetaResolver
	if cfg.ImageClient.ClientID != "" && cfg.ImageClient.ClientSecret != "" {
		imgCli = imageclient.New(imageclient.Config{
			BaseURL:      cfg.ImageClient.BaseURL,
			CDNBase:      cfg.NextMoeAPI.ImageCDNBase,
			ClientID:     cfg.ImageClient.ClientID,
			ClientSecret: cfg.ImageClient.ClientSecret,
		})
		slog.Info("image_service client configured", "base_url", cfg.ImageClient.BaseURL)

		imageMeta = imgCli.NewMetaResolver(0)
		markdown.SetContentImageMetaResolver(imageMeta.Resolve)

		gc.SetImageMetaResolver(imageMeta.Resolve)
	} else {
		slog.Warn("image_service client NOT configured; /image/galgame upload will return 未配置 — set KUN_IMAGE_CLIENT_ID / KUN_IMAGE_CLIENT_SECRET")
	}

	artClientID := cfg.ArtifactClient.ClientID
	if artClientID == "" {
		artClientID = cfg.OAuth.ClientID
	}
	artClientSecret := cfg.ArtifactClient.ClientSecret
	if artClientSecret == "" {
		artClientSecret = cfg.OAuth.ClientSecret
	}
	artCli := artifactclient.New(artifactclient.Config{
		BaseURL:      cfg.ArtifactClient.BaseURL,
		ClientID:     artClientID,
		ClientSecret: artClientSecret,
	})
	if artCli.Configured() {
		slog.Info("artifact service client configured", "base_url", cfg.ArtifactClient.BaseURL)
	} else {
		slog.Warn("artifact service client NOT configured; toolset upload will return 未配置 — set KUN_ARTIFACT_CLIENT_BASE_URL + OAuth creds")
	}

	trustCli := trustclient.New(trustclient.Config{
		BaseURL:      cfg.Trust.BaseURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
	})
	if trustCli.Configured() {
		slog.Info("trust service client configured", "base_url", cfg.Trust.BaseURL)
	} else {
		slog.Warn("trust service client NOT configured; reporting returns 未启用 — set KUN_TRUST_BASE_URL + OAuth creds")
	}

	if trustCli.Configured() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			results, err := trustCli.EnsureSubjectKinds(ctx, gate.CanonicalSubjectKindItems())
			if err != nil {
				slog.Warn("trust subject-kind ensure failed (non-fatal)", "err", err)
				return
			}
			changed := make([]string, 0, len(results))
			for _, r := range results {
				if r.Result != "unchanged" {
					changed = append(changed, r.Key+"="+r.Result)
				}
			}
			if len(changed) > 0 {
				slog.Info("trust subject-kind ensure applied", "changed", changed, "total", len(results))
			} else {
				slog.Info("trust subject-kind ensure: all kinds already registered", "total", len(results))
			}
		}()
	}

	var trustChecker gate.Checker
	if cfg.Trust.CheckEnabled && trustCli.Configured() {
		trustChecker = trustCli
	}
	trustCheck := gate.NewCheckService(trustChecker)
	if trustCheck.Enabled() {
		slog.Info("trust check gate enabled (synchronous word-list gate on all forum user-text writes)")
	} else {
		slog.Info("trust check gate disabled (KUN_TRUST_CHECK_ENABLED off or trust client unconfigured)")
	}

	var trustScanner gate.Scanner
	if cfg.Trust.ScanEnabled && trustCli.Configured() {
		trustScanner = trustCli
	}
	trustScan := gate.NewScanService(trustScanner)
	if trustScan.Enabled() {
		slog.Info("trust shadow scan enabled (async post-commit scan on all forum user-text writes)")
	} else {
		slog.Info("trust shadow scan disabled (KUN_TRUST_SCAN_ENABLED off or trust client unconfigured)")
	}

	catalogCli := catalogclient.New(catalogclient.Config{
		BaseURL: cfg.Catalog.BaseURL,
		AppKey:  cfg.NextMoeAPI.APIKey,
	})
	if catalogCli.AppConfigured() {
		slog.Info("catalog service client configured", "base_url", cfg.Catalog.BaseURL)
	} else {
		slog.Warn("catalog service client NOT configured; galgame edit review returns 未启用 — set KUN_CATALOG_API_BASE + KUN_NEXTMOE_API_KEY")
	}

	commClientID := cfg.Community.ClientID
	if commClientID == "" {
		commClientID = cfg.OAuth.ClientID
	}
	commClientSecret := cfg.Community.ClientSecret
	if commClientSecret == "" {
		commClientSecret = cfg.OAuth.ClientSecret
	}
	communityCli := communityclient.New(communityclient.Config{
		BaseURL:      cfg.Community.BaseURL,
		ClientID:     commClientID,
		ClientSecret: commClientSecret,
	})
	if communityCli.Configured() {
		slog.Info("community comment backend configured", "base_url", cfg.Community.BaseURL)
	} else {
		slog.Warn("community comment backend NOT configured; comments degrade (reads empty / writes 503) — set KUN_COMMUNITY_API_BASE + OAuth creds")
	}
	anchorResolver := anchor.New(db, gc)

	var storeCli *storeclient.Client
	if cfg.Dlsite.StoreConfigured() {
		storeCli = storeclient.New(storeclient.Config{
			BaseURL: cfg.Dlsite.StoreAPIBase,
			APIKey:  cfg.Dlsite.StoreAPIKey,
		})
	}
	storeLinks := storelink.New(storelink.Options{
		DB:           db,
		Client:       storeCli,
		LinkTemplate: cfg.Dlsite.LinkTemplate,
		StaticCoupon: cfg.Dlsite.CouponURL,
	})
	if cfg.Dlsite.Configured() || storeLinks.Configured() {
		slog.Info("dlsite purchase link configured",
			"verified_whitelist", dlsite.VerifiedCount(),
			"short_links", storeLinks.Configured(),
			"template_fallback", cfg.Dlsite.Configured(),
			"static_coupon", cfg.Dlsite.CouponURL != "")
	} else {
		slog.Info("dlsite purchase link off (KUN_DLSITE_LINK_TEMPLATE and KUN_STORE_API_KEY both unset); 补票提示 renders its plain form")
	}
	communityBooster := communitytrust.New(communityCli, rdb, db)

	var bearerVerifier middleware.AccessTokenVerifier
	var bearerStance *middleware.BearerStance
	if cfg.Bearer.Enabled() {
		verifier := oauth.NewAccessTokenVerifier(
			oauth.NewJWKS(cfg.Bearer.JWKSURL), cfg.Bearer.Issuer, cfg.Bearer.ClientIDs,
		)
		bearerVerifier = verifier
		bearerStance = middleware.NewBearerStance(verifier, oauthClient, rdb)
		slog.Info("Bearer 直连已开启", "issuer", cfg.Bearer.Issuer, "clients", cfg.Bearer.ClientIDs)
	}
	authn := middleware.NewAuthenticator(rdb, oauthClient, middleware.NewBearer(
		bearerVerifier, rdb,
		func(userID int, roles []string) error {
			if err := userStateRepo.Ensure(userID); err != nil {
				return err
			}
			communityBooster.Boost(userID, roles)
			return nil
		},
	))

	var linkChecker *linkcheck.Client
	if cfg.LinkChecker.BaseURL != "" && cfg.LinkChecker.APIKey != "" {
		linkChecker = linkcheck.New(linkcheck.Config{
			BaseURL:              cfg.LinkChecker.BaseURL,
			APIKey:               cfg.LinkChecker.APIKey,
			CFAccessClientID:     cfg.LinkChecker.CFAccessClientID,
			CFAccessClientSecret: cfg.LinkChecker.CFAccessClientSecret,
		})
		slog.Info("link-live-checker gate configured",
			"base_url", cfg.LinkChecker.BaseURL,
			"cf_access", cfg.LinkChecker.CFAccessClientID != "")
	} else {
		slog.Warn("link-live-checker NOT configured; resource 报告失效 falls back to legacy single-report-expires — set LINK_CHECKER_BASE_URL / LINK_CHECKER_API_KEY")
	}

	galgameLocalRepo := galgameRepo.NewGalgameRepository(db)
	galgameMergeRepo := galgameRepo.NewGalgameMergeRepository(db)
	galgameUserStatsSvc := galgameService.NewGalgameUserStatsService(catalogCli, gc, galgameLocalRepo)

	authService := service.NewAuthService(userStateRepo, rdb, oauthClient, uc)
	userService := service.NewUserService(userStateRepo, userStatsRepo, rdb, gc, galgameUserStatsSvc, uc, communityCli)
	messageSvc := msgService.NewMessageService(communityCli)
	notifier := msgService.NewNotifier(messageRepository)

	topicRepository := topicRepo.NewTopicRepository(db)
	replyRepository := topicRepo.NewReplyRepository(db)
	topicCommentRepo := topicRepo.NewCommentRepository(db)
	lotteryRepository := topicRepo.NewLotteryRepository(db)
	replySvc := topicService.NewReplyService(replyRepository)
	commentSvc := topicService.NewCommentService(replyRepository, topicCommentRepo)
	lotteryBox, err := secretbox.New(cfg.Lottery.CodeKey)
	if err != nil {
		slog.Error("KUN_LOTTERY_CODE_KEY 无效, 兑换码托管已禁用 (抽奖其余功能不受影响)", "error", err)
	}
	if lotteryBox == nil {
		slog.Warn("KUN_LOTTERY_CODE_KEY 未设置; 抽奖将拒绝「系统托管兑换码」奖项, 而不是明文存码")
	}
	lotterySvc := topicService.NewLotteryService(
		lotteryRepository, userStateRepo, uc, notifier, lotteryBox)
	lotteryDrawer := topicService.NewLotteryDrawer(lotterySvc)

	galgameCommunityPostRepo := galgameRepo.NewCommunityPostRepository(db)
	creatorSvc := galgameService.NewCreatorService(galgameRepo.NewRatingStore(db), galgameUserStatsSvc, uc)
	galgameContributorRepo := galgameRepo.NewGalgameContributorRepository(db)
	galgameCollectionRepo := galgameRepo.NewGalgameCollectionRepository(db)
	galgamePlaytimeSvc := galgameService.NewPlaytimeService(gc, catalogCli)
	galgameClaimSync := galgameService.NewGalgameClaimEventSync(catalogCli, galgameLocalRepo, rdb)
	galgameRevisionSync := galgameService.NewGalgameEditRevisionSync(catalogCli, gc, db, rdb)
	galgameContributorSync := galgameService.NewGalgameContributorSync(catalogCli, galgameContributorRepo, rdb)
	galgameMergeSync := galgameService.NewGalgameMergeSync(gc, galgameMergeRepo, rdb)
	galgameCatalogMirror := galgameService.NewGalgameCatalogMirror(gc, galgameLocalRepo, rdb, galgameMergeSync)

	adminOverviewRepo := adminRepo.NewOverviewRepository(db)
	adminPurgeRepo := adminRepo.NewPurgeRepository(db)
	adminPurgeSvc := adminService.NewPurgeService(adminPurgeRepo, uc, communityCli, catalogCli)
	adminRolePermRepo := adminRepo.NewRolePermissionRepository(db)
	adminUserPermRepo := adminRepo.NewUserPermissionRepository(db)
	adminPermSync := adminService.NewPermissionOverrideSync(adminRolePermRepo, adminUserPermRepo)

	galgameCommentEnforcer := galgameService.NewGalgameCommentEnforcer(communityCli, galgameCommunityPostRepo)
	trustRegistry := enforce.Registry{
		"forum_topic": {
			Hide: func(_ context.Context, id int) error {
				return topicRepository.UpdateFields(id, map[string]any{"status": 1, "hidden_by": "trust"})
			},
			Remove: func(_ context.Context, id int) error {
				return topicRepository.UpdateFields(id, map[string]any{"status": 1, "hidden_by": "trust"})
			},
			AuthorID: func(_ context.Context, id int) (int, error) {
				t, err := topicRepository.FindByID(id)
				if err != nil {
					return 0, nil
				}
				return t.UserID, nil
			},
		},
		"forum_reply": {
			Hide:   func(_ context.Context, id int) error { return replyRepository.SetStatus(id, 1) },
			Remove: func(_ context.Context, id int) error { return replySvc.ModerationRemove(id) },
			AuthorID: func(_ context.Context, id int) (int, error) {
				r, err := replyRepository.FindByID(id)
				if err != nil {
					return 0, nil
				}
				return r.UserID, nil
			},
		},
		"forum_comment": {
			Hide:   func(_ context.Context, id int) error { return topicCommentRepo.SetStatus(id, 1) },
			Remove: func(_ context.Context, id int) error { return commentSvc.ModerationRemove(id) },
			AuthorID: func(_ context.Context, id int) (int, error) {
				c, err := topicCommentRepo.FindCommentByID(id)
				if err != nil {
					return 0, nil
				}
				return c.UserID, nil
			},
		},
		"galgame_comment": {
			Hide:     galgameCommentEnforcer.Tombstone,
			Remove:   galgameCommentEnforcer.Tombstone,
			AuthorID: galgameCommentEnforcer.AuthorID,
		},
	}
	trustEnforce := enforce.NewService(db, trustRegistry, nil)

	app := &App{
		DB: db, Redis: rdb, Config: cfg, OAuthClient: oauthClient,
		UserState:         userStateRepo,
		TrustCheck:        trustCheck,
		TrustScan:         trustScan,
		Notifier:          notifier,
		Messages:          messageSvc,
		UserClient:        uc,
		UserService:       userService,
		CreatorService:    creatorSvc,
		Authn:             authn,
		BearerStance:      bearerStance,
		ImageMeta:         imageMetaResolve(imageMeta),
		GalgameV1:         galgameapiv1.New(gc, moyuCli, uc, rdb, cfg.NextMoeAPI.ImageCDNBase).WithWork(db, catalogCli, storeLinks, moemoepoint.Award).WithUserPlane(trustCheck, trustScan, galgameCollectionRepo).WithEditing(notifier),
		GalgameEntityV1:   galgameentityv1.New(gc, db, cfg.NextMoeAPI.ImageCDNBase),
		GalgameCalendarV1: calendarapiv1.New(gc, db, cfg.NextMoeAPI.ImageCDNBase),
		GalgameRatingV1:   newRatingV1(db, gc, uc, trustCheck, trustScan, galgamePlaytimeSvc, cfg.NextMoeAPI.ImageCDNBase),
		QuizCatalog:       gc,
		ResourceCatalog:   gc,
		ResourceClaim: func(ctx context.Context, token string, workID int64) *errors.AppError {
			if catalogCli == nil || !catalogCli.Configured() {
				return nil
			}
			_, err := galgameService.AdoptAndPublish(ctx, catalogCli, token, workID)
			return err
		},
		ResourceChecker:    linkChecker,
		StoreLinks:         storeLinks,
		TrustV1:            trustapiv1.New(trustCli, uc, cfg.Trust.Site, cfg.NextMoeAPI.ImageCDNBase),
		WallV1:             newWallV1(db, communityCli, uc, gc, imageMetaResolve(imageMeta), cfg.NextMoeAPI.ImageCDNBase),
		Community:          communityCli,
		ContributedWorkIDs: galgameUserStatsSvc.ContributedWorkIDs,
		ActivityV1:         newActivityV1(db, gc, uc, imageMetaResolve(imageMeta), cfg.NextMoeAPI.ImageCDNBase),
		OAuthHandler:       handler.NewOAuthHandler(authService, cfg.Server.Mode == "prod", communityBooster),
		LotteryService:     lotterySvc,
		OverviewV1:         overviewapiv1.New(adminOverviewRepo, nil),
		AdminPurge:         adminPurgeSvc,
		TrustHandler:       trustHandler.NewTrustHandler(trustEnforce, cfg.Trust.CallbackSecret),
		NewsV1:             newsapiv1.New(newsCli, uc, cfg.NextMoeAPI.ImageCDNBase),
		ImagesV1:           imageapiv1.New(imgCli, catalogCli, db, cfg.NextMoeAPI.ImageCDNBase),
		Artifact:           artCli,
		FileStorage:        fileStorageClient,
		CronStop: cronPkg.Start(db, rdb, imgCli, cronPkg.Jobs{
			GalgameClaimSync:           galgameClaimSync.Run,
			GalgameRevisionSync:        galgameRevisionSync.Run,
			GalgameContributorSync:     galgameContributorSync.Run,
			GalgameCatalogMirror:       galgameCatalogMirror.RunMirror,
			GalgameCatalogMirrorVerify: galgameCatalogMirror.RunVerify,
			GalgameMergeSync:           galgameMergeSync.Run,
			DlsiteCampaignRefresh:      storeLinks.RefreshCampaign,
			TopicMiniAppDeadlines:      lotteryDrawer.Run,
			UserPurgeArchiveExpiry:     expirePurgeArchive(adminPurgeRepo),
		}),
		StoreLinkStop:       storeLinks.Start(),
		CommunityNotifyStop: communitynotify.New(communityCli, messageRepository, anchorResolver, rdb).Start(),
	}

	if err := adminPermSync.Load(context.Background()); err != nil {
		slog.Warn("加载权限覆盖失败, 暂时沿用编译期基线", "error", err)
	}
	app.RolePermStop = adminPermSync.StartRefresher(60 * time.Second)

	app.Fiber = newFiber()

	app.RankingV1 = rankingapiv1.New(rankingRepo.NewRankingRepository(db), uc, gc, app.newTopicV1(), cfg.NextMoeAPI.ImageCDNBase)
	app.SearchV1 = searchapiv1.New(searchapiv1.Deps{
		Repo: searchRepo.NewSearchRepository(db), Topics: app.newTopicV1(), Users: uc, Galgame: gc,
		Community: communityCli, Anchors: anchorResolver, CDN: cfg.NextMoeAPI.ImageCDNBase,
	})
	app.setupRoutes()
	return app
}

// The image service caps one file at 10 MiB for this client. Without room for
// the multipart framing, a 10 MiB image was refused as a 10 MiB body.
const bodyLimit = 10*1024*1024 + 64*1024

func newFiber() *fiber.App {
	f := fiber.New(fiber.Config{
		ErrorHandler:   globalErrorHandler,
		BodyLimit:      bodyLimit,
		ReadBufferSize: 16 * 1024,
	})
	f.Use(recover.New())
	return f
}

func globalErrorHandler(c fiber.Ctx, err error) error {
	if apiv1.IsV1Path(c.Path()) {
		return apiv1.WriteFiberError(c, err)
	}
	if appErr, ok := err.(*errors.AppError); ok {
		return response.Error(c, appErr)
	}
	// Every retired legacy route reached here as Fiber's 404 and went out as a
	// 500 with an ERROR line: a browser tab still running an old bundle filled
	// the log after each retirement, and its reader was told the server broke.
	if fe, ok := err.(*fiber.Error); ok && fe.Code < fiber.StatusInternalServerError {
		return response.Error(c, errors.New(errors.CodeBiz, "页面版本已过期，请刷新页面后重试", fe.Code))
	}
	slog.Error("未处理的错误", "error", err.Error(), "path", c.Path(), "method", c.Method())
	return response.Error(c, errors.ErrInternal("服务器内部错误"))
}

func imageMetaResolve(r *imageclient.MetaResolver) func([]string) map[string]imageclient.ImageMeta {
	if r == nil {
		return nil
	}
	return r.Resolve
}
