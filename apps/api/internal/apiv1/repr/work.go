package repr

type LocalizedName struct {
	Value     string `json:"value" maxLength:"512" doc:"The name in this locale. Free text; never use it as a decision input."`
	IsMachine bool   `json:"is_machine" doc:"Whether the name is a machine translation."`
}

// CatalogName is infra's name primitive (api-v2 04 §8). Render
// localized[locale] ?? display_name ?? latin; the server never picks one.
type CatalogName struct {
	DisplayName string                   `json:"display_name" maxLength:"512" doc:"The entity's own name. Never empty. Free text; never use it as a decision input."`
	Latin       *string                  `json:"latin" maxLength:"512" doc:"Romanization of the name. null when none is recorded. Free text; never use it as a decision input."`
	Localized   map[string]LocalizedName `json:"localized" doc:"Names by BCP-47 tag, sparse. Empty object when there are none, never null."`
}

func NewCatalogName(displayName, latin string, localized map[string]LocalizedName) CatalogName {
	name := CatalogName{DisplayName: displayName, Localized: localized}
	if latin != "" {
		name.Latin = &latin
	}
	if name.Localized == nil {
		name.Localized = map[string]LocalizedName{}
	}
	return name
}

// WorkRef is the smallest embedding of a work. The work id is the catalog work
// id; nothing translates it.
type WorkRef struct {
	Object string    `json:"object" enum:"work" maxLength:"4" doc:"Type discriminant. Always work."`
	ID     DecimalID `json:"id" doc:"Work id: the catalog work id, which is also the id in the web's /galgame/{id}."`
	CatalogName
	Cover  *Image `json:"cover" doc:"The portrait cover at its original size, never the 16:9 crop. null when the work has none."`
	IsNSFW bool   `json:"is_nsfw" doc:"Whether this forum displays the work as adult content: the editorial display axis (the claim's content limit), not the age rating."`
}

func NewWorkRef(workID int, name CatalogName, cover *Image, isNSFW bool) WorkRef {
	if name.Localized == nil {
		name.Localized = map[string]LocalizedName{}
	}
	return WorkRef{Object: "work", ID: ID(workID), CatalogName: name, Cover: cover, IsNSFW: isNSFW}
}
