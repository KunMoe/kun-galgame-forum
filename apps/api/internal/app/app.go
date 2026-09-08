package app

import (
	"context"
	"log/slog"
	"time"

	activityHandler "kun-galgame-api/internal/activity/handler"
	activityRepo "kun-galgame-api/internal/activity/repository"
	activityService "kun-galgame-api/internal/activity/service"
	adminHandler "kun-galgame-api/internal/admin/handler"
	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
	communitytrust "kun-galgame-api/internal/community/trust"
	docHandler "kun-galgame-api/internal/doc/handler"
	docRepo "kun-galgame-api/internal/doc/repository"
	docService "kun-galgame-api/internal/doc/service"
	friendHandler "kun-galgame-api/internal/friendlink/handler"
	friendRepo "kun-galgame-api/internal/friendlink/repository"
	"kun-galgame-api/internal/galgame/client"
	galgameHandler "kun-galgame-api/internal/galgame/handler"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	homeHandler "kun-galgame-api/internal/home/handler"
	homeRepo "kun-galgame-api/internal/home/repository"
	homeService "kun-galgame-api/internal/home/service"
	imageHandler "kun-galgame-api/internal/image/handler"
	imageRepo "kun-galgame-api/internal/image/repository"
	imageService "kun-galgame-api/internal/image/service"
	"kun-galgame-api/internal/infrastructure/cache"
	cronPkg "kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/infrastructure/storage"
	"kun-galgame-api/internal/infrastructure/storelink"
	msgHandler "kun-galgame-api/internal/message/handler"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	newsHandler "kun-galgame-api/internal/news/handler"
	rankingHandler "kun-galgame-api/internal/ranking/handler"
	rankingRepo "kun-galgame-api/internal/ranking/repository"
	rankingService "kun-galgame-api/internal/ranking/service"
	rssHandler "kun-galgame-api/internal/rss/handler"
	rssRepo "kun-galgame-api/internal/rss/repository"
	searchHandler "kun-galgame-api/internal/search/handler"
	searchRepo "kun-galgame-api/internal/search/repository"
	searchService "kun-galgame-api/internal/search/service"
	sectionHandler "kun-galgame-api/internal/section/handler"
	sectionRepo "kun-galgame-api/internal/section/repository"
	sectionService "kun-galgame-api/internal/section/service"
	toolsetHandler "kun-galgame-api/internal/toolset/handler"
	toolsetRepo "kun-galgame-api/internal/toolset/repository"
	toolsetService "kun-galgame-api/internal/toolset/service"
	topicHandler "kun-galgame-api/internal/topic/handler"
	topicRepo "kun-galgame-api/internal/topic/repository"
	topicService "kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/trust/enforce"
	"kun-galgame-api/internal/trust/gate"
	trustHandler "kun-galgame-api/internal/trust/handler"
	trustService "kun-galgame-api/internal/trust/service"
	updateHandler "kun-galgame-api/internal/update/handler"
	updateRepo "kun-galgame-api/internal/update/repository"
	"kun-galgame-api/internal/user/handler"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/internal/user/service"
	websiteHandler "kun-galgame-api/internal/website/handler"
	websiteRepo "kun-galgame-api/internal/website/repository"
	websiteService "kun-galgame-api/internal/website/service"
	"kun-galgame-api/pkg/artifactclient"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/dlsite"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/newsclient"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/secretbox"
	"kun-galgame-api/pkg/storeclient"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Fiber       *fiber.App
	DB          *gorm.DB
	Redis       *redis.Client
	Config      *config.Config
	OAuthClient *oauth.Client
	UserState   *repository.StateRepository
	UserClient  *userclient.Client

	OAuthHandler                   *handler.OAuthHandler
	UserHandler                    *handler.UserHandler
	UserProfileHandler             *handler.ProfileHandler
	HomeHandler                    *homeHandler.HomeHandler
	TopicHandler                   *topicHandler.TopicHandler
	TopicDraftHandler              *topicHandler.TopicDraftHandler
	ReplyHandler                   *topicHandler.ReplyHandler
	TopicCommentHandler            *topicHandler.CommentHandler
	PollHandler                    *topicHandler.PollHandler
	LotteryHandler                 *topicHandler.LotteryHandler
	MessageHandler                 *msgHandler.MessageHandler
	MessageChatHandler             *msgHandler.ChatHandler
	AdminOverviewHandler           *adminHandler.OverviewHandler
	AdminPurgeHandler              *adminHandler.PurgeHandler
	AdminTopicHandler              *adminHandler.TopicAdminHandler
	AdminRolePermissionHandler     *adminHandler.RolePermissionHandler
	AdminUserPermissionHandler     *adminHandler.UserPermissionHandler
	AdminPermissionAuditHandler    *adminHandler.PermissionAuditHandler
	RankingHandler                 *rankingHandler.RankingHandler
	SectionHandler                 *sectionHandler.SectionHandler
	DocArticleHandler              *docHandler.ArticleHandler
	DocCategoryHandler             *docHandler.CategoryHandler
	DocTagHandler                  *docHandler.TagHandler
	WebsiteHandler                 *websiteHandler.WebsiteHandler
	WebsiteCategoryHandler         *websiteHandler.CategoryHandler
	WebsiteTagHandler              *websiteHandler.TagHandler
	WebsiteTagGroupHandler         *websiteHandler.TagGroupHandler
	UpdateHandler                  *updateHandler.UpdateHandler
	FriendLinkHandler              *friendHandler.FriendLinkHandler
	TrustHandler                   *trustHandler.TrustHandler
	RSSHandler                     *rssHandler.RSSHandler
	NewsHandler                    *newsHandler.NewsHandler
	GalgameHandler                 *galgameHandler.GalgameHandler
	GalgameCollectionHandler       *galgameHandler.GalgameCollectionHandler
	GalgameCommunityCommentHandler *galgameHandler.CommunityCommentHandler
	ResourceCommentHandler         *galgameHandler.ResourceCommentHandler
	GalgameResourceHandler         *galgameHandler.ResourceHandler
	GalgameRatingHandler           *galgameHandler.RatingHandler
	GalgameQuizHandler             *galgameHandler.QuizHandler
	CreatorHandler                 *galgameHandler.CreatorHandler
	GalgameEntityHandler           *galgameHandler.EntityHandler
	GalgameCalendarHandler         *galgameHandler.CalendarHandler
	GalgameDraftsHandler           *galgameHandler.DraftsHandler
	GalgameProxyHandler            *galgameHandler.GalgameProxyHandler
	GalgameSubmissionHandler       *galgameHandler.SubmissionHandler
	GalgameClaimReviewHandler      *galgameHandler.ClaimReviewHandler
	GalgameEditHandler             *galgameHandler.EditHandler
	GalgameCoverVoteHandler        *galgameHandler.CoverVoteHandler
	GalgamePlaytimeHandler         *galgameHandler.PlaytimeHandler
	ActivityHandler                *activityHandler.ActivityHandler
	ImageHandler                   *imageHandler.ImageHandler
	SearchHandler                  *searchHandler.SearchHandler
	ToolsetHandler                 *toolsetHandler.ToolsetHandler
	ToolsetPracticalityHandler     *toolsetHandler.PracticalityHandler
	ToolsetResourceHandler         *toolsetHandler.ResourceHandler
	ToolsetUploadHandler           *toolsetHandler.UploadHandler
	CronStop                       func()
	RolePermStop                   func()
	StoreLinkStop                  func()
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
	userContentRepo := repository.NewUserContentRepository(db)
	messageRepository := msgRepo.NewMessageRepository(db)
	chatRepository := msgRepo.NewChatRepository(db)

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
	userContentService := service.NewUserContentService(userContentRepo, gc, galgameUserStatsSvc, uc, communityCli)
	messageSvc := msgService.NewMessageService(messageRepository, userStateRepo, uc)
	chatSvc := msgService.NewChatService(chatRepository, uc)
	notifier := msgService.NewNotifier(messageRepository)

	topicRepository := topicRepo.NewTopicRepository(db)
	topicListRepo := topicRepo.NewTopicListRepository(db)
	topicTaxonomyRepo := topicRepo.NewTopicTaxonomyRepository(db)
	replyRepository := topicRepo.NewReplyRepository(db)
	topicCommentRepo := topicRepo.NewCommentRepository(db)
	pollRepository := topicRepo.NewPollRepository(db)
	lotteryRepository := topicRepo.NewLotteryRepository(db)
	draftRepository := topicRepo.NewTopicDraftRepository(db)
	topicSvc := topicService.NewTopicService(topicRepository, topicListRepo, topicTaxonomyRepo, rdb, uc, userStateRepo)
	topicWriteSvc := topicService.NewTopicWriteService(topicRepository, topicTaxonomyRepo, replyRepository, userStateRepo, rdb, notifier, trustCheck, trustScan)
	replySvc := topicService.NewReplyService(replyRepository, topicCommentRepo, topicRepository, userStateRepo, uc, rdb, trustCheck, trustScan)
	commentSvc := topicService.NewCommentService(replyRepository, topicCommentRepo, userStateRepo, uc, rdb, trustCheck, trustScan)
	pollSvc := topicService.NewPollService(pollRepository, topicRepository, userStateRepo, uc, rdb, trustCheck, trustScan)
	lotteryBox, err := secretbox.New(cfg.Lottery.CodeKey)
	if err != nil {
		slog.Error("KUN_LOTTERY_CODE_KEY 无效, 兑换码托管已禁用 (抽奖其余功能不受影响)", "error", err)
	}
	if lotteryBox == nil {
		slog.Warn("KUN_LOTTERY_CODE_KEY 未设置; 抽奖将拒绝「系统托管兑换码」奖项, 而不是明文存码")
	}
	lotterySvc := topicService.NewLotteryService(
		lotteryRepository, topicRepository, userStateRepo, uc, notifier,
		lotteryBox, cfg.NextMoeAPI.ImageCDNBase, trustCheck, trustScan)
	if imageMeta != nil {
		lotterySvc.SetImageMetaResolver(imageMeta.Resolve)
	}
	lotteryDrawer := topicService.NewLotteryDrawer(lotterySvc)
	draftSvc := topicService.NewDraftService(draftRepository)

	galgameCommunityPostRepo := galgameRepo.NewCommunityPostRepository(db)
	galgameCommunityCommentSvc := galgameService.NewCommunityCommentService(communityCli, galgameCommunityPostRepo, uc, db)
	resourceCommentSvc := galgameService.NewResourceCommentService(communityCli, galgameCommunityPostRepo, uc, db)
	galgameResourceRepo := galgameRepo.NewResourceRepository(db)
	galgameResourceSvc := galgameService.NewResourceService(galgameResourceRepo, galgameLocalRepo, gc, catalogCli, uc, linkChecker, trustCheck, trustScan, storeLinks)
	galgameRatingRepo := galgameRepo.NewRatingRepository(db)
	galgameRatingSvc := galgameService.NewRatingService(galgameRatingRepo, gc, uc, trustCheck, trustScan)
	galgameQuizRepo := galgameRepo.NewQuizRepository(db)
	galgameQuizSvc := galgameService.NewQuizService(galgameQuizRepo, gc, uc, trustCheck, trustScan)
	creatorSvc := galgameService.NewCreatorService(galgameRatingRepo, galgameUserStatsSvc, uc)
	galgameInteractionRepo := galgameRepo.NewGalgameInteractionRepository(db)
	galgameListRepo := galgameRepo.NewGalgameListRepository(db)
	galgameResourceMetaRepo := galgameRepo.NewGalgameResourceMetaRepository(db)
	galgameDetailRatingRepo := galgameRepo.NewGalgameDetailRatingRepository(db)
	galgameContributorRepo := galgameRepo.NewGalgameContributorRepository(db)
	galgameEnricher := galgameService.NewGalgameEnricher(
		galgameLocalRepo, galgameResourceMetaRepo, galgameListRepo, uc,
	)
	galgameCoreSvc := galgameService.NewGalgameService(
		galgameLocalRepo, galgameInteractionRepo, galgameListRepo,
		galgameResourceMetaRepo, galgameDetailRatingRepo, galgameContributorRepo,
		galgameMergeRepo, userStateRepo, gc, uc, catalogCli, storeLinks,
	)
	galgameCollectionRepo := galgameRepo.NewGalgameCollectionRepository(db)
	galgameCollectionSvc := galgameService.NewCollectionService(galgameCollectionRepo, galgameCoreSvc, gc, uc, catalogCli, trustCheck, trustScan)
	galgameOfficialSvc := galgameService.NewOfficialService(gc, galgameCoreSvc)
	galgameEngineSvc := galgameService.NewEngineService(gc, galgameCoreSvc)
	galgameSeriesSvc := galgameService.NewSeriesService(gc, galgameEnricher, galgameCoreSvc)
	galgameTagSvc := galgameService.NewTagService(gc, galgameEnricher, galgameCoreSvc)
	galgameStaffSvc := galgameService.NewStaffService(gc, galgameEnricher)
	galgameCharacterSvc := galgameService.NewCharacterService(gc, galgameEnricher)
	galgameCalendarSvc := galgameService.NewCalendarService(gc, galgameEnricher)
	galgameDraftsSvc := galgameService.NewDraftsService(gc, galgameEnricher)
	galgameProxySvc := galgameService.NewGalgameProxyService(gc, galgameLocalRepo, uc)
	galgameSubmissionSvc := galgameService.NewSubmissionService(gc, catalogCli, galgameLocalRepo)
	galgameClaimReviewSvc := galgameService.NewClaimReviewService(gc, catalogCli)
	galgamePlaytimeSvc := galgameService.NewPlaytimeService(galgameCoreSvc, gc, catalogCli, cfg.OAuth.ClientID)
	galgameClaimSync := galgameService.NewGalgameClaimEventSync(catalogCli, galgameLocalRepo, rdb)
	galgameRevisionSync := galgameService.NewGalgameEditRevisionSync(catalogCli, gc, db, rdb)
	galgameContributorSync := galgameService.NewGalgameContributorSync(catalogCli, galgameContributorRepo, rdb)
	galgameCatalogMirror := galgameService.NewGalgameCatalogMirror(gc, galgameLocalRepo, rdb)
	galgameMergeSync := galgameService.NewGalgameMergeSync(gc, galgameMergeRepo, rdb)

	websiteRepository := websiteRepo.NewWebsiteRepository(db)
	websiteCategoryRepo := websiteRepo.NewCategoryRepository(db)
	websiteTagRepo := websiteRepo.NewTagRepository(db)
	websiteCoreSvc := websiteService.NewWebsiteService(
		websiteRepository, websiteCategoryRepo, websiteTagRepo, uc, communityCli, cfg.NextMoeAPI.ImageCDNBase,
	)
	websiteCategorySvc := websiteService.NewCategoryService(websiteCategoryRepo, websiteRepository, websiteTagRepo, cfg.NextMoeAPI.ImageCDNBase)
	websiteTagSvc := websiteService.NewTagService(websiteTagRepo, websiteRepository, websiteCategoryRepo, cfg.NextMoeAPI.ImageCDNBase)
	websiteTagGroupSvc := websiteService.NewTagGroupService(websiteRepo.NewTagGroupRepository(db))

	adminOverviewRepo := adminRepo.NewOverviewRepository(db)
	adminOverviewSvc := adminService.NewOverviewService(adminOverviewRepo)
	adminPurgeSvc := adminService.NewPurgeService(adminRepo.NewPurgeRepository(db), uc, communityCli, catalogCli)
	adminTopicSvc := adminService.NewTopicAdminService(adminRepo.NewTopicAdminRepository(db), uc)
	adminRolePermRepo := adminRepo.NewRolePermissionRepository(db)
	adminUserPermRepo := adminRepo.NewUserPermissionRepository(db)
	adminPermSync := adminService.NewPermissionOverrideSync(adminRolePermRepo, adminUserPermRepo)
	adminRolePermSvc := adminService.NewRolePermissionService(adminRolePermRepo, adminPermSync)
	adminUserPermSvc := adminService.NewUserPermissionService(adminUserPermRepo, uc, adminPermSync)
	adminPermAuditSvc := adminService.NewPermissionAuditService(adminRepo.NewPermissionAuditRepository(db), uc)

	docArticleRepo := docRepo.NewArticleRepository(db)
	docCategoryRepo := docRepo.NewCategoryRepository(db)
	docTagRepo := docRepo.NewTagRepository(db)
	docArticleSvc := docService.NewArticleService(docArticleRepo, docCategoryRepo, cfg.NextMoeAPI.ImageCDNBase)
	docCategorySvc := docService.NewCategoryService(docCategoryRepo)
	docTagSvc := docService.NewTagService(docTagRepo)

	toolsetRepository := toolsetRepo.NewToolsetRepository(db)
	toolsetResourceRepo := toolsetRepo.NewResourceRepository(db)
	toolsetPracticalityRepo := toolsetRepo.NewPracticalityRepository(db)
	toolsetPracticalitySvc := toolsetService.NewPracticalityService(toolsetPracticalityRepo)
	toolsetCommentSvc := toolsetService.NewCommentService(uc, communityCli)
	toolsetResourceSvc := toolsetService.NewResourceService(toolsetResourceRepo, toolsetRepository, fileStorageClient, artCli, uc, trustCheck, trustScan)
	toolsetUploadSvc := toolsetService.NewUploadService(artCli, rdb, db)
	toolsetCoreSvc := toolsetService.NewToolsetService(
		toolsetRepository, toolsetResourceRepo, toolsetPracticalityRepo,
		fileStorageClient, uc, toolsetPracticalitySvc, toolsetCommentSvc,
		trustCheck, trustScan,
	)

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
		UserState:                      userStateRepo,
		UserClient:                     uc,
		OAuthHandler:                   handler.NewOAuthHandler(authService, cfg.Server.Mode == "prod", communityBooster),
		UserHandler:                    handler.NewUserHandler(userService, userContentService),
		UserProfileHandler:             handler.NewProfileHandler(oauthClient, uc),
		HomeHandler:                    homeHandler.NewHomeHandler(homeService.NewHomeService(homeRepo.NewHomeRepository(db), gc, uc, rdb)),
		TopicHandler:                   topicHandler.NewTopicHandler(topicSvc, topicWriteSvc),
		TopicDraftHandler:              topicHandler.NewTopicDraftHandler(draftSvc),
		ReplyHandler:                   topicHandler.NewReplyHandler(replySvc),
		TopicCommentHandler:            topicHandler.NewCommentHandler(commentSvc),
		PollHandler:                    topicHandler.NewPollHandler(pollSvc),
		LotteryHandler:                 topicHandler.NewLotteryHandler(lotterySvc),
		MessageHandler:                 msgHandler.NewMessageHandler(messageSvc),
		MessageChatHandler:             msgHandler.NewChatHandler(chatSvc),
		AdminOverviewHandler:           adminHandler.NewOverviewHandler(adminOverviewSvc),
		AdminPurgeHandler:              adminHandler.NewPurgeHandler(adminPurgeSvc),
		AdminTopicHandler:              adminHandler.NewTopicAdminHandler(adminTopicSvc),
		AdminRolePermissionHandler:     adminHandler.NewRolePermissionHandler(adminRolePermSvc),
		AdminUserPermissionHandler:     adminHandler.NewUserPermissionHandler(adminUserPermSvc),
		AdminPermissionAuditHandler:    adminHandler.NewPermissionAuditHandler(adminPermAuditSvc),
		RankingHandler:                 rankingHandler.NewRankingHandler(rankingService.NewRankingService(rankingRepo.NewRankingRepository(db), gc, uc)),
		SectionHandler:                 sectionHandler.NewSectionHandler(sectionService.NewSectionService(sectionRepo.NewSectionRepository(db), uc)),
		DocArticleHandler:              docHandler.NewArticleHandler(docArticleSvc),
		DocCategoryHandler:             docHandler.NewCategoryHandler(docCategorySvc),
		DocTagHandler:                  docHandler.NewTagHandler(docTagSvc),
		WebsiteHandler:                 websiteHandler.NewWebsiteHandler(websiteCoreSvc),
		WebsiteCategoryHandler:         websiteHandler.NewCategoryHandler(websiteCategorySvc),
		WebsiteTagHandler:              websiteHandler.NewTagHandler(websiteTagSvc),
		WebsiteTagGroupHandler:         websiteHandler.NewTagGroupHandler(websiteTagGroupSvc),
		UpdateHandler:                  updateHandler.NewUpdateHandler(updateRepo.NewUpdateRepository(db), uc, trustCheck, trustScan),
		FriendLinkHandler:              friendHandler.NewFriendLinkHandler(friendRepo.NewFriendLinkRepository(db), cfg.NextMoeAPI.ImageCDNBase),
		TrustHandler:                   trustHandler.NewTrustHandler(trustService.NewTrustService(trustCli, cfg.Trust.Site), trustEnforce, cfg.Trust.CallbackSecret),
		RSSHandler:                     rssHandler.NewRSSHandler(rssRepo.NewRSSRepository(db), gc, uc),
		NewsHandler:                    newsHandler.NewNewsHandler(newsCli, uc),
		GalgameHandler:                 galgameHandler.NewGalgameHandler(galgameCoreSvc),
		GalgameCollectionHandler:       galgameHandler.NewGalgameCollectionHandler(galgameCollectionSvc),
		GalgameCommunityCommentHandler: galgameHandler.NewCommunityCommentHandler(galgameCommunityCommentSvc),
		ResourceCommentHandler:         galgameHandler.NewResourceCommentHandler(resourceCommentSvc),
		GalgameResourceHandler:         galgameHandler.NewResourceHandler(galgameResourceSvc),
		GalgameRatingHandler:           galgameHandler.NewRatingHandler(galgameRatingSvc),
		GalgameQuizHandler:             galgameHandler.NewQuizHandler(galgameQuizSvc),
		CreatorHandler:                 galgameHandler.NewCreatorHandler(creatorSvc),
		GalgameEntityHandler: galgameHandler.NewEntityHandler(
			galgameOfficialSvc, galgameEngineSvc, galgameSeriesSvc, galgameTagSvc,
			galgameStaffSvc, galgameCharacterSvc,
		),
		GalgameCalendarHandler:    galgameHandler.NewCalendarHandler(galgameCalendarSvc),
		GalgameDraftsHandler:      galgameHandler.NewDraftsHandler(galgameDraftsSvc),
		GalgameProxyHandler:       galgameHandler.NewGalgameProxyHandler(galgameProxySvc),
		GalgameSubmissionHandler:  galgameHandler.NewSubmissionHandler(galgameSubmissionSvc),
		GalgameClaimReviewHandler: galgameHandler.NewClaimReviewHandler(galgameClaimReviewSvc),
		GalgameEditHandler:        galgameHandler.NewEditHandler(catalogCli, gc, uc, notifier, galgameLocalRepo),
		GalgameCoverVoteHandler:   galgameHandler.NewCoverVoteHandler(catalogCli, gc),
		GalgamePlaytimeHandler:    galgameHandler.NewPlaytimeHandler(galgamePlaytimeSvc),
		ActivityHandler:           activityHandler.NewActivityHandler(activityService.NewActivityService(activityRepo.NewActivityRepository(db), gc, uc, rdb)),
		ImageHandler:              imageHandler.NewImageHandler(imageService.NewImageService(imageRepo.NewImageRepository(db), imgCli, catalogCli)),
		SearchHandler: searchHandler.NewSearchHandler(searchService.NewSearchService(
			searchRepo.NewSearchRepository(db), gc, galgameEnricher, uc,
			galgameService.NewEntitySearchService(gc, galgameTagSvc), toolsetCoreSvc,
			galgameResourceSvc,
		)),
		ToolsetHandler:             toolsetHandler.NewToolsetHandler(toolsetCoreSvc),
		ToolsetPracticalityHandler: toolsetHandler.NewPracticalityHandler(toolsetPracticalitySvc),
		ToolsetResourceHandler:     toolsetHandler.NewResourceHandler(toolsetResourceSvc),
		ToolsetUploadHandler:       toolsetHandler.NewUploadHandler(toolsetUploadSvc),
		CronStop: cronPkg.Start(db, rdb, imgCli, cronPkg.Jobs{
			GalgameClaimSync:         galgameClaimSync.Run,
			GalgameRevisionSync:      galgameRevisionSync.Run,
			GalgameContributorSync:   galgameContributorSync.Run,
			GalgameCatalogMirror:     galgameCatalogMirror.RunMirror,
			GalgameCatalogMirrorFill: galgameCatalogMirror.RunPending,
			GalgameMergeSync:         galgameMergeSync.Run,
			DlsiteCampaignRefresh:    storeLinks.RefreshCampaign,
			TopicMiniAppDeadlines:    lotteryDrawer.Run,
		}),
		StoreLinkStop: storeLinks.Start(),
	}

	if err := adminPermSync.Load(context.Background()); err != nil {
		slog.Warn("加载权限覆盖失败, 暂时沿用编译期基线", "error", err)
	}
	app.RolePermStop = adminPermSync.StartRefresher(60 * time.Second)

	fiberApp := fiber.New(fiber.Config{
		ErrorHandler:   globalErrorHandler,
		BodyLimit:      10 * 1024 * 1024,
		ReadBufferSize: 16 * 1024,
	})
	fiberApp.Use(recover.New())
	app.Fiber = fiberApp

	app.setupRoutes()
	return app
}

func globalErrorHandler(c fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return response.Error(c, appErr)
	}
	slog.Error("未处理的错误", "error", err.Error(), "path", c.Path(), "method", c.Method())
	return response.Error(c, errors.ErrInternal("服务器内部错误"))
}
