package entityapiv1

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

type AliasName string

func (AliasName) Schema(huma.Registry) *huma.Schema {
	n := 512
	return &huma.Schema{Type: huma.TypeString, MaxLength: &n, Description: "One alias. " + repr.FreeTextSentence}
}

func aliases(values []string) []AliasName {
	out := make([]AliasName, 0, len(values))
	for _, v := range values {
		if len(v) <= 512 {
			out = append(out, AliasName(v))
		}
	}
	return out
}

type TagSummary struct {
	Object string         `json:"object" enum:"tag" maxLength:"3" doc:"Type discriminant. Always tag."`
	ID     repr.DecimalID `json:"id" doc:"Tag id: the catalog tag id, which is also the id in the web's /galgame/tag/{id}."`
	repr.CatalogName
	TagKind          string `json:"tag_kind" enum:"content,meta" maxLength:"7" doc:"What the tag describes: content is the story and characters, meta is the game as a product."`
	IsSexual         bool   `json:"is_sexual" doc:"Whether the tag is adult content. Such tags are left out unless include_nsfw=true."`
	CatalogWorkCount int    `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
}

type Tag struct {
	Object string         `json:"object" enum:"tag" maxLength:"3" doc:"Type discriminant. Always tag."`
	ID     repr.DecimalID `json:"id" doc:"Tag id: the catalog tag id, which is also the id in the web's /galgame/tag/{id}."`
	repr.CatalogName
	TagKind          string                  `json:"tag_kind" enum:"content,meta" maxLength:"7" doc:"What the tag describes: content is the story and characters, meta is the game as a product."`
	IsSexual         bool                    `json:"is_sexual" doc:"Whether the tag is adult content. Such a tag is NOT_FOUND unless include_nsfw=true."`
	CatalogWorkCount int                     `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
	IsHidden         bool                    `json:"is_hidden" doc:"Whether catalog keeps the tag out of sight. A hidden tag is never listed or searched, but its own page still answers."`
	Intros           []workrepr.CatalogIntro `json:"intros" doc:"Descriptions in every language catalog has, unordered. Empty array, never null."`
}

type CompanySummary struct {
	Object string         `json:"object" enum:"company" maxLength:"7" doc:"Type discriminant. Always company."`
	ID     repr.DecimalID `json:"id" doc:"Company id: the catalog company id, which is also the id in the web's /galgame/official/{id}."`
	repr.CatalogName
	CompanyKind      string      `json:"company_kind" enum:"game_brand,bunko,publisher,anime_studio,doujin_circle,group" maxLength:"13" doc:"What sort of company it is."`
	Logo             *repr.Image `json:"logo" doc:"The company's logo. null when catalog has none."`
	Aliases          []AliasName `json:"aliases" maxItems:"1000" doc:"Other names it goes by, never its display_name. Empty array, never null."`
	CatalogWorkCount int         `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
}

type Company struct {
	Object string         `json:"object" enum:"company" maxLength:"7" doc:"Type discriminant. Always company."`
	ID     repr.DecimalID `json:"id" doc:"Company id: the catalog company id, which is also the id in the web's /galgame/official/{id}."`
	repr.CatalogName
	CompanyKind      string                  `json:"company_kind" enum:"game_brand,bunko,publisher,anime_studio,doujin_circle,group" maxLength:"13" doc:"What sort of company it is."`
	Logo             *repr.Image             `json:"logo" doc:"The company's logo. null when catalog has none."`
	Aliases          []AliasName             `json:"aliases" maxItems:"1000" doc:"Other names it goes by, never its display_name. Empty array, never null."`
	CatalogWorkCount int                     `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
	Lang             *string                 `json:"lang" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"The company's own language as a BCP-47 tag. null when unrecorded."`
	Links            []workrepr.CatalogLink  `json:"links" doc:"Official site, social accounts and database pages, in catalog's order. Empty array, never null."`
	Intros           []workrepr.CatalogIntro `json:"intros" doc:"Descriptions in every language catalog has, unordered. Empty array, never null."`
}

type CompanyWork struct {
	Object      string               `json:"object" enum:"company_work" maxLength:"12" doc:"Type discriminant. Always company_work."`
	WorkSummary workrepr.WorkSummary `json:"work_summary" doc:"The work."`
	ViaCompany  *workrepr.CompanyRef `json:"via_company" doc:"The imprint of this company the work is credited to, when it counts here only through that imprint. null for the company's own works."`
}

type CompanyGraph struct {
	Object    string             `json:"object" enum:"company_graph" maxLength:"13" doc:"Type discriminant. Always company_graph."`
	CompanyID repr.DecimalID     `json:"company_id" doc:"The company the graph was drawn around."`
	Nodes     []CompanyGraphNode `json:"nodes" doc:"Every company in the family, this one included. Empty array, never null."`
	Edges     []CompanyGraphEdge `json:"edges" doc:"Relations between the nodes, each stored in both directions. Empty array, never null."`
}

