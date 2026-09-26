package apiv1

import "kun-galgame-api/internal/apiv1/repr"

type StickerPack struct {
	Object      string    `json:"object" enum:"sticker_pack" maxLength:"12" doc:"Type discriminant. Always sticker_pack."`
	ID          string    `json:"id" pattern:"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" maxLength:"36" doc:"The pack's id on sticker.kungal.com. Stable."`
	DisplayName string    `json:"display_name" maxLength:"200" doc:"The pack's name, for its tab in the picker. Free text; never use it as a decision input."`
	Stickers    []Sticker `json:"stickers" minItems:"1" doc:"In picker order. Never empty."`
}

type Sticker struct {
	Object      string      `json:"object" enum:"sticker" maxLength:"7" doc:"Type discriminant. Always sticker."`
	ID          string      `json:"id" pattern:"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" maxLength:"36" doc:"The sticker's id on sticker.kungal.com. Stable."`
	DisplayName string      `json:"display_name" maxLength:"220" doc:"The pack's display_name, \" - \" and the sticker's 1-based position in the pack: the alt text and title the picker inserts. Free text; never use it as a decision input."`
	Image       *repr.Image `json:"image" doc:"The sticker at original size. Never null here. A sticker goes into a body as a Markdown image of its 320-pixel variant, /image/{hash}_320, with display_name as its alt text and title."`
}

type listStickerPacksInput struct {
	IfNoneMatch string `header:"If-None-Match" maxLength:"1024" pattern:"^(\\*|(W/)?\"[!#-~]{0,64}\"( *, *(W/)?\"[!#-~]{0,64}\")*)$" doc:"The ETag of a copy already held. While it still matches, the answer is 304 with no body."`
}

type listStickerPacksOutput struct {
	Status       int
	ETag         string `header:"ETag" maxLength:"66" doc:"Strong validator of the whole list. Send it back as If-None-Match."`
	CacheControl string `header:"Cache-Control" maxLength:"32" doc:"private, max-age=3600. The list is the same for every viewer; private keeps shared caches out because the response also carries per-origin CORS headers."`
	Body         repr.List[StickerPack]
}
