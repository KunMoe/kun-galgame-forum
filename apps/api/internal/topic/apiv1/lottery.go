package apiv1

import (
	"encoding/json"

	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

const (
	lotteryTitleLimit            = 100
	lotteryDescriptionLimit      = 1000
	lotteryPrizeTitleLimit       = 100
	lotteryPrizeDescriptionLimit = 500
	lotteryFloorRuleLimit        = 200
	lotteryCodeLimit             = 200
	lotteryPointBudgetLimit      = 100000
)

type LotteryPrizeImage struct {
	Hash             string      `json:"hash" minLength:"64" maxLength:"64" pattern:"^[0-9a-f]{64}$" doc:"Image-service content hash. Present even when image is withheld, so an edit form can write the whole gallery back."`
	Image            *repr.Image `json:"image" doc:"The image. null when it is marked adult or graded explicit and the request did not ask for include_nsfw=true: the URL is the gate, so nothing renders and the bytes are never fetched."`
	IsMarkedAdult    bool        `json:"is_marked_adult" doc:"Whether the lottery's author marked this image adult."`
	IsGradedExplicit bool        `json:"is_graded_explicit" doc:"Whether the image service graded it explicit. It adds to what the author marked and the author cannot take it off."`
}

type LotteryPrize struct {
	Object      string              `json:"object" enum:"lottery_prize" maxLength:"13" doc:"Type discriminant. Always lottery_prize."`
	ID          repr.DecimalID      `json:"id" doc:"Prize id. JSON string of a decimal integer."`
	Title       string              `json:"title" maxLength:"100" doc:"Prize name as stored. Free text; never use it as a decision input."`
	Description string              `json:"description" maxLength:"500" doc:"Prize description as stored. Empty string when there is none. Free text; never use it as a decision input."`
	Images      []LotteryPrizeImage `json:"images" maxItems:"9" doc:"Prize images in the author's order. Empty array if none."`
	Delivery    string              `json:"delivery" enum:"code,offline,point" maxLength:"7" doc:"How the prize reaches a winner: a redemption code held by the site, handed over by the author off the site, or moemoepoint paid at the draw."`
	PointMode   *string             `json:"point_mode" enum:"fixed,split,random" maxLength:"6" doc:"For a point prize: fixed pays point_amount to every winner, split shares point_amount evenly, random shares it by the revealed seed. null for other prizes."`
	PointAmount *int                `json:"point_amount" minimum:"1" maximum:"100000" doc:"For a point prize: each winner's amount when point_mode is fixed, the whole pool otherwise. null for other prizes."`
	PointBudget *int                `json:"point_budget" minimum:"1" maximum:"100000" doc:"For a point prize: what it pays out when every slot is filled, which the lottery's author paid for it. null for other prizes."`
	SlotCount   int                 `json:"slot_count" minimum:"0" maximum:"500" doc:"How many winners this prize has."`
	CodeCount   *int                `json:"code_count" minimum:"0" maximum:"500" doc:"For a code prize: how many codes the site holds for it. The codes themselves are never in any read. null for other prizes."`
}

type lotteryPrizeFields LotteryPrize

func (LotteryPrize) Schema(r huma.Registry) *huma.Schema {
	return inlineObjectSchema[lotteryPrizeFields](r)
}

type LotteryWinner struct {
	Object         string         `json:"object" enum:"lottery_winner" maxLength:"14" doc:"Type discriminant. Always lottery_winner."`
	ID             repr.DecimalID `json:"id" doc:"Winner id. JSON string of a decimal integer; the path parameter winner_id of updateLotteryWinner."`
	PrizeID        repr.DecimalID `json:"prize_id" doc:"Id of the prize won."`
	Winner         repr.UserRef   `json:"winner" doc:"The user who won."`
	WinningFloor   *int           `json:"winning_floor" minimum:"1" doc:"For a floor lottery: the floor that won. null otherwise."`
	RankKey        *string        `json:"rank_key" maxLength:"64" pattern:"^[0-9a-f]{64}$" doc:"HMAC-SHA256(seed, \"<lottery_id>:<user_id>\") in hex. Lowest keys win, so anyone holding the revealed seed can check the order. null for a floor lottery."`
	Fulfillment    string         `json:"fulfillment" enum:"pending,shipped,received,forfeited" maxLength:"9" doc:"Delivery progress. received and forfeited are final."`
	PointAwarded   int            `json:"point_awarded" minimum:"0" doc:"Moemoepoint paid to this winner. 0 for other prizes."`
	WonAt          repr.DateTime  `json:"won_at" doc:"When the lottery was drawn."`
	ClaimExpiresAt *repr.DateTime `json:"claim_expires_at" doc:"For a code prize: when the code is forfeited if it has not been revealed. null for other prizes."`
}

type LotteryViewer struct {
	HasEntered           bool            `json:"has_entered" doc:"Whether the caller has entered."`
	CanEnter             bool            `json:"can_enter" doc:"Whether enterLottery would be accepted now. It is decided by the same check as the write."`
	EnterBlockedReason   *string         `json:"enter_blocked_reason" enum:"not_open,past_closes_at,no_signup,own_lottery,reply_required,moemoepoint_below_minimum,account_too_new" maxLength:"25" doc:"Why can_enter is false for a caller who has not entered. null when can_enter is true or the caller has entered. The thresholds are min_moemoepoint and min_account_age_days on the lottery."`
	CanEdit              bool            `json:"can_edit" doc:"Whether the caller may change the lottery: its author, or staff holding the manage permission, while it is open. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete            bool            `json:"can_delete" doc:"Whether the caller may delete the lottery: its author before the draw, staff holding the manage permission unless it is being drawn."`
	CanDraw              bool            `json:"can_draw" doc:"Whether the caller may draw the lottery now."`
	CanCancel            bool            `json:"can_cancel" doc:"Whether the caller may cancel the lottery now."`
	CanViewEntries       bool            `json:"can_view_entries" doc:"Whether listLotteryEntries would be accepted: is_entry_list_public, or the author, or staff holding the view permission."`
	CanManageFulfillment bool            `json:"can_manage_fulfillment" doc:"Whether the caller may move any winner's fulfillment: the author or staff holding the manage permission, after the draw."`
	WinnerID             *repr.DecimalID `json:"winner_id" doc:"Id of the caller's own entry in winners when they won. null otherwise."`
	CanRevealCode        bool            `json:"can_reveal_code" doc:"Whether the caller won a code prize that has not been forfeited, so revealLotteryCode would return it."`
}

type Lottery struct {
	Object            string          `json:"object" enum:"lottery" maxLength:"7" doc:"Type discriminant. Always lottery."`
	ID                repr.DecimalID  `json:"id" doc:"Lottery id. JSON string of a decimal integer."`
	TopicID           repr.DecimalID  `json:"topic_id" doc:"Id of the topic the lottery belongs to."`
	Author            repr.UserRef    `json:"author" doc:"The user who created the lottery and paid for its point prizes."`
	Title             string          `json:"title" maxLength:"100" doc:"Lottery title as stored. Free text; never use it as a decision input."`
	Description       string          `json:"description" maxLength:"1000" doc:"Lottery description as stored. Empty string when there is none. Free text; never use it as a decision input."`
	EntryMode         string          `json:"entry_mode" enum:"signup,reply,floor" maxLength:"6" doc:"signup takes anyone who enters, reply takes only those who have replied to the topic, floor takes no entries and awards the replies on the floors named by floor_rule."`
	FloorRule         *string         `json:"floor_rule" maxLength:"200" doc:"For a floor lottery: the winning floors, either comma-separated (8,18,28) or every:N. null otherwise. Free text; never use it as a decision input."`
	DrawMode          string          `json:"draw_mode" enum:"deadline,manual,threshold" maxLength:"9" doc:"deadline draws at closes_at, manual when the author draws it, threshold when entry_count reaches draw_threshold."`
	DrawThreshold     *int            `json:"draw_threshold" minimum:"1" maximum:"100000" doc:"For a threshold draw: the entry count that triggers it. null otherwise."`
	ClosesAt          *repr.DateTime  `json:"closes_at" doc:"When the lottery stops taking entries, and for a deadline draw when it is drawn. null when it has no deadline."`
	MinAccountAgeDays int             `json:"min_account_age_days" minimum:"0" maximum:"3650" doc:"Account age an entrant needs, in days. 0 means no requirement."`
	MinMoemoepoint    int             `json:"min_moemoepoint" minimum:"0" maximum:"1000000" doc:"Moemoepoint an entrant needs. 0 means no requirement."`
	IsEntryListPublic bool            `json:"is_entry_list_public" doc:"Whether everyone may list the entries, or only the author and staff."`
	State             string          `json:"state" enum:"open,drawing,drawn,cancelled" maxLength:"9" doc:"Lifecycle state. drawing lasts while the winners are being fixed."`
	SeedHash          *string         `json:"seed_hash" maxLength:"64" pattern:"^[0-9a-f]{64}$" doc:"SHA-256 of seed, published when the lottery was created so the author cannot re-roll after seeing the entrants. null for a floor lottery, which has no randomness to commit to."`
	Seed              *string         `json:"seed" maxLength:"64" pattern:"^[0-9a-f]{64}$" doc:"The secret behind seed_hash. null until the lottery is drawn, and always null for a floor lottery."`
	EntryCount        int             `json:"entry_count" minimum:"0" doc:"Number of entries."`
	SlotCount         int             `json:"slot_count" minimum:"0" doc:"Number of winners across every prize."`
	DrawnAt           *repr.DateTime  `json:"drawn_at" doc:"When the lottery was drawn. null before that."`
	Prizes            []LotteryPrize  `json:"prizes" maxItems:"10" doc:"The prizes, in the author's order."`
	Winners           []LotteryWinner `json:"winners" maxItems:"5000" doc:"Winners by prize, then by entry. Empty array before the draw. Banned winners are left out."`
	CreatedAt         repr.DateTime   `json:"created_at" doc:"Creation time."`
	UpdatedAt         repr.DateTime   `json:"updated_at" doc:"Time of the latest change."`
	Viewer            *LotteryViewer  `json:"viewer" doc:"The caller's own state on this lottery. null for an anonymous caller."`
}

type LotteryEntry struct {
	Object    string         `json:"object" enum:"lottery_entry" maxLength:"13" doc:"Type discriminant. Always lottery_entry."`
	ID        repr.DecimalID `json:"id" doc:"Entry id. JSON string of a decimal integer."`
	Entrant   repr.UserRef   `json:"entrant" doc:"The user who entered."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"When they entered. For a floor lottery, when the draw recorded the winner."`
}

type LotteryCodeReveal struct {
	Object         string         `json:"object" enum:"lottery_code_reveal" maxLength:"19" doc:"Type discriminant. Always lottery_code_reveal."`
	LotteryID      repr.DecimalID `json:"lottery_id" doc:"Id of the lottery the code was won in."`
	RedemptionCode string         `json:"redemption_code" maxLength:"200" doc:"The redemption code in plain text. Never keep it anywhere a page load could read it back. Free text; never use it as a decision input."`
}

type LotteryPrizeInput struct {
	Title            string      `json:"title" minLength:"1" maxLength:"100" doc:"Prize name. Only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	Description      *string     `json:"description,omitempty" maxLength:"500" doc:"Prize description. Absent means none. Free text; never use it as a decision input."`
	ImageHashes      []ImageHash `json:"image_hashes" required:"false" maxItems:"9" uniqueItems:"true" doc:"Prize images by image-service hash, in display order."`
	AdultImageHashes []ImageHash `json:"adult_image_hashes" required:"false" maxItems:"9" uniqueItems:"true" doc:"Which of image_hashes the author marks adult. A hash not in image_hashes is refused as INCONSISTENT_WITH."`
	Delivery         string      `json:"delivery" enum:"code,offline,point" maxLength:"7" doc:"How the prize reaches a winner."`
	PointMode        *string     `json:"point_mode" required:"false" enum:"fixed,split,random" maxLength:"6" doc:"Required for a point prize, refused for others. A floor lottery cannot use random: it has no seed, so anyone could compute the shares in advance."`
	PointAmount      *int        `json:"point_amount" required:"false" minimum:"1" maximum:"100000" doc:"Required for a point prize, refused for others. A pool (split or random) needs at least one point per slot."`
	SlotCount        int         `json:"slot_count" minimum:"0" maximum:"500" doc:"How many winners this prize has, 1 to 500. 0 is refused as OUT_OF_RANGE."`
	Codes            []string    `json:"codes" required:"false" maxItems:"500" doc:"For a code prize: exactly slot_count redemption codes, each 1 to 200 characters after trimming. Refused for other prizes. They are sealed at rest and never returned by any read."`
}

type lotteryPrizeInputFields LotteryPrizeInput

func (LotteryPrizeInput) Schema(r huma.Registry) *huma.Schema {
	s := inlineObjectSchema[lotteryPrizeInputFields](r)
	if codes := s.Properties["codes"]; codes != nil && codes.Items != nil {
		min, max := 1, lotteryCodeLimit
		codes.Items.MinLength = &min
		codes.Items.MaxLength = &max
		codes.Items.Description = "A redemption code. Free text; never use it as a decision input."
	}
	return s
}

// A PATCH has to tell "leave closes_at alone" from "clear it", and a
// *repr.DateTime decodes an absent field and an explicit null to the same nil.
type LotteryClosesAt struct {
	Present bool
	Value   *repr.DateTime
}

func (d *LotteryClosesAt) UnmarshalJSON(data []byte) error {
	d.Present = true
	if string(data) == "null" {
		d.Value = nil
		return nil
	}
	var v repr.DateTime
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	d.Value = &v
	return nil
}

func (LotteryClosesAt) Schema(r huma.Registry) *huma.Schema {
	s := repr.DateTime("").Schema(r)
	s.Nullable = true
	s.Description = "When the lottery stops taking entries, as RFC 3339 in UTC with second precision; a deadline draw happens then. null means no deadline."
	return s
}

type LotteryCreate struct {
	Title             string              `json:"title" minLength:"1" maxLength:"100" doc:"Lottery title. Only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	Description       *string             `json:"description,omitempty" maxLength:"1000" doc:"Lottery description. Absent means none. Free text; never use it as a decision input."`
	EntryMode         string              `json:"entry_mode" enum:"signup,reply,floor" maxLength:"6" doc:"Who is in the draw."`
	FloorRule         *string             `json:"floor_rule" required:"false" maxLength:"200" doc:"Required for a floor lottery, ignored otherwise: 8,18,28 or every:N, naming exactly as many floors as there are slots. Free text; never use it as a decision input."`
	DrawMode          string              `json:"draw_mode" enum:"deadline,manual,threshold" maxLength:"9" doc:"When the draw happens. A floor lottery cannot use threshold."`
	DrawThreshold     *int                `json:"draw_threshold" required:"false" minimum:"1" maximum:"100000" doc:"Required for a threshold draw, ignored otherwise. It must be at least the total number of slots."`
	ClosesAt          LotteryClosesAt     `json:"closes_at" required:"false" doc:"Required for a deadline draw, and must be in the future."`
	MinAccountAgeDays *int                `json:"min_account_age_days,omitempty" minimum:"0" maximum:"3650" doc:"Account age an entrant needs, in days. Absent means 0."`
	MinMoemoepoint    *int                `json:"min_moemoepoint,omitempty" minimum:"0" maximum:"1000000" doc:"Moemoepoint an entrant needs. Absent means 0."`
	IsEntryListPublic *bool               `json:"is_entry_list_public,omitempty" doc:"Whether everyone may list the entries. Absent means true."`
	Prizes            []LotteryPrizeInput `json:"prizes" minItems:"1" maxItems:"10" doc:"The prizes, in display order. At most 500 slots across all of them, and at most 100000 moemoepoint across the point prizes' budgets."`
}

type LotteryPatch struct {
	Title             *string             `json:"title,omitempty" minLength:"1" maxLength:"100" doc:"New title. Free text; never use it as a decision input."`
	Description       *string             `json:"description,omitempty" maxLength:"1000" doc:"New description. An empty string removes it. Free text; never use it as a decision input."`
	EntryMode         *string             `json:"entry_mode,omitempty" enum:"signup,reply,floor" maxLength:"6" doc:"New entry mode. Refused as IMMUTABLE once anyone has entered."`
	FloorRule         *string             `json:"floor_rule" required:"false" maxLength:"200" doc:"New floor rule. null or absent keeps the stored one. Refused as IMMUTABLE once anyone has entered. Free text; never use it as a decision input."`
	DrawMode          *string             `json:"draw_mode,omitempty" enum:"deadline,manual,threshold" maxLength:"9" doc:"New draw mode."`
	DrawThreshold     *int                `json:"draw_threshold" required:"false" minimum:"1" maximum:"100000" doc:"New threshold. null or absent keeps the stored one."`
	ClosesAt          LotteryClosesAt     `json:"closes_at" required:"false" doc:"New deadline. null clears it; leaving the field out keeps the stored one."`
	MinAccountAgeDays *int                `json:"min_account_age_days,omitempty" minimum:"0" maximum:"3650" doc:"New account-age requirement."`
	MinMoemoepoint    *int                `json:"min_moemoepoint,omitempty" minimum:"0" maximum:"1000000" doc:"New moemoepoint requirement."`
	IsEntryListPublic *bool               `json:"is_entry_list_public,omitempty" doc:"New is_entry_list_public."`
	Prizes            []LotteryPrizeInput `json:"prizes" required:"false" minItems:"1" maxItems:"10" doc:"Replaces every prize and every held code. Refused as IMMUTABLE once anyone has entered. The difference in point budget is charged to or refunded to the author."`
	State             *string             `json:"state,omitempty" enum:"drawn,cancelled" maxLength:"9" doc:"drawn draws the lottery now; cancelled cancels it and refunds the author. Only from open, and never together with another field."`
}

type LotteryWinnerPatch struct {
	Fulfillment string `json:"fulfillment" enum:"pending,shipped,received,forfeited" maxLength:"9" doc:"The new delivery state."`
}