type CompanyGraphNode struct {
	Object string         `json:"object" enum:"company" maxLength:"7" doc:"Type discriminant. Always company."`
	ID     repr.DecimalID `json:"id" doc:"Company id."`
	repr.CatalogName
	Logo             *repr.Image `json:"logo" doc:"The company's logo. null when catalog has none."`
	CatalogWorkCount int         `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
}

type CompanyGraphEdge struct {
	FromCompanyID repr.DecimalID `json:"from_company_id" doc:"The company the relation is stated from."`
	ToCompanyID   repr.DecimalID `json:"to_company_id" doc:"The company it points at."`
	Relation      string         `json:"relation" enum:"parent,subsidiary,imprint,imprint_of,succeeded_by,formerly,spawned,origin" maxLength:"12" doc:"What to_company is to from_company. Every relation also appears reversed: parent with subsidiary, imprint with imprint_of, succeeded_by with formerly, spawned with origin."`
}

type WikiCompanyRedirect struct {
	Object        string         `json:"object" enum:"wiki_company_redirect" maxLength:"21" doc:"Type discriminant. Always wiki_company_redirect."`
	WikiCompanyID repr.DecimalID `json:"wiki_company_id" doc:"The company id the retired galgame wiki used."`
	CompanyID     repr.DecimalID `json:"company_id" doc:"The catalog company it became."`
}

type Engine struct {
	Object string         `json:"object" enum:"engine" maxLength:"6" doc:"Type discriminant. Always engine."`
	ID     repr.DecimalID `json:"id" doc:"Engine id: the catalog engine id, which is also the id in the web's /galgame/engine/{id}."`
	repr.CatalogName
	Aliases          []AliasName `json:"aliases" maxItems:"1000" doc:"Other names it goes by, never its display_name. Empty array, never null."`
	Description      string      `json:"description" maxLength:"65535" doc:"Catalog's note on the engine, in whatever language it was written. Empty string if none. Free text; never use it as a decision input."`
	CatalogWorkCount int         `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
}

type SeriesSampleWork struct {
	repr.WorkRef
	Banner *repr.Image `json:"banner" doc:"The landscape art at its original size, never the 16:9 crop. null when the work has none; clients fall back to cover."`
}

