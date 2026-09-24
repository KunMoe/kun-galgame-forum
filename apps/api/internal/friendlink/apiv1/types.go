package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type FriendLink struct {
	Object             string         `json:"object" enum:"friend_link" maxLength:"11" doc:"Type discriminant. Always friend_link."`
	ID                 repr.DecimalID `json:"id" doc:"Friend link id. JSON string of a decimal integer."`
	FriendLinkCategory string         `json:"friend_link_category" enum:"official,galgame,others" maxLength:"8" doc:"Shelf of the friend-link page, displayed in the order official, galgame, others. Clients label the tokens themselves."`
	Title              string         `json:"title" maxLength:"100" doc:"Site name. Free text; never use it as a decision input."`
	URL                string         `json:"url" format:"uri" maxLength:"500" doc:"The linked site. Always http or https."`
	Description        string         `json:"description" maxLength:"500" doc:"Short blurb. Empty string if none. Free text; never use it as a decision input."`
	Banner             *repr.Image    `json:"banner" doc:"Banner image. null when the link has none."`
	State              string         `json:"state" enum:"normal,down" maxLength:"6" doc:"normal, or down when the linked site is offline."`
}

type BannerImageHash string

func (BannerImageHash) Schema(huma.Registry) *huma.Schema {
	n := 64
	return &huma.Schema{
		Type:        huma.TypeString,
		Pattern:     `^([0-9a-f]{64})?$`,
		MaxLength:   &n,
		Description: "Image-service content hash of the banner, or an empty string for no banner.",
	}
}

type FriendLinkCreate struct {
	FriendLinkCategory string           `json:"friend_link_category" enum:"official,galgame,others" maxLength:"8" doc:"Shelf. The new link goes last on it."`
	Title              string           `json:"title" minLength:"1" maxLength:"100" doc:"Site name. Trimmed; only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	URL                string           `json:"url" format:"uri" pattern:"^https?://" maxLength:"500" doc:"The linked site. Only http and https are accepted."`
	Description        *string          `json:"description,omitempty" maxLength:"500" doc:"Short blurb. Trimmed. Absent means empty. Free text; never use it as a decision input."`
	BannerImageHash    *BannerImageHash `json:"banner_image_hash,omitempty" doc:"Banner by image-service hash. Absent or an empty string means no banner."`
	State              *string          `json:"state,omitempty" enum:"normal,down" maxLength:"6" doc:"Absent means normal."`
}

type FriendLinkPatch struct {
	FriendLinkCategory *string          `json:"friend_link_category,omitempty" enum:"official,galgame,others" maxLength:"8" doc:"New shelf. The link moves to the end of it."`
	Title              *string          `json:"title,omitempty" minLength:"1" maxLength:"100" doc:"New site name, checked as in createFriendLink. Free text; never use it as a decision input."`
	URL                *string          `json:"url,omitempty" format:"uri" pattern:"^https?://" maxLength:"500" doc:"New link. Only http and https are accepted."`
	Description        *string          `json:"description,omitempty" maxLength:"500" doc:"New blurb. Trimmed. Free text; never use it as a decision input."`
	BannerImageHash    *BannerImageHash `json:"banner_image_hash,omitempty" doc:"New banner by image-service hash. An empty string removes the banner."`
	State              *string          `json:"state,omitempty" enum:"normal,down" maxLength:"6" doc:"New state."`
}

type FriendLinkOrder struct {
	FriendLinkCategory string           `json:"friend_link_category" enum:"official,galgame,others" maxLength:"8" doc:"The shelf being reordered."`
	FriendLinkIDs      []repr.DecimalID `json:"friend_link_ids" minItems:"1" maxItems:"500" uniqueItems:"true" doc:"Every link on the shelf, once each, in the new order. An id that is not on this shelf is UNKNOWN_REFERENCE; leaving one out is TOO_FEW_ITEMS with min_items set to the shelf's size."`
}

type listFriendLinksInput struct {
	collect.Page
	FriendLinkCategory string `query:"friend_link_category" enum:"official,galgame,others" maxLength:"8" doc:"When set, only this shelf. Omitted means every shelf."`
}

type listFriendLinksOutput struct {
	Body repr.List[FriendLink]
}

type friendLinkInput struct {
	FriendLinkID string `path:"friend_link_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Friend link id."`
}

type friendLinkOutput struct {
	Body FriendLink
}

type createFriendLinkInput struct {
	Body FriendLinkCreate
}

type createFriendLinkOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new link, /api/v1/admin/friend-links/{friend_link_id}."`
	Body     FriendLink
}

type updateFriendLinkInput struct {
	FriendLinkID string `path:"friend_link_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Friend link id."`
	Body         FriendLinkPatch
}

type putFriendLinkOrderInput struct {
	Body FriendLinkOrder
}
