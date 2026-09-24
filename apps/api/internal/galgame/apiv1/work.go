package apiv1

import (
	"context"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/middleware"
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
	body, p := s.assembleWork(ctx, workID, d, in.IncludeNSFW, v1.User(ctx))
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

	out.Covers = coversOf(d, s.cdn, s.coverTallies(ctx, workID))
	out.Screenshots = screenshotsOf(d, s.cdn)

	if user != nil {
		viewer, p := s.workViewer(user, workID)
		if p != nil {
			return Work{}, p
		}
		out.Viewer = viewer
	}
	return out, nil
}

// Nothing here reads catalog with the reader's token. The detail GET runs in
// SSR for every tab a browser restores, and on 2026-09-24 one reader's
// restored tabs spent catalog's per-uid bucket (100 a minute across every app)
// at four calls each. The reader's own state is on GET /me/works.
func (s *Service) workViewer(user *middleware.UserInfo, workID int) (*WorkViewer, *problem.Problem) {
	v := &WorkViewer{CanBanResourcePublish: user.Can(perm.GalgameBanResourcePublish)}
	if s.store != nil && s.store.Ready() {
		liked, err := s.store.HasLiked(user.ID, workID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		v.HasLiked = liked
	}
	return v, nil
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
