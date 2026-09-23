package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

type Work struct {
	workrepr.WorkSummary
	Aliases                 []entityapiv1.AliasName     `json:"aliases" maxItems:"1000" doc:"Other titles, never display_name. Empty array, never null."`
	OriginalLanguage        *string                     `json:"original_language" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"The work's original language as a BCP-47 tag. null when catalog has none."`
	ContentRating           string                      `json:"content_rating" enum:"all_ages,r18" maxLength:"8" doc:"Age rating: all_ages or r18."`
	Intros                  []workrepr.CatalogIntro     `json:"intros" doc:"Descriptions in every language catalog has. Empty array, never null."`
	Links                   []workrepr.CatalogLink      `json:"links" doc:"Official site, database pages and other links, in catalog's order. Empty array, never null."`
	ExternalRefs            []WorkExternalRef           `json:"external_refs" doc:"The work's entries on other sites, such as its VNDB or Bangumi id. Empty array, never null."`
	Covers                  []WorkCover                 `json:"covers" doc:"Cover images. Empty array, never null."`
	Screenshots             []WorkScreenshot            `json:"screenshots" doc:"Screenshots. Empty array, never null."`
	Companies               []WorkCompany               `json:"companies" doc:"Credited companies. Empty array, never null."`
	Engines                 []entityapiv1.Engine        `json:"engines" doc:"Engines the work runs on. Empty array, never null."`
	Series                  []entityapiv1.SeriesSummary `json:"series" doc:"Series the work belongs to. Empty array, never null."`
	Tags                    []WorkTag                   `json:"tags" doc:"Tags. Adult tags are left out unless include_nsfw=true. Empty array, never null."`
	Credits                 []WorkCreditGroup           `json:"credits" doc:"Staff grouped by role. Empty array, never null."`
	Roster                  []WorkCharacter             `json:"roster" doc:"The work's characters, main cast first. Empty array, never null."`
	ResourceTypes           []workrepr.ResourceType     `json:"resource_types" doc:"Kinds of download resources the work has on the forum, each once, in vocabulary order. Empty array, never null."`
	FavoriteCount           int                         `json:"favorite_count" minimum:"0" doc:"Favorites from catalog popularity source=nextmoe metric=favorites."`
	IsResourcePublishBanned bool                        `json:"is_resource_publish_banned" doc:"Whether new download resources may not be published on this work. false when the forum has no row."`
	Creator                 *repr.UserRef               `json:"creator" doc:"Who created the forum page. null when there is no creator_user_id or the account is not renderable."`
	Contributors            []repr.UserRef              `json:"contributors" maxItems:"50" doc:"People who contributed to the forum page, unrenderable accounts dropped. Empty array, never null."`
	ExternalRatings         []WorkExternalRating        `json:"external_ratings" doc:"Ratings from other sites. Empty array, never null."`
	Playtimes               []WorkPlaytimeAggregate     `json:"playtimes" doc:"Playtime aggregates from other sites. Empty array, never null."`
	Dlsite                  *workrepr.DlsiteOffer       `json:"dlsite" doc:"DLsite purchase offer for the work. null when there is no DLsite workno."`
	CreatedAt               repr.DateTime               `json:"created_at" doc:"When catalog created the work."`
	UpdatedAt               repr.DateTime               `json:"updated_at" doc:"When catalog last updated the work."`
	Viewer                  *WorkViewer                 `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type WorkCover struct {
	Object    string           `json:"object" enum:"work_cover" maxLength:"10" doc:"Type discriminant. Always work_cover."`
	ID        repr.DecimalID   `json:"id" doc:"Catalog cover row id, which the cover vote path takes."`
	Image     *repr.Image      `json:"image" doc:"The cover at original size. Never null on a cover; the type is shared with images that can be absent."`
	CoverSlot string           `json:"cover_slot" enum:"main,pkgfront,dig,pkgback,pkgcontent,pkgside,pkgmed,other" maxLength:"10" doc:"Which face of the package this cover is."`
	Site      string           `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"Where the image came from, such as vndb or dlsite. An open vocabulary."`
	SortOrder int              `json:"sort_order" minimum:"0" maximum:"9999" doc:"Catalog order among covers."`
	VoteCount int              `json:"vote_count" minimum:"0" doc:"Public votes for this cover. 0 when the vote store is unread."`
	Viewer    *WorkCoverViewer `json:"viewer" doc:"The caller's vote on this cover. null for an anonymous caller."`
}

type WorkCoverViewer struct {
	HasVoted bool `json:"has_voted" doc:"Whether the caller voted for this cover."`
}

type WorkScreenshot struct {
	Object    string      `json:"object" enum:"work_screenshot" maxLength:"15" doc:"Type discriminant. Always work_screenshot."`
	Image     *repr.Image `json:"image" doc:"The screenshot at original size. Never null on a screenshot; the type is shared with images that can be absent."`
	Caption   string      `json:"caption" maxLength:"512" doc:"Caption. Empty string when none. Free text; never use it as a decision input."`
	Site      string      `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"Where the image came from. An open vocabulary."`
	SortOrder int         `json:"sort_order" minimum:"0" maximum:"9999" doc:"Catalog order among screenshots."`
}

type WorkCompany struct {
	entityapiv1.CompanySummary
	Lang             *string                `json:"lang" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"The company's own language as a BCP-47 tag. null when unrecorded."`
	Links            []workrepr.CatalogLink `json:"links" doc:"Official site, social accounts and database pages. Empty array, never null."`
	AttributionRoles []AttributionRole      `json:"attribution_roles" doc:"What the company did on this work: developer, publisher, circle or brand. Unique, never null."`
}

type AttributionRole string

func (AttributionRole) Schema(huma.Registry) *huma.Schema {
	n := 9
	return &huma.Schema{
		Type: huma.TypeString, Enum: []any{"developer", "publisher", "circle", "brand"},
		MaxLength: &n, Description: "What a company did on a work.",
	}
}

type WorkTag struct {
	entityapiv1.TagSummary
	Spoiler string `json:"spoiler" enum:"none,minor,major" maxLength:"5" doc:"How much the tag gives away."`
}

type WorkCreditGroup struct {
	RoleKey     string             `json:"role_key" maxLength:"64" pattern:"^\\S+$" doc:"Catalog's role key, such as scenario, illustration, music or voice-actor. An open vocabulary."`
	DisplayName string             `json:"display_name" maxLength:"128" doc:"The role's name as catalog records it. Free text; never use it as a decision input."`
	People      []WorkCreditPerson `json:"people" doc:"People credited in this role. Empty array, never null."`
}

type WorkCreditPerson struct {
	entityapiv1.CreditNameRef
	VoicedCharacters []VoicedCharacter `json:"voiced_characters" doc:"Character names this credit voices, as catalog wrote them. Empty array, never null."`
}

type VoicedCharacter string

func (VoicedCharacter) Schema(huma.Registry) *huma.Schema {
	n := 512
	return &huma.Schema{Type: huma.TypeString, MaxLength: &n, Description: "A voiced character's name as the credit wrote it. " + repr.FreeTextSentence}
}

type WorkCharacter struct {
	entityapiv1.CharacterRef
	Image         *repr.Image                 `json:"image" doc:"The character's portrait. null when catalog has none."`
	Figure        *repr.Image                 `json:"figure" doc:"A full-body standing picture. null when catalog has none."`
	CharacterKind string                      `json:"character_kind" enum:"main,secondary,appears" maxLength:"9" doc:"How large a part the character plays."`
	Spoiler       string                      `json:"spoiler" enum:"none,minor,major" maxLength:"5" doc:"How much naming the character gives away."`
	Identity      string                      `json:"identity" maxLength:"512" doc:"Who the character is in the story. Empty string when none. Free text; never use it as a decision input."`
	Voices        []entityapiv1.CreditNameRef `json:"voices" doc:"Who voices the character. Empty array, never null."`
}

type WorkExternalRef struct {
	Site       string `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The other site, such as vndb or bangumi. An open vocabulary."`
	ExternalID string `json:"external_id" maxLength:"64" doc:"The work's id on that site. Free text; never use it as a decision input."`
}

