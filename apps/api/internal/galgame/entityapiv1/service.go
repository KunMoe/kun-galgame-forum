package entityapiv1

import (
	"context"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/errors"

	"gorm.io/gorm"
)

// Catalog is the slice of the catalog client these faces read. Everything here
// is a public catalog read; nothing writes.
type Catalog interface {
	workrepr.Rows
	CatalogTaxonomyList(ctx context.Context, entity string, q url.Values) (*client.CatalogTaxonomyPage, *errors.AppError)
	CatalogEntitySearch(ctx context.Context, searchType, keywords string, page, limit int) ([]client.CatalogEntityHit, int64, *errors.AppError)
	CatalogSexualTagIDs(ctx context.Context, ids []int) map[int]bool
	CatalogTag(ctx context.Context, id string) (*client.CatalogTagDetail, bool, *errors.AppError)
	CatalogLabel(ctx context.Context, id string) (*client.CatalogLabelDetail, bool, int64, *errors.AppError)
	CatalogEngine(ctx context.Context, id string) (*client.CatalogEngineDetail, bool, *errors.AppError)
	CatalogSeries(ctx context.Context, id string) (*client.CatalogSeriesDetail, bool, *errors.AppError)
	CatalogLabelRelationGraph(ctx context.Context, id string) (*client.CatalogLabelRelationGraph, bool, *errors.AppError)
	LookupWikiLabel(ctx context.Context, wikiID int) (int64, bool, *errors.AppError)
	CatalogNameDetail(ctx context.Context, id int64, limit, offset int) (*client.CatalogName, bool, int64, *errors.AppError)
	CatalogCharacterDetail(ctx context.Context, id int64, limit, offset int, withWorks bool) (*client.CatalogCharacter, bool, int64, *errors.AppError)
	CatalogMemberWorkIDs(ctx context.Context, filter url.Values, isSFW bool, pageCap int) ([]int, *errors.AppError)
	CatalogLabelRollupMembers(ctx context.Context, labelID, sort string, isSFW bool, pageCap int) ([]client.CatalogRollupMember, *errors.AppError)
	CatalogRowsByCatalogIDs(ctx context.Context, ids []int64, isSFW bool) (map[int64]client.CatalogWorkListItem, *errors.AppError)
	CatalogWorksSearch(ctx context.Context, q url.Values) (*client.CatalogWorksPage, *errors.AppError)
	CatalogTraitVocabulary(ctx context.Context) ([]client.CatalogTrait, *errors.AppError)
	CatalogCharacterList(ctx context.Context, in client.CatalogCharacterQuery) (*client.CatalogCharacterPage, *errors.AppError)
}

type Service struct {
	catalog Catalog
	works   *workrepr.Hydrator
	lists   *repository.GalgameListRepository
	cdn     string

	tags        index[TagSummary]
	companies   index[CompanySummary]
	engines     index[Engine]
	series      index[seriesRow]
	seriesCards index[SeriesSummary]
	traits      index[traitVocab]
}

func New(catalog Catalog, db *gorm.DB, cdn string) *Service {
	return &Service{
		catalog: catalog,
		works:   workrepr.NewHydrator(catalog, db, cdn),
		lists:   repository.NewGalgameListRepository(db),
		cdn:     cdn,
	}
}
