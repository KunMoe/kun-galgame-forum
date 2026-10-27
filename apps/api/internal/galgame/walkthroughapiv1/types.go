package walkthroughapiv1

import (
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type ChartKey string

func (ChartKey) Schema(huma.Registry) *huma.Schema {
	minLen, maxLen := 1, 32
	return &huma.Schema{Type: huma.TypeString, Pattern: "^[A-Za-z0-9_]+$", MinLength: &minLen, MaxLength: &maxLen}
}

type Localized = map[string]repr.LocalizedName

type Walkthrough struct {
	Object              string                 `json:"object" enum:"walkthrough" maxLength:"11" doc:"Type discriminant. Always walkthrough."`
	ID                  repr.DecimalID         `json:"id" doc:"Walkthrough id."`
	WorkID              repr.DecimalID         `json:"work_id" doc:"The work this walkthrough plays through."`
	IsNSFW              bool                   `json:"is_nsfw" doc:"Whether the work is shown as adult content, the same field as Work.is_nsfw. A client that hides adult content gates the walkthrough on it."`
	HasInGameRouteHints bool                   `json:"has_in_game_route_hints" doc:"The game shows route or choice hints itself. A reader may prefer them, so a client says so before showing the routes."`
	FollowedRelease     *WalkthroughRelease    `json:"followed_release" doc:"The release this walkthrough follows, such as an official or fan translation whose choice texts differ from the original. null when it follows no particular release."`
	SourceSite          string                 `json:"source_site" enum:"seiya,sagaoz_foolmaker,nextmoe" maxLength:"32" doc:"Where the choice graph came from: a walkthrough site it was derived from, or nextmoe for one written in the NextMoe editor center."`
	SourcePageURL       *string                `json:"source_page_url" format:"uri" maxLength:"2048" doc:"The original walkthrough page. Only its choice graph is carried here; link to it for attribution. null for a nextmoe walkthrough, which has no page to link to."`
	SourceAuthor        *string                `json:"source_author" maxLength:"256" doc:"Author the source page credits. null if none. Free text; never use it as a decision input."`
	SourceUpdatedDate   *repr.CalendarDate     `json:"source_updated_date" doc:"Date the source page says it was last updated. null if unstated."`
	TextLang            string                 `json:"text_lang" pattern:"^[a-z]{2,3}(-[A-Za-z0-9]+)*$" maxLength:"32" doc:"BCP-47 language of the chart's text fields: ja for an imported chart, ja or zh-Hans for a nextmoe one. Translations sit in localized maps beside them."`
	ChartCharacters     []WalkthroughCharacter `json:"chart_characters" doc:"Characters the chart routes to, in chart order. Empty array, never null."`
	EndingCount         int                    `json:"ending_count" minimum:"0" doc:"Endings in the chart."`
	SaveCount           int                    `json:"save_count" minimum:"0" doc:"Save points in the chart."`
	SegmentCount        int                    `json:"segment_count" minimum:"0" doc:"Segments in the chart."`
	UpdatedAt           repr.DateTime          `json:"updated_at" doc:"Last time this walkthrough changed."`
	Chart               *WalkthroughChart      `json:"chart,omitempty" doc:"The choice graph. Present on GET /walkthroughs/{walkthrough_id} only."`
}

type WalkthroughRelease struct {
	Object       string         `json:"object" enum:"release" maxLength:"7" doc:"Type discriminant. Always release."`
	ID           repr.DecimalID `json:"id" doc:"Catalog release id, a release of the walkthrough's work."`
	ReleaseTitle *string        `json:"release_title" maxLength:"512" doc:"The release's title. null if it has none. Free text; never use it as a decision input."`
	ReleaseKind  string         `json:"release_kind" enum:"default,digital,physical,trial,patch" maxLength:"8" doc:"patch includes fan translation patches."`
	Lang         *string        `json:"lang" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"The release's main language as a BCP-47 tag. null when unrecorded."`
	CompanyNames []CompanyName  `json:"company_names" doc:"Companies and groups credited on the release, such as a patch's translation group. Empty array, never null."`
}

type CompanyName string

func (CompanyName) Schema(huma.Registry) *huma.Schema {
	n := 256
	return &huma.Schema{Type: huma.TypeString, MaxLength: &n,
		Description: "A company or group's display name as catalog records it. " + repr.FreeTextSentence}
}

type WalkthroughCharacter struct {
	Key              ChartKey                 `json:"key" doc:"Chart-local key that sections and endings refer to."`
	SourceName       string                   `json:"source_name" maxLength:"512" doc:"The name as the source page writes it. Free text; never use it as a decision input."`
	CatalogCharacter *WalkthroughCharacterRef `json:"catalog_character" doc:"The catalog character this name is pinned to. null if the pin is gone."`
}

type WalkthroughCharacterRef struct {
	Object      string           `json:"object" enum:"character" maxLength:"9" doc:"Type discriminant. Always character."`
	ID          repr.DecimalID   `json:"id" doc:"Character id."`
	CatalogName repr.CatalogName `json:"catalog_name" doc:"The character's catalog name."`
}

type WalkthroughChart struct {
	RouteSections []WalkthroughSection `json:"route_sections" doc:"Headings of the source page that group segments, usually one per route. Empty array, never null."`
	EntryPoints   []WalkthroughEntry   `json:"entry_points" doc:"Places a playthrough starts from. Empty array, never null."`
	SavePoints    []WalkthroughSave    `json:"save_points" doc:"Save points the chart creates and resumes from. Empty array, never null."`
	ChartEndings  []WalkthroughEnding  `json:"chart_endings" doc:"Endings, one per occurrence in the source. Empty array, never null."`
	ChartSegments []WalkthroughSegment `json:"chart_segments" doc:"Runs of steps from one node to the next, in source order. Empty array, never null."`
}

type WalkthroughSection struct {
	Key           ChartKey   `json:"key" doc:"Chart-local key segments refer to."`
	Title         string     `json:"title" maxLength:"512" doc:"The heading as the source writes it. Free text; never use it as a decision input."`
	Localized     Localized  `json:"localized" doc:"Translations of title by BCP-47 tag. Empty object when none."`
	CharacterKeys []ChartKey `json:"character_keys" doc:"chart_characters[].key this heading is about. Empty array, never null."`
}

type WalkthroughEntry struct {
	Key                 ChartKey  `json:"key" doc:"Chart-local key nodes refer to."`
	Label               string    `json:"label" maxLength:"256" doc:"The entry as the source names it, such as START or RESTART. Free text; never use it as a decision input."`
	UnlockedByEndingKey *ChartKey `json:"unlocked_by_ending_key" doc:"chart_endings[].key whose completion unlocks this entry. null if open from the start."`
}

type WalkthroughSave struct {
	Key   ChartKey `json:"key" doc:"Chart-local key save steps and nodes refer to."`
	Label string   `json:"label" maxLength:"256" doc:"The save point as the source labels it. Free text; never use it as a decision input."`
}

type WalkthroughEnding struct {
	Key           ChartKey   `json:"key" doc:"Chart-local key ending nodes refer to."`
	EndingName    string     `json:"ending_name" maxLength:"512" doc:"The ending's name as the source writes it. Free text; never use it as a decision input."`
	Localized     Localized  `json:"localized" doc:"Translations of ending_name by BCP-47 tag. Empty object when none."`
	EndingType    string     `json:"ending_type" enum:"heroine,harem,true,normal,bad,after,other" maxLength:"8" doc:"heroine concludes one character's route, harem several; after is an after-story."`
	CharacterKeys []ChartKey `json:"character_keys" doc:"chart_characters[].key whose route this ending concludes. Empty array, never null."`
}

type WalkthroughSegment struct {
	Key                ChartKey          `json:"key" doc:"Chart-local segment key."`
	SectionKey         *ChartKey         `json:"section_key" doc:"route_sections[].key this segment is listed under. null if none."`
	FromNode           WalkthroughNode   `json:"from_node" doc:"Where the segment starts."`
	Condition          *string           `json:"condition" maxLength:"1024" doc:"Precondition the source states. null if none. Free text; never use it as a decision input."`
	ConditionLocalized Localized         `json:"condition_localized" doc:"Translations of condition by BCP-47 tag. Empty object when none."`
	Steps              []WalkthroughStep `json:"steps" doc:"Steps in play order. Empty array, never null."`
	ToNode             WalkthroughNode   `json:"to_node" doc:"Where the segment ends."`
}

type WalkthroughNode struct {
	NodeType string    `json:"node_type" enum:"entry,save,ending,collect,open" maxLength:"7" doc:"collect is a stop after collecting CG or scenes; open is a segment that simply stops."`
	NodeKey  *ChartKey `json:"node_key" doc:"entry_points, save_points or chart_endings key for those node types. null for collect and open."`
}

type WalkthroughStep struct {
	StepType          string                   `json:"step_type" enum:"choice,action,save" maxLength:"6" doc:"choice picks step_text; action is a non-choice step; save creates save_key here."`
	StepText          *string                  `json:"step_text" maxLength:"1024" doc:"The choice or action as the source writes it. null for a save step. Free text; never use it as a decision input."`
	Localized         Localized                `json:"localized" doc:"Translations of step_text by BCP-47 tag. Empty object when none."`
	Alternatives      []WalkthroughAlternative `json:"alternatives" doc:"Other choices kept for collecting CG differences. Empty array, never null."`
	IsAchievementOnly bool                     `json:"is_achievement_only" doc:"The source marks this step as needed for an achievement only."`
	SaveKey           *ChartKey                `json:"save_key" doc:"save_points[].key this save step creates. null for other steps."`
}

type WalkthroughAlternative struct {
	Text      string    `json:"text" maxLength:"1024" doc:"The alternative choice as the source writes it. Free text; never use it as a decision input."`
	Localized Localized `json:"localized" doc:"Translations of text by BCP-47 tag. Empty object when none."`
}