type SeriesSummary struct {
	Object string         `json:"object" enum:"series" maxLength:"6" doc:"Type discriminant. Always series."`
	ID     repr.DecimalID `json:"id" doc:"Series id: the catalog series id, which is also the id in the web's /galgame/series/{id}."`
	repr.CatalogName
	HasNSFWWorks     *bool              `json:"has_nsfw_works" doc:"Whether any work of the series is adult content. null when catalog does not know; a browse without include_nsfw leaves out true and null alike."`
	CatalogWorkCount int                `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
	ListedWorkCount  int                `json:"listed_work_count" minimum:"0" doc:"Works of the series the forum lists: ones with a resource and a published page."`
	SampleWorks      []SeriesSampleWork `json:"sample_works" maxItems:"5" doc:"Up to five listed works of the series, earliest release first. Empty array, never null."`
}

type Series struct {
	Object string         `json:"object" enum:"series" maxLength:"6" doc:"Type discriminant. Always series."`
	ID     repr.DecimalID `json:"id" doc:"Series id: the catalog series id, which is also the id in the web's /galgame/series/{id}."`
	repr.CatalogName
	HasNSFWWorks     *bool                   `json:"has_nsfw_works" doc:"Whether any work of the series is adult content. null when catalog does not know."`
	CatalogWorkCount int                     `json:"catalog_work_count" minimum:"0" doc:"Works catalog files under it, NSFW ones included. How many a reader can page through is the total of its works collection."`
	ListedWorkCount  int                     `json:"listed_work_count" minimum:"0" doc:"Works of the series the forum lists: ones with a resource and a published page."`
	SampleWorks      []SeriesSampleWork      `json:"sample_works" maxItems:"5" doc:"Up to five listed works of the series, earliest release first. Empty array, never null."`
	Intros           []workrepr.CatalogIntro `json:"intros" doc:"Descriptions in every language catalog has, unordered. Empty array, never null."`
}

type CreditNameRef struct {
	Object string         `json:"object" enum:"credit_name" maxLength:"11" doc:"Type discriminant. Always credit_name."`
	ID     repr.DecimalID `json:"id" doc:"Credit name id: the catalog credit name id, which is also the id in the web's /galgame/staff/{id}."`
	repr.CatalogName
	Lang *string `json:"lang" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"The name's own language as a BCP-47 tag. null when unrecorded."`
}

type CreditName struct {
	Object string         `json:"object" enum:"credit_name" maxLength:"11" doc:"Type discriminant. Always credit_name."`
	ID     repr.DecimalID `json:"id" doc:"Credit name id: the catalog credit name id, which is also the id in the web's /galgame/staff/{id}."`
	repr.CatalogName
	Lang       *string                 `json:"lang" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"The name's own language as a BCP-47 tag. null when unrecorded."`
	Photo      *repr.Image             `json:"photo" doc:"A photo of the person. null when catalog has none."`
	Gender     *string                 `json:"gender" enum:"male,female" maxLength:"6" doc:"null when unrecorded."`
	BirthYear  *int                    `json:"birth_year" minimum:"1" maximum:"9999" doc:"null when unrecorded. Birthdays are often known only in part."`
	BirthMonth *int                    `json:"birth_month" minimum:"1" maximum:"12" doc:"null when unrecorded."`
	BirthDay   *int                    `json:"birth_day" minimum:"1" maximum:"31" doc:"null when unrecorded."`
	Intros     []workrepr.CatalogIntro `json:"intros" doc:"Profiles in every language catalog has, unordered. Empty array, never null."`
	Links      []workrepr.CatalogLink  `json:"links" doc:"Database pages and sites about the person. Empty array, never null."`
	Siblings   []CreditNameRef         `json:"siblings" doc:"Other names the same person is credited under. Empty array, never null."`
}

type Credit struct {
	Object      string               `json:"object" enum:"credit" maxLength:"6" doc:"Type discriminant. Always credit."`
	WorkSummary workrepr.WorkSummary `json:"work_summary" doc:"The credited work."`
	CreditRoles []CreditRole         `json:"credit_roles" minItems:"1" doc:"What the name did on the work, most prominent first."`
	Characters  []CreditCharacter    `json:"characters" doc:"For a voice credit, the characters voiced. Empty array, never null."`
}

type CreditRole struct {
	RoleKey     string `json:"role_key" maxLength:"64" pattern:"^\\S+$" doc:"Catalog's role key, such as scenario, illustration, music or voice-actor. An open vocabulary."`
	DisplayName string `json:"display_name" maxLength:"128" doc:"The role's name as catalog records it. Free text; never use it as a decision input."`
}

type CreditCharacter struct {
	CharacterID *repr.DecimalID `json:"character_id" doc:"The character's id when catalog links one. null when the credit names the character only as text."`
	DisplayName string          `json:"display_name" maxLength:"512" doc:"The character's name as the credit writes it, in the source language. Free text; never use it as a decision input."`
}

type CharacterRef struct {
	Object string         `json:"object" enum:"character" maxLength:"9" doc:"Type discriminant. Always character."`
	ID     repr.DecimalID `json:"id" doc:"Character id: the catalog character id, which is also the id in the web's /galgame/character/{id}."`
	repr.CatalogName
}

type CharacterSummary struct {
	CharacterRef
	Image            *repr.Image `json:"image" doc:"The character's portrait. null when catalog has none or its picture could not be read."`
	CatalogWorkCount int         `json:"catalog_work_count" minimum:"0" doc:"Works the character appears in, under include_nsfw."`
	MatchedTraits    []TraitRef  `json:"matched_traits" maxItems:"100" doc:"The character's own traits that made it match trait_ids, descendants of a named trait included. Empty array without trait_ids, never null."`
}

type Character struct {
	Object string         `json:"object" enum:"character" maxLength:"9" doc:"Type discriminant. Always character."`
	ID     repr.DecimalID `json:"id" doc:"Character id: the catalog character id, which is also the id in the web's /galgame/character/{id}."`
	repr.CatalogName
	Lang   *string                 `json:"lang" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"The character's own language as a BCP-47 tag. null when unrecorded."`
	Image  *repr.Image             `json:"image" doc:"The character's portrait. null when catalog has none."`
	Figure *repr.Image             `json:"figure" doc:"A full-body standing picture. null when catalog has none."`
	Intros []workrepr.CatalogIntro `json:"intros" doc:"Profiles in every language catalog has, unordered. Empty array, never null."`
	Traits []CharacterTrait        `json:"traits" doc:"Traits in catalog's order. Adult traits are left out unless include_nsfw=true. Empty array, never null."`
	Links  []workrepr.CatalogLink  `json:"links" doc:"Database pages about the character. Empty array, never null."`
}

type CharacterTrait struct {
	Object string         `json:"object" enum:"trait" maxLength:"5" doc:"Type discriminant. Always trait."`
	ID     repr.DecimalID `json:"id" doc:"Catalog trait id."`
	repr.CatalogName
	TraitGroup repr.CatalogName `json:"trait_group" doc:"The group the trait sits in, such as hair or personality."`
	Spoiler    string           `json:"spoiler" enum:"none,minor,major" maxLength:"5" doc:"How much the trait gives away."`
	IsLie      bool             `json:"is_lie" doc:"Whether the trait is one the story later reveals to be false."`
	IsSexual   bool             `json:"is_sexual" doc:"Whether the trait is adult content."`
}

type Appearance struct {
	Object      string               `json:"object" enum:"appearance" maxLength:"10" doc:"Type discriminant. Always appearance."`
	WorkSummary workrepr.WorkSummary `json:"work_summary" doc:"The work the character appears in."`
	Voices      []CreditNameRef      `json:"voices" doc:"Who voices the character in that work. Empty array, never null."`
}

type TraitRef struct {
	Object string         `json:"object" enum:"trait" maxLength:"5" doc:"Type discriminant. Always trait."`
	ID     repr.DecimalID `json:"id" doc:"Trait id: the catalog trait id, which is also the id in the web's /galgame/trait/{id}."`
	repr.CatalogName
	TraitGroup   repr.CatalogName `json:"trait_group" doc:"The root group the trait sits in, such as hair or personality. A root trait is its own group. Two traits can share a name across groups, such as the same act under Engages in and Subject of; show the group beside the name there."`
	TraitGroupID repr.DecimalID   `json:"trait_group_id" doc:"The root group's trait id."`
}

type TraitSummary struct {
	Object string         `json:"object" enum:"trait" maxLength:"5" doc:"Type discriminant. Always trait."`
	ID     repr.DecimalID `json:"id" doc:"Trait id: the catalog trait id, which is also the id in the web's /galgame/trait/{id}."`
	repr.CatalogName
	TraitGroup     repr.CatalogName `json:"trait_group" doc:"The root group the trait sits in, such as hair or personality. A root trait is its own group."`
	TraitGroupID   repr.DecimalID   `json:"trait_group_id" doc:"The root group's trait id."`
	Parents        []TraitRef       `json:"parents" maxItems:"100" doc:"The traits directly above this one; a few traits sit under more than one. Empty array for a root trait, never null."`
	IsSexual       bool             `json:"is_sexual" doc:"Whether the trait is adult content. Such traits are left out unless include_nsfw=true."`
	IsSearchable   bool             `json:"is_searchable" doc:"Whether the trait is specific enough to filter characters by. Grouping traits such as hair colour are not, though they still have a page."`
	CharacterCount int              `json:"character_count" minimum:"0" doc:"Characters carrying the trait or one of its descendants without a spoiler, under include_nsfw: the total of /characters?trait_ids= this trait with the same include_nsfw. Refreshed nightly."`
	ChildCount     int              `json:"child_count" minimum:"0" doc:"Traits directly below this one that the reader may see."`
}

type Trait struct {
	Object string         `json:"object" enum:"trait" maxLength:"5" doc:"Type discriminant. Always trait."`
	ID     repr.DecimalID `json:"id" doc:"Trait id: the catalog trait id, which is also the id in the web's /galgame/trait/{id}."`
	repr.CatalogName
	TraitGroup     repr.CatalogName `json:"trait_group" doc:"The root group the trait sits in, such as hair or personality. A root trait is its own group."`
	TraitGroupID   repr.DecimalID   `json:"trait_group_id" doc:"The root group's trait id."`
	Parents        []TraitRef       `json:"parents" maxItems:"100" doc:"The traits directly above this one; a few traits sit under more than one. Empty array for a root trait, never null."`
	IsSexual       bool             `json:"is_sexual" doc:"Whether the trait is adult content. Such a trait is NOT_FOUND unless include_nsfw=true."`
	IsSearchable   bool             `json:"is_searchable" doc:"Whether the trait is specific enough to filter characters by. Grouping traits such as hair colour are not, though they still have a page."`
	CharacterCount int              `json:"character_count" minimum:"0" doc:"Characters carrying the trait or one of its descendants without a spoiler, under include_nsfw: the total of /characters?trait_ids= this trait with the same include_nsfw. Refreshed nightly."`
	ChildCount     int              `json:"child_count" minimum:"0" doc:"Traits directly below this one that the reader may see."`
	Aliases        []AliasName      `json:"aliases" maxItems:"1000" doc:"Other names the trait goes by. Empty array, never null."`
	Description    string           `json:"description" maxLength:"65535" doc:"Catalog's note on the trait, plain text, usually English. Empty string if none. Free text; never use it as a decision input."`
	Subtraits      []TraitSummary   `json:"subtraits" doc:"Traits up to two levels below this one: the direct ones first, then theirs, most characters first under each parent. An item's parents say where it hangs. Empty array, never null."`
}
