package apiv1

import (
	"encoding/json"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/model"

	"github.com/danielgtaylor/huma/v2"
)

type TopicDraftPoll struct {
	Title            string               `json:"title" maxLength:"100" doc:"Poll question as the author left it. Free text; never use it as a decision input."`
	Description      string               `json:"description" maxLength:"500" doc:"Longer explanation. Empty string when there is none. Free text; never use it as a decision input."`
	ChoiceType       PollChoiceType       `json:"choice_type" doc:"Whether a voter picks exactly one option or several."`
	MinChoice        int                  `json:"min_choice" minimum:"1" maximum:"20" doc:"Fewest options a vote may hold."`
	MaxChoice        int                  `json:"max_choice" minimum:"1" maximum:"20" doc:"Most options a vote may hold."`
	ClosesAt         *repr.DateTime       `json:"closes_at" doc:"When the poll would stop accepting votes. null means it never closes. Not checked against the clock: a draft can outlive its own deadline, and createPoll refuses it then."`
	ResultVisibility PollResultVisibility `json:"result_visibility" doc:"Who may see the tallies."`
	IsAnonymous      bool                 `json:"is_anonymous" doc:"Whether who voted for what is hidden."`
	CanChangeVote    bool                 `json:"can_change_vote" doc:"Whether a voter may replace or retract their vote."`
	Options          []PollOptionCreate   `json:"options" maxItems:"20" doc:"The options, in display order."`
}

type TopicDraftLotteryPrize struct {
	Title            string      `json:"title" maxLength:"100" doc:"Prize name as the author left it. Free text; never use it as a decision input."`
	Description      string      `json:"description" maxLength:"500" doc:"Prize description. Empty string when there is none. Free text; never use it as a decision input."`
	ImageHashes      []ImageHash `json:"image_hashes" maxItems:"9" uniqueItems:"true" doc:"Prize images by image-service hash, in display order."`
	AdultImageHashes []ImageHash `json:"adult_image_hashes" maxItems:"9" uniqueItems:"true" doc:"Which of image_hashes the author marks adult."`
	Delivery         string      `json:"delivery" enum:"code,offline,point" maxLength:"7" doc:"How the prize reaches a winner. A code prize holds no codes in a draft."`
	PointMode        *string     `json:"point_mode" enum:"fixed,split,random" maxLength:"6" doc:"For a point prize, as in createLottery. null for other prizes."`
	PointAmount      *int        `json:"point_amount" minimum:"1" maximum:"100000" doc:"For a point prize, as in createLottery. null for other prizes."`
	SlotCount        int         `json:"slot_count" minimum:"0" maximum:"500" doc:"How many winners this prize has."`
}

type topicDraftLotteryPrizeFields TopicDraftLotteryPrize

// prizes is inline on Lottery and LotteryCreate, and G8 compares a $ref as part
// of the shape.
func (TopicDraftLotteryPrize) Schema(r huma.Registry) *huma.Schema {
	return inlineObjectSchema[topicDraftLotteryPrizeFields](r)
}

type TopicDraftLottery struct {
	Title             string                   `json:"title" maxLength:"100" doc:"Lottery title as the author left it. Free text; never use it as a decision input."`
	Description       string                   `json:"description" maxLength:"1000" doc:"Lottery description. Empty string when there is none. Free text; never use it as a decision input."`
	EntryMode         string                   `json:"entry_mode" enum:"signup,reply,floor" maxLength:"6" doc:"Who is in the draw, as in createLottery."`
	FloorRule         *string                  `json:"floor_rule" maxLength:"200" doc:"For a floor lottery: the winning floors, as in createLottery. null otherwise. Free text; never use it as a decision input."`
	DrawMode          string                   `json:"draw_mode" enum:"deadline,manual,threshold" maxLength:"9" doc:"When the draw happens, as in createLottery."`
	DrawThreshold     *int                     `json:"draw_threshold" minimum:"1" maximum:"100000" doc:"For a threshold draw: the entry count that triggers it. null otherwise."`
	ClosesAt          *repr.DateTime           `json:"closes_at" doc:"When the lottery would stop taking entries. null when it has none. Not checked against the clock: a draft can outlive its own deadline, and createLottery refuses it then."`
	MinAccountAgeDays int                      `json:"min_account_age_days" minimum:"0" maximum:"3650" doc:"Account age an entrant needs, in days."`
	MinMoemoepoint    int                      `json:"min_moemoepoint" minimum:"0" maximum:"1000000" doc:"Moemoepoint an entrant needs."`
	IsEntryListPublic bool                     `json:"is_entry_list_public" doc:"Whether everyone may list the entries."`
	Prizes            []TopicDraftLotteryPrize `json:"prizes" maxItems:"10" doc:"The prizes, in display order. Redemption codes are never part of a draft."`
}

type draftMiniApps struct {
	polls       json.RawMessage
	lotteries   json.RawMessage
	prizeImages []string
}

func encodeDraftMiniApps(polls []TopicDraftPoll, lotteries []TopicDraftLottery) (draftMiniApps, error) {
	if polls == nil {
		polls = []TopicDraftPoll{}
	}
	if lotteries == nil {
		lotteries = []TopicDraftLottery{}
	}
	images := []string{}
	seen := map[ImageHash]bool{}
	for _, lottery := range lotteries {
		for _, prize := range lottery.Prizes {
			for _, hash := range prize.ImageHashes {
				if !seen[hash] {
					seen[hash] = true
					images = append(images, string(hash))
				}
			}
		}
	}
	p, err := json.Marshal(polls)
	if err != nil {
		return draftMiniApps{}, err
	}
	l, err := json.Marshal(lotteries)
	if err != nil {
		return draftMiniApps{}, err
	}
	return draftMiniApps{polls: p, lotteries: l, prizeImages: images}, nil
}

func decodeDraftMiniApps(d *model.TopicDraft) ([]TopicDraftPoll, []TopicDraftLottery, error) {
	polls := []TopicDraftPoll{}
	lotteries := []TopicDraftLottery{}
	if len(d.Polls) > 0 {
		if err := json.Unmarshal(d.Polls, &polls); err != nil {
			return nil, nil, err
		}
	}
	if len(d.Lotteries) > 0 {
		if err := json.Unmarshal(d.Lotteries, &lotteries); err != nil {
			return nil, nil, err
		}
	}
	return polls, lotteries, nil
}