type WorkExternalRating struct {
	Object      string                     `json:"object" enum:"work_external_rating" maxLength:"20" doc:"Type discriminant. Always work_external_rating."`
	Site        string                     `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The rating source, such as vndb or erogamescape. An open vocabulary."`
	RatingValue float64                    `json:"rating_value" minimum:"0" doc:"The source's own score, on the source's own scale."`
	VoteCount   int                        `json:"vote_count" minimum:"0" doc:"Votes that source counted."`
	SourceRank  *int                       `json:"source_rank" minimum:"1" doc:"Rank on that source. null when unranked."`
	Buckets     []WorkExternalRatingBucket `json:"buckets" doc:"That source's histogram. Empty array, never null."`
	Stats       *WorkExternalRatingStats   `json:"stats" doc:"That source's summary statistics. null when none."`
}

type WorkExternalRatingBucket struct {
	Bucket    float64 `json:"bucket" minimum:"0" doc:"The source's own bucket label."`
	VoteCount int     `json:"vote_count" minimum:"0" doc:"Votes in this bucket."`
}

type WorkExternalRatingStats struct {
	Mean    *float64 `json:"mean" minimum:"0" doc:"Mean score. null when unrecorded."`
	Stdev   *float64 `json:"stdev" minimum:"0" doc:"Standard deviation of the scores. null when unrecorded."`
	Lowest  *float64 `json:"lowest" minimum:"0" doc:"Lowest score. null when unrecorded."`
	Highest *float64 `json:"highest" minimum:"0" doc:"Highest score. null when unrecorded."`
}

