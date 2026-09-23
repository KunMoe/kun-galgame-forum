package entityapiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const (
	notFoundDesc   = "NOT_FOUND when no such entity is visible."
	mergedDesc     = "ENTITY_MERGED when the entity was merged into another; current_id names it."
	unavailableMsg = "SERVICE_UNAVAILABLE when catalog cannot be reached."
)

func op(id, path, summary, desc string, tags []string, errs map[int]string) huma.Operation {
	return v1.Public(huma.Operation{
		OperationID: id,
		Method:      http.MethodGet,
		Path:        path,
		Summary:     summary,
		Description: desc,
		Tags:        tags,
		Responses:   problemResponses(errs),
	})
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		tags := []string{"galgame-tags"}
		huma.Register(api, op("listTags", "/tags", "List galgame tags",
			"Without q: every visible tag catalog files works under, most works first, ties broken by ascending id. "+
				"Hidden tags never appear; adult tags only with include_nsfw=true. A page-number collection.",
			tags, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 503: unavailableMsg}), s.listTags)
		huma.Register(api, op("getTag", "/tags/{tag_id}", "Get a galgame tag",
			"An adult tag is NOT_FOUND unless include_nsfw=true, the same answer as a tag that does not exist.",
			tags, map[int]string{404: notFoundDesc, 503: unavailableMsg}), s.getTag)
		huma.Register(api, op("listTagWorks", "/tags/{tag_id}/works", "List a tag's works", worksDescription,
			tags, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 404: notFoundDesc, 503: unavailableMsg}), s.listTagWorks)
		huma.Register(api, op("listTaggedWorks", "/tagged-works", "List works carrying every given tag",
			"Catalog's own search population, newest release first; unlike a tag's works collection it takes no forum filter or sort. "+
				"A page-number collection.",
			tags, map[int]string{400: "INVALID_PARAMETER when tag_ids is empty, longer than 10, or not ids, or page × limit is too deep.", 503: unavailableMsg}), s.listTaggedWorks)

		companies := []string{"galgame-companies"}
		huma.Register(api, op("listCompanies", "/companies", "List galgame companies",
			"Without q: every company catalog files works under, most works first, ties broken by ascending id. A page-number collection.",
			companies, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 503: unavailableMsg}), s.listCompanies)
		huma.Register(api, op("getCompany", "/companies/{company_id}", "Get a galgame company", "",
			companies, map[int]string{404: notFoundDesc + " " + mergedDesc, 503: unavailableMsg}), s.getCompany)
		huma.Register(api, op("listCompanyWorks", "/companies/{company_id}/works", "List a company's works",
			worksDescription+" A company's works include its imprints' works; via narrows to one or the other.",
			companies, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 404: notFoundDesc + " " + mergedDesc, 503: unavailableMsg}), s.listCompanyWorks)
		huma.Register(api, op("getCompanyGraph", "/companies/{company_id}/graph", "Get a company's family graph",
			"Parent companies, subsidiaries, imprints, renames and spin-offs around the company.",
			companies, map[int]string{404: notFoundDesc, 503: unavailableMsg}), s.getCompanyGraph)
		huma.Register(api, op("getWikiCompanyRedirect", "/wiki-company-redirects/{wiki_company_id}", "Resolve a retired wiki company id",
			"The retired galgame wiki numbered companies on its own; old links carry those numbers.",
			companies, map[int]string{404: "NOT_FOUND when the number maps to no company.", 503: unavailableMsg}), s.getWikiCompanyRedirect)

		engines := []string{"galgame-engines"}
		huma.Register(api, op("listEngines", "/engines", "List galgame engines",
			"Every engine catalog records, most works first, ties broken by ascending id. A page-number collection.",
			engines, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 503: unavailableMsg}), s.listEngines)
		huma.Register(api, op("getEngine", "/engines/{engine_id}", "Get a galgame engine", "",
			engines, map[int]string{404: notFoundDesc, 503: unavailableMsg}), s.getEngine)
		huma.Register(api, op("listEngineWorks", "/engines/{engine_id}/works", "List an engine's works", worksDescription,
			engines, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 404: notFoundDesc, 503: unavailableMsg}), s.listEngineWorks)

		series := []string{"galgame-series"}
		huma.Register(api, op("listSeries", "/series", "List galgame series",
			"Without q: series with at least one work the forum lists, most listed works first, ties broken by ascending id. A page-number collection.",
			series, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 503: unavailableMsg}), s.listSeries)
		huma.Register(api, op("getSeries", "/series/{series_id}", "Get a galgame series", "",
			series, map[int]string{404: notFoundDesc, 503: unavailableMsg}), s.getSeries)
		huma.Register(api, op("listSeriesWorks", "/series/{series_id}/works", "List a series' works", worksDescription,
			series, map[int]string{400: "INVALID_PARAMETER when page × limit is too deep.", 404: notFoundDesc, 503: unavailableMsg}), s.listSeriesWorks)

		credits := []string{"galgame-credit-names"}
		huma.Register(api, op("listCreditNames", "/credit-names", "Search credit names",
			"The names staff and voice actors are credited under. A page-number collection.",
			credits, map[int]string{400: "INVALID_PARAMETER when q is only whitespace or page × limit exceeds 100.", 503: unavailableMsg}), s.listCreditNames)
		huma.Register(api, op("getCreditName", "/credit-names/{credit_name_id}", "Get a credit name", "",
			credits, map[int]string{404: notFoundDesc + " " + mergedDesc, 503: unavailableMsg}), s.getCreditName)
		huma.Register(api, op("listCreditNameCredits", "/credit-names/{credit_name_id}/credits", "List a credit name's credits",
			"Works credited to the name, in catalog's order. A cursor collection.",
			credits, map[int]string{400: "INVALID_CURSOR when the cursor is broken or was made with another include_nsfw.", 404: notFoundDesc + " " + mergedDesc, 503: unavailableMsg}), s.listCreditNameCredits)

		characters := []string{"galgame-characters"}
		huma.Register(api, op("listCharacters", "/characters", "Search characters", "A page-number collection.",
			characters, map[int]string{400: "INVALID_PARAMETER when q is only whitespace or page × limit exceeds 100.", 503: unavailableMsg}), s.listCharacters)
		huma.Register(api, op("getCharacter", "/characters/{character_id}", "Get a character", "",
			characters, map[int]string{404: notFoundDesc + " " + mergedDesc, 503: unavailableMsg}), s.getCharacter)
		huma.Register(api, op("listCharacterAppearances", "/characters/{character_id}/appearances", "List a character's appearances",
			"Works the character appears in, with who voices it there, in catalog's order. A cursor collection.",
			characters, map[int]string{400: "INVALID_CURSOR when the cursor is broken or was made with another include_nsfw.", 404: notFoundDesc + " " + mergedDesc, 503: unavailableMsg}), s.listCharacterAppearances)
	}
}
