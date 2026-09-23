package apiv1

import (
	"context"
	"errors"
	"log/slog"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type getWorkInput struct {
	WorkID      string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog work id, which is also the forum galgame page id."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult tags are included. Default false: adult tags are stripped."`
}

type getWorkOutput struct {
	Body Work
}

func (s *Service) getWork(ctx context.Context, in *getWorkInput) (*getWorkOutput, error) {
	if s == nil || s.works == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	detail := s.detailCatalog()
	if detail == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	d, found, movedTo, appErr := detail.CatalogWorkDetail(ctx, workID)
	if appErr != nil {
		return nil, catalogUnavailable(appErr)
	}
	if movedTo != 0 {
		return nil, mergedWork(movedTo)
	}
	if !found {
		return nil, notFound()
	}
	if s.store != nil && s.store.Ready() {
		if err := s.store.IncrementView(workID); err != nil {
			return nil, problem.Internal(err)
		}
	}
	body, p := s.assembleWork(ctx, workID, d, in.IncludeNSFW, v1.User(ctx), accessToken(ctx))
	if p != nil {
		return nil, p
	}
	return &getWorkOutput{Body: body}, nil
}

func (s *Service) assembleWork(
	ctx context.Context,
	workID int,
	d *client.CatalogWorkDetail,
	includeNSFW bool,
	user *middleware.UserInfo,
	token string,
) (Work, *problem.Problem) {
	item := d.ListItem()
	summary := workrepr.WorkSummary{
		WorkRef:           workrepr.Ref(ctx, &item, s.cdn),
		Banner:            workrepr.Banner(&item, s.cdn),
		Maker:             workrepr.Maker(&item),
		ResourcePlatforms: []workrepr.ResourcePlatform{},
		ResourceLanguages: []workrepr.ResourceLanguage{},
	}
	date, precision := workrepr.Release(item.ReleaseDate)
	summary.ReleaseDate, summary.ReleaseDatePrecision = date, precision
	if s.hydrator != nil {
		hydrated, p := s.hydrator.FromRows(ctx, []client.CatalogWorkListItem{item})
		if p != nil {
			return Work{}, p
		}
		if len(hydrated) == 1 {
			summary = hydrated[0]
		}
	}

	name := workrepr.Name(d.DisplayName, d.Latin, client.LocalizedValues(d.Localized))
	out := Work{
		WorkSummary:      summary,
		Aliases:          aliasesOf(d, name.DisplayName),
		OriginalLanguage: originalLanguageOf(d.OLang),
		ContentRating:    contentRatingOf(d.ContentRating),
		Intros:           workrepr.Intros(d.IntroRows()),
		Links:            linksOf(d),
		ExternalRefs:     externalRefsOf(d),
		Engines:          s.enginesOf(ctx, d),
		Series:           s.seriesOf(ctx, d),
		Tags:             tagsOf(d, includeNSFW),
		Credits:          creditsOf(d),
		Roster:           charactersOf(ctx, d, s.cdn),
		ResourceTypes:    []workrepr.ResourceType{},
		FavoriteCount:    favoriteCountOf(d),
		Contributors:     []repr.UserRef{},
		ExternalRatings:  externalRatingsOf(d),
		Playtimes:        playtimesOf(d),
		CreatedAt:        catalogTime(d.Created),
		UpdatedAt:        catalogTime(d.Updated),
	}
	if s.storeLinks != nil {
		brief := client.CatalogItemToBrief(ctx, &item)
		out.Dlsite = workrepr.DlsiteOf(s.storeLinks.Resolve(workID, brief.DlsiteWorkno()))
	}
	out.Companies = s.companiesOf(ctx, d)

	var creatorID int
	if s.store != nil && s.store.Ready() {
		local, ok, err := s.store.FindLocal(workID)
		if err != nil {
			return Work{}, problem.Internal(err)
		}
		if ok {
			out.ViewCount = max(local.View, 0)
			out.LikeCount = max(local.LikeCount, 0)
			out.IsPublished = local.Published
			out.IsResourcePublishBanned = local.ResourcePublishBanned
			if local.CreatorUserID != nil {
				creatorID = *local.CreatorUserID
			}
		}
		types, err := s.store.ResourceTypes(workID)
		if err != nil {
			return Work{}, problem.Internal(err)
		}
		out.ResourceTypes = resourceTypesOf(types)
	}

	contribIDs := []int{}
	if s.store != nil && s.store.Ready() {
		ids, err := s.store.ContributorIDs(workID, contributorMax)
		if err != nil {
			return Work{}, problem.Internal(err)
		}
		contribIDs = ids
	}
	userIDs := append([]int{}, contribIDs...)
	if creatorID > 0 {
		userIDs = append(userIDs, creatorID)
	}
	var users map[int]userclient.User
	if len(userIDs) > 0 {
		var p *problem.Problem
		users, p = s.lookupUsers(ctx, userIDs)
		if p != nil {
			return Work{}, p
		}
	}
	if creatorID > 0 && renderable(users, creatorID) {
		ref := repr.NewUserRef(s.cdn, users[creatorID])
		out.Creator = &ref
	}
	for _, id := range contribIDs {
		if !renderable(users, id) {
			continue
		}
		out.Contributors = append(out.Contributors, repr.NewUserRef(s.cdn, users[id]))
	}

	var coverViewer *WorkCoverViewer
	if user != nil {
		coverViewer = &WorkCoverViewer{}
	}
	tallies := s.coverTallies(ctx, workID, token)
	out.Covers = coversOf(d, s.cdn, tallies, coverViewer)
	out.Screenshots = screenshotsOf(d, s.cdn)

	if user != nil {
		viewer, p := s.workViewer(ctx, workID, user, token, creatorID)
		if p != nil {
			return Work{}, p
		}
		out.Viewer = viewer
	}
	return out, nil
}

func (s *Service) coverTallies(ctx context.Context, workID int, token string) []catalogclient.CoverTally {
	if s.catalog == nil {
		return nil
	}
	var (
		tallies []catalogclient.CoverTally
		err     error
	)
	if token != "" {
		tallies, err = s.catalog.WorkCoversUser(ctx, token, int64(workID))
		// The user lane answers 401 for every reader, not the 403 SCOPE_REQUIRED
		// this fallback was written for, so signed-in readers saw no tallies at all
		// while signed-out readers saw them. Degrading loses only the `voted` flag.
		if errors.Is(err, catalogclient.ErrInsufficientScope) || errors.Is(err, catalogclient.ErrUnauthorized) {
			tallies, err = s.catalog.WorkCoverVotes(ctx, int64(workID))
		}
	} else {
		tallies, err = s.catalog.WorkCoverVotes(ctx, int64(workID))
	}
	if err != nil {
		slog.Warn("galgame detail: cover vote tallies unavailable", "work_id", workID, "error", err)
		return nil
	}
	return tallies
}

func (s *Service) workViewer(ctx context.Context, workID int, user *middleware.UserInfo, token string, _ int) (*WorkViewer, *problem.Problem) {
	v := &WorkViewer{}
	if s.store != nil && s.store.Ready() {
		liked, err := s.store.HasLiked(user.ID, workID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		v.HasLiked = liked
	}
	v.HasFavorited = s.favorited(ctx, token, workID)
	if token != "" && s.catalog != nil {
		got, err := s.catalog.MyPlaytime(ctx, token, int64(workID))
		// A token minted before playtime joined the authorize scope is the
		// ordinary case, not a fault. It used to log nothing at all, and that is
		// how the 2026-09-08 folder-scope outage stayed invisible on the sibling
		// call sites for an hour — so it is counted rather than swallowed.
		if err != nil {
			if errors.Is(err, catalogclient.ErrInsufficientScope) {
				service.WarnPlaytimeUnreadable(workID)
			} else {
				slog.Warn("galgame detail: own playtime unavailable", "work_id", workID, "error", err)
			}
		} else {
			var ws *catalogclient.WorkStateRecord
			if rec, werr := s.catalog.MyWorkState(ctx, token, int64(workID)); werr != nil {
				slog.Warn("galgame detail: own work-state unavailable", "work_id", workID, "error", werr)
			} else {
				ws = rec
			}
			v.Playtime = viewerPlaytime(got, ws)
		}
	}
	v.CanBanResourcePublish = user.Can(perm.GalgameBanResourcePublish)
	return v, nil
}

func (s *Service) favorited(ctx context.Context, token string, workID int) bool {
	if token == "" || s.catalog == nil {
		return false
	}
	folders, err := s.catalog.MyFoldersContaining(ctx, token, int64(workID))
	if err != nil {
		if errors.Is(err, catalogclient.ErrInsufficientScope) {
			service.WarnFavoriteUnreadable(workID)
		} else {
			slog.Warn("galgame: favourite state unreadable", "work_id", workID, "err", err)
		}
		return false
	}
	return len(folders) > 0
}

func workNamePreview(ctx context.Context, it *client.CatalogWorkListItem) string {
	brief := client.CatalogItemToBrief(ctx, it)
	name := client.BriefName(&brief)
	runes := []rune(name)
	if len(runes) <= constants.TextPreviewLength {
		return name
	}
	return string(runes[:constants.TextPreviewLength])
}