type WorkPlaytimeAggregate struct {
	Site      string `json:"site" maxLength:"64" pattern:"^[a-z0-9][a-z0-9_-]*$" doc:"The playtime source. An open vocabulary."`
	Minutes   int    `json:"minutes" minimum:"0" doc:"Aggregate minutes."`
	VoteCount int    `json:"vote_count" minimum:"0" doc:"Votes that source counted."`
}

type WorkViewer struct {
	HasLiked              bool                `json:"has_liked" doc:"Whether the caller liked this work."`
	HasFavorited          bool                `json:"has_favorited" doc:"Whether the caller holds this work in any folder."`
	Playtime              *WorkViewerPlaytime `json:"playtime" doc:"The caller's own playtime. null when they have none or the token cannot read it."`
	CanBanResourcePublish bool                `json:"can_ban_resource_publish" doc:"Whether the caller may ban publishing download resources on this work. Requests authenticated with a Bearer token never carry staff powers."`
}

type WorkViewerPlaytime struct {
	Minutes   int     `json:"minutes" minimum:"0" doc:"Minutes the caller reported. 0 when they reported none or withdrew the report."`
	PlayState *string `json:"play_state" enum:"wish,doing,done_one_route,done_main,done_all,on_hold,dropped,done" maxLength:"14" doc:"The caller's play state as catalog records it: a rating's play_status, or done for a finished game with no completion recorded, which no write accepts. null when they have none."`
}

type WorkState struct {
	Object       string         `json:"object" enum:"work_state" maxLength:"10" doc:"Type discriminant. Always work_state."`
	WorkID       repr.DecimalID `json:"work_id" doc:"Work id this state is about."`
	HasLiked     bool           `json:"has_liked" doc:"Whether the caller liked this work."`
	HasFavorited bool           `json:"has_favorited" doc:"Whether the caller holds this work in any folder."`
}

type WorkEngagement struct {
	Object    string                `json:"object" enum:"work_engagement" maxLength:"15" doc:"Type discriminant. Always work_engagement."`
	WorkID    repr.DecimalID        `json:"work_id" doc:"Work id."`
	LikeCount int                   `json:"like_count" minimum:"0" doc:"Likes on the forum page after this request."`
	Viewer    *WorkEngagementViewer `json:"viewer" doc:"The caller's like state after this request."`
}

type WorkEngagementViewer struct {
	HasLiked bool `json:"has_liked" doc:"Whether the caller liked this work."`
}
