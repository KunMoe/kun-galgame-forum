package client

func (d *CatalogWorkDetail) IntroRows() []CatalogIntro {
	if len(d.Intros) > 0 {
		return d.Intros
	}
	return d.Intro
}

func (d *CatalogWorkDetail) ListItem() CatalogWorkListItem {
	if d == nil {
		return CatalogWorkListItem{}
	}
	return CatalogWorkListItem{
		ID:            d.ID,
		DisplayName:   d.DisplayName,
		ContentRating: d.ContentRating,
		ContentLimit:  d.ContentLimit,
		OLang:         d.OLang,
		ReleaseDate:   d.ReleaseDate,
		Claim:         d.Claim,
		Updated:       d.Updated,
		Localized:     d.Localized,
		Latin:         d.Latin,
		Intros:        catIntros(d.IntroRows()),
		Labels:        d.Labels,
		Ratings:       d.Ratings,
		Covers:        d.CoverSlots,
		CoverSlots:    d.CoverSlots,
		Refs:          d.Refs,
	}
}
