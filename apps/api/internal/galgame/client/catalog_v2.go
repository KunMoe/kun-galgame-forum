package client

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/errors"
)

type v2Problem struct {
	Code      string `json:"code"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Status    int    `json:"status"`
	CurrentID string `json:"current_id"`
}

func catalogOrigin(raw string) string {
	u := strings.TrimRight(strings.TrimSpace(raw), "/")
	for _, suf := range []string{"/api/v1", "/v2", "/v1"} {
		if strings.HasSuffix(u, suf) {
			return strings.TrimSuffix(u, suf)
		}
	}
	return u
}

// Catalog is read entirely over /v2. The /v1 family is being retired upstream,
// so nothing here may address it — a surviving caller does not degrade, it 404s
// the moment infra deletes the route.
//
// Reads lived on /v1 from 2026-08-26 to 2026-08-28 because v2 was then a strict
// subset: entity reprs were id+name with no include vocabulary, covers[].sexual
// was typed as a string so one field failed the whole record, and the limiter
// counted per client IP at 100/min regardless of key tier, which spent the
// backend's whole minute on one page view. Infra PRs #79/#81/#83/#84/#86 closed
// the field gaps and the limiter now honours the internal tier; each was
// re-verified against the running catalog before this flip, face by face,
// rather than taken from the report.

// v2CatalogDetailInclude is the full include vocabulary of each detail face.
// v1 returned every block by default; v2 returns nothing that was not asked for,
// so a face reached with no include= answers a bare id+name row and the page
// renders empty without erroring. The tokens are the faces' own enums — an
// unknown one is a 400, so this list has to track catalog's collect specs.
var v2CatalogDetailInclude = map[string]string{
	"works":        "titles,refs,relations,credits,releases,popularity,ratings,tags,playtimes,series,platforms,intros,covers,screenshots,characters,companies,engines,links",
	"characters":   "gender,birthday,height_cm,weight_kg,measurements,blood_type,instance_of_id,image,figure,traits,aliases,intros,refs",
	"companies":    "aliases,logo,intros,links",
	"credit-names": "aliases,photo,siblings,intros,links,refs",
	"tags":         "intros",
	"series":       "intros,refs",
}

func v2CatalogPath(path string) string {
	switch {
	case path == "/catalog/works/search":
		return "/v2/catalog/works"
	case strings.HasPrefix(path, "/catalog/labels/") && strings.HasSuffix(path, "/relation-graph"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/catalog/labels/"), "/relation-graph")
		return "/v2/catalog/companies/" + id + "/graph"
	case strings.HasPrefix(path, "/catalog/labels"):
		return "/v2/catalog/companies" + strings.TrimPrefix(path, "/catalog/labels")
	case strings.HasPrefix(path, "/catalog/names"):
		return "/v2/catalog/credit-names" + strings.TrimPrefix(path, "/catalog/names")
	case path == "/catalog/calendar/pending", path == "/catalog/calendar/tba":
		return "/v2/catalog/calendar"
	case strings.HasPrefix(path, "/catalog/"):
		return "/v2" + path
	default:
		return "/v2/catalog" + path
	}
}

func v2CatalogQuery(path string, query url.Values) url.Values {
	out := url.Values{}
	for k, vs := range query {
		out[k] = append([]string(nil), vs...)
	}
	for _, flag := range []string{"nsfw", "has_works"} {
		switch out.Get(flag) {
		case "1", "on", "yes":
			out.Set(flag, "true")
		case "0", "off", "no":
			out.Del(flag)
		}
	}
	if entity, ok := v2DetailEntity(v2CatalogPath(path)); ok {
		out.Set("include", v2CatalogDetailInclude[entity])
	} else if inc, ok := v2CatalogListInclude[v2ListEntity(v2CatalogPath(path))]; ok {
		out.Set("include", inc)
	} else if inc := out.Get("include"); inc != "" {
		inc = strings.ReplaceAll(inc, "names", "titles")
		inc = strings.ReplaceAll(inc, "labels", "companies")
		out.Set("include", inc)
	}
	// v1 took a numeric ceiling named `spoilers`; v2 takes a closed enum named
	// `spoiler`, and only on the three faces that have a spoiler axis. Leaving
	// the v1 name behind would not fail — huma drops unknown query parameters
	// silently — it would just answer the default `none` and hide every
	// spoilered tag and trait, which is the bug the ceiling exists to avoid.
	if ceiling := out.Get("spoilers"); ceiling != "" {
		out.Del("spoilers")
		if v2HasSpoilerAxis(v2CatalogPath(path)) {
			n, _ := strconv.Atoi(ceiling)
			out.Set("spoiler", v2SpoilerCeiling(n))
		}
	}
	if lid := out.Get("label_id"); lid != "" {
		out.Del("label_id")
		out.Set("company_id", lid)
	}
	if out.Get("label_rollup") == "1" || out.Get("label_rollup") == "true" {
		out.Del("label_rollup")
		out.Set("company_rollup", "true")
	}
	if t := out.Get("type"); t != "" && (path == "/catalog/search" || strings.HasSuffix(v2CatalogPath(path), "/search")) {
		out.Del("type")
		out.Set("object", v2SearchObject(t))
	}
	if path == "/catalog/search" {
		if page := out.Get("page"); page != "" {
			out.Del("page")
			if n, err := strconv.Atoi(page); err == nil && n > 1 {
				out.Set("cursor", encodePageCursor(n))
			}
		}
	}
	if path == "/catalog/works/search" || path == "/catalog/works" {
		if out.Get("include_total") == "" {
			out.Set("include_total", "true")
		}
		convertedPage := false
		if page := out.Get("page"); page != "" {
			out.Del("page")
			if n, err := strconv.Atoi(page); err == nil && n > 1 {
				out.Set("cursor", encodePageCursor(n))
				convertedPage = true
			}
		}
		if out.Get("q") == "" && out.Get("ids") == "" && out.Get("refs") == "" && out.Get("facets") == "" {
			if path == "/catalog/works/search" || convertedPage {
				out.Set("facets", "olang,tag_id")
			}
		}
	}
	// v2 folded v1's three calendar routes into one windowed face, and both
	// "announced" and "unknown" resolve to the SAME undated bucket there — so
	// 年内待定 and 发售日期未定 served identical rows. The year-known bucket is
	// selected by precision, not by a status.
	switch path {
	case "/catalog/calendar/pending":
		out.Set("precision", "year")
	case "/catalog/calendar/tba":
		out.Set("status", "unknown")
	}
	// v2's calendar answers with items and next_cursor only, so the month
	// header's 共 N 部 has no source unless the total is asked for.
	if strings.HasPrefix(path, "/catalog/calendar") && out.Get("include_total") == "" {
		out.Set("include_total", "true")
	}
	return out
}

func v2SearchObject(t string) string {
	switch t {
	case "labels", "label", "officials", "official":
		return "company"
	case "names", "name", "staff":
		return "credit_name"
	case "characters", "character":
		return "character"
	case "tags", "tag":
		return "tag"
	case "engines", "engine":
		return "engine"
	case "series":
		return "series"
	case "works", "work", "galgame":
		return "work"
	default:
		return t
	}
}

func encodePageCursor(page int) string {
	return "cur_" + base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(page)))
}

// The sub-faces page by opaque cursor and drop `offset` silently, but their
// cursor is the next offset in base64 — verified against a live catalog, where
// cursor=cur_Mw skips exactly three rows and a limit=50 page answers cur_NTA to
// v1's next_offset=50. Reading it back is what keeps next_offset exact for a
// credits page, whose items are grouped by work and so never number the rows.
func encodeOffsetCursor(offset int) string {
	return "cur_" + base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeOffsetCursor(cursor string) (int, bool) {
	rest, ok := strings.CutPrefix(cursor, "cur_")
	if !ok {
		return 0, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return 0, false
	}
	offset, err := strconv.Atoi(string(raw))
	if err != nil || offset < 0 {
		return 0, false
	}
	return offset, true
}

func (c *GalgameClient) doV2(ctx context.Context, method, path string, query url.Values, body any) (int, []byte, *errors.AppError) {
	v2Path := v2CatalogPath(path)
	q := v2CatalogQuery(path, query)
	var (
		status   int
		respBody []byte
		appErr   *errors.AppError
	)
	if method == http.MethodGet {
		status, respBody, appErr = c.getV2(ctx, v2Path, q)
	} else {
		status, respBody, appErr = c.doV2HTTP(ctx, method, v2Path, q, body)
	}
	if appErr != nil {
		return status, respBody, appErr
	}
	out := rewriteV2JSON(respBody, c.imageCDNBase)
	if method == http.MethodGet && status >= 200 && status < 300 {
		out = c.spliceV2Subface(ctx, path, v2Path, query, out)
	}
	return status, out, nil
}

func (c *GalgameClient) doV2HTTP(ctx context.Context, method, v2Path string, q url.Values, body any) (int, []byte, *errors.AppError) {
	reqURL := c.origin + v2Path
	if len(q) > 0 {
		reqURL += "?" + q.Encode()
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, errors.ErrInternal("序列化请求失败")
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, rdr)
	if err != nil {
		return 0, nil, errors.ErrInternal("创建请求失败")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Galgame 服务请求失败 (传输层)",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return 0, nil, errors.ErrInternal(fmt.Sprintf("Galgame 服务不可达: %v", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, errors.ErrInternal("读取 Galgame 响应失败")
	}
	return resp.StatusCode, respBody, nil
}

// v2 moved the character's works and the credit name's credits off their detail
// records onto their own cursor-paged sub-faces. They are spliced back into the
// parent here so the decoders and the thirteen call sites keep speaking v1's
// embedded block plus next_offset.
func (c *GalgameClient) spliceV2Subface(
	ctx context.Context, path, v2Path string, query url.Values, doc []byte,
) []byte {
	entity, ok := v2DetailEntity(v2Path)
	if !ok {
		return doc
	}
	var subface, block string
	switch {
	case entity == "characters" && includeHasToken(query, "works"):
		subface, block = "appearances", "works"
	case entity == "credit-names" && includeHasToken(query, "credits"):
		subface, block = "credits", "credits"
	default:
		return doc
	}

	// The sub-face applies the population gate itself, so it has to inherit the
	// parent's: a credits page fetched without nsfw=true answered 14 works where
	// the staff page shows 50, silently cutting every r18 credit off the roster.
	q := v2CatalogQuery(path, query)
	q.Del("include")
	q.Del("offset")
	q.Del("page")
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset > 0 {
		q.Set("cursor", encodeOffsetCursor(offset))
	}
	raw, appErr := c.getV2Raw(ctx, v2Path+"/"+subface, q)
	if appErr != nil {
		return doc
	}

	var page struct {
		Items      []json.RawMessage `json:"items"`
		NextCursor string            `json:"next_cursor"`
	}
	if json.Unmarshal(raw, &page) != nil {
		return doc
	}
	parent := map[string]json.RawMessage{}
	if json.Unmarshal(doc, &parent) != nil {
		return doc
	}
	items, err := json.Marshal(page.Items)
	if err != nil {
		return doc
	}
	parent[block] = items
	if next, ok := decodeOffsetCursor(page.NextCursor); ok {
		parent["next_offset"] = json.RawMessage(strconv.Itoa(next))
	}
	out, err := json.Marshal(parent)
	if err != nil {
		return doc
	}
	return out
}

func (c *GalgameClient) getV2Raw(ctx context.Context, v2Path string, q url.Values) ([]byte, *errors.AppError) {
	status, body, appErr := c.getV2(ctx, v2Path, q)
	if appErr != nil {
		return nil, appErr
	}
	if status < 200 || status >= 300 {
		return nil, errors.New(errors.CodeBiz, "Galgame 资源不存在", status)
	}
	return rewriteV2JSON(body, c.imageCDNBase), nil
}

func includeHasToken(query url.Values, token string) bool {
	for _, want := range strings.Split(query.Get("include"), ",") {
		if strings.TrimSpace(want) == token {
			return true
		}
	}
	return false
}

// v2MergedInto reports the survivor of a merged entity. Catalog answers it two
// ways and both have been seen live: a problem+json ENTITY_MERGED, and a bare
// 301 whose body carries current_id.
func v2MergedInto(status int, raw []byte) (int64, bool) {
	var p v2Problem
	_ = json.Unmarshal(raw, &p)
	if p.Code == "ENTITY_MERGED" && p.CurrentID != "" {
		if id, err := strconv.ParseInt(p.CurrentID, 10, 64); err == nil && id != 0 {
			return id, true
		}
	}
	if status == http.StatusMovedPermanently {
		var moved struct {
			CurrentID json.Number `json:"current_id"`
		}
		if json.Unmarshal(raw, &moved) == nil {
			if id, err := moved.CurrentID.Int64(); err == nil && id != 0 {
				return id, true
			}
		}
	}
	return 0, false
}

func (c *GalgameClient) v2Error(status int, raw []byte) *errors.AppError {
	var p v2Problem
	_ = json.Unmarshal(raw, &p)
	msg := cmp.Or(p.Detail, p.Title, fmt.Sprintf("Galgame 服务返回了非预期响应 (HTTP %d)", status))
	return errors.New(errors.CodeBiz, msg, status)
}

func rewriteV2JSON(raw []byte, cdnBase string) []byte {
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var tree any
	if err := dec.Decode(&tree); err != nil {
		return raw
	}
	if !walkV2(tree, cdnBase) {
		if _, ok := tree.(map[string]any); !ok && tree == nil {
			return raw
		}
		out, err := json.Marshal(tree)
		if err != nil {
			return raw
		}
		return out
	}
	out, err := json.Marshal(tree)
	if err != nil {
		return raw
	}
	return out
}

func walkV2(node any, cdnBase string) bool {
	changed := false
	switch v := node.(type) {
	case map[string]any:
		changed = rewriteV2Object(v, cdnBase) || changed
		for k, child := range v {
			if k == "cover_slots" {
				continue
			}
			if walkV2(child, cdnBase) {
				changed = true
			}
		}
	case []any:
		for _, child := range v {
			if walkV2(child, cdnBase) {
				changed = true
			}
		}
	}
	return changed
}

func rewriteV2Object(v map[string]any, cdnBase string) bool {
	changed := false
	for _, key := range []string{"id", "work_id", "canonical_id", "character_id", "person_id", "from", "to", "from_id", "to_id"} {
		if n, ok := digitString(v[key]); ok {
			v[aliasIDKey(key)] = json.Number(n)
			if key == "from_id" {
				delete(v, "from_id")
			}
			if key == "to_id" {
				delete(v, "to_id")
			}
			changed = true
		}
	}
	if claim, ok := v["claim"].(map[string]any); ok {
		if n, ok := digitString(claim["site_work_id"]); ok {
			claim["site_work_id"] = json.Number(n)
			changed = true
		}
	}
	if companies, ok := v["companies"]; ok {
		if _, has := v["labels"]; !has {
			v["labels"] = companies
			changed = true
		}
	}
	if _, has := v["updated"]; !has {
		if u, ok := v["updated_at"]; ok {
			v["updated"] = u
			changed = true
		}
	}
	if _, has := v["created"]; !has {
		if u, ok := v["created_at"]; ok {
			v["created"] = u
			changed = true
		}
	}
	if machine, ok := v["is_machine"]; ok {
		if _, has := v["machine"]; !has {
			v["machine"] = machine
			changed = true
		}
	}
	if intro, ok := v["value"].(string); ok && v["intro"] == nil && (v["lang"] != nil || v["source"] != nil) {
		v["intro"] = intro
		changed = true
	}
	if sexual, ok := v["is_sexual"]; ok {
		if _, has := v["sexual"]; !has {
			v["sexual"] = sexual
			changed = true
		}
	}
	if lie, ok := v["is_lie"]; ok {
		if _, has := v["lie"]; !has {
			v["lie"] = lie
			changed = true
		}
	}
	if kind, ok := v["company_kind"].(string); ok {
		if _, has := v["kind"]; !has {
			v["kind"] = kind
			changed = true
		}
		if _, has := v["label_kind"]; !has {
			v["label_kind"] = kind
			changed = true
		}
	}
	// v1 sent one `kind` that meant developer/publisher; v2 split it into
	// company_kind (what the company IS) and attribution_role (what it DID on
	// this work). Only the first was ever mapped, so the detail page printed a
	// raw "game_brand" as a role chip beside the 游戏品牌 category chip.
	if role, ok := v["attribution_role"].(string); ok {
		if _, has := v["role"]; !has {
			v["role"] = role
			changed = true
		}
	}
	if kind, ok := v["tag_kind"].(string); ok {
		if _, has := v["kind"]; !has {
			v["kind"] = kind
			changed = true
		}
	}
	if kind, ok := v["title_kind"].(string); ok {
		if _, has := v["kind"]; !has {
			v["kind"] = kind
			changed = true
		}
	}
	// v2 spec gate G8 forbids a bare `kind`, so the cover's grouping travels as
	// cover_kind and the gallery renders one flat pile without this alias.
	if kind, ok := v["cover_kind"].(string); ok {
		if _, has := v["kind"]; !has {
			v["kind"] = kind
			changed = true
		}
	}
	// The rollup lane's one-hop attribution: v1 named it via_label, v2 names it
	// via_company. Absent, every rollup row reads as directly attributed, which
	// collapsed a company's own/imprint split to 52/0 where it is 39/13.
	if via, ok := v["via_company"]; ok {
		if _, has := v["via_label"]; !has {
			v["via_label"] = via
			changed = true
		}
	}
	if _, has := v["cover_slots"]; !has {
		if slots, ok := v["covers"].(map[string]any); ok {
			v["cover_slots"] = slots
			changed = true
		} else if slots := coverSlotsFromImages(v, cdnBase); slots != nil {
			v["cover_slots"] = slots
			changed = true
		}
	}
	if img, ok := v["cover"].(map[string]any); ok {
		if url := imageURL(img, cdnBase); url != "" {
			v["cover"] = url
			changed = true
		}
	}
	for _, key := range []string{"image", "figure"} {
		if img, ok := v[key].(map[string]any); ok {
			if url := imageURL(img, cdnBase); url != "" {
				v[key] = url
				changed = true
			}
		}
	}
	if hash, ok := v["hash"].(string); ok && hash != "" {
		if url, _ := v["url"].(string); url == "" {
			if u := bannerURLFromHash(cdnBase, hash); u != "" {
				v["url"] = u
				changed = true
			}
		}
	}
	if role, ok := v["roster_role"].(string); ok {
		if _, has := v["kind"]; !has {
			v["kind"] = role
			changed = true
		}
	}
	if spoiler, ok := v["spoiler"].(string); ok {
		v["spoiler"] = spoilerLevel(spoiler)
		changed = true
	}
	// The work's tag rows are the one place v2 names the canonical tag `id`
	// while every forum reader calls it canonical_id — and catalog_detail.go
	// drops a row whose canonical_id is 0, so the mismatch empties the tag
	// panel instead of degrading it. A tag row is the only object carrying
	// both a kind and a spoiler; the standalone tag list row has no spoiler.
	if _, hasKind := v["tag_kind"]; hasKind {
		if _, hasSpoiler := v["spoiler"]; hasSpoiler {
			if _, has := v["canonical_id"]; !has {
				if id, ok := v["id"]; ok && id != nil {
					v["canonical_id"] = id
					changed = true
				}
			}
		}
	}
	if sexual, ok := v["sexual"].(string); ok {
		v["sexual"] = json.Number(strconv.Itoa(sexualLevel(sexual)))
		changed = true
	}
	for _, pair := range [][2]string{{"logo", "logo_hash"}, {"photo", "photo_hash"}} {
		if img, ok := v[pair[0]].(map[string]any); ok {
			if hash, _ := img["hash"].(string); hash != "" {
				if _, has := v[pair[1]]; !has {
					v[pair[1]] = hash
					changed = true
				}
			}
		}
	}
	if kind, ok := v["alias_kind"].(string); ok {
		if _, has := v["kind"]; !has {
			v["kind"] = kind
			changed = true
		}
	}
	if gender, ok := v["gender"].(string); ok {
		if code, known := genderCode(gender); known {
			v["gender"] = json.Number(strconv.Itoa(code))
		} else {
			v["gender"] = nil
		}
		changed = true
	}
	for _, pair := range [][2]string{{"birth_year", "birth_y"}, {"birth_month", "birth_m"}, {"birth_day", "birth_d"}} {
		if n, ok := v[pair[0]]; ok {
			if _, has := v[pair[1]]; !has {
				v[pair[1]] = n
				changed = true
			}
		}
	}
	if name, ok := v["character_name"]; ok {
		if _, has := v["character"]; !has {
			v["character"] = name
			changed = true
		}
	}
	if obj, ok := v["object"].(string); ok && obj == "search_result" {
		if t, ok := v["target_object"].(string); ok {
			v["entity_type"] = searchEntityType(t)
			changed = true
		}
	}
	return changed
}

func aliasIDKey(key string) string {
	switch key {
	case "from_id":
		return "from"
	case "to_id":
		return "to"
	default:
		return key
	}
}

func digitString(v any) (string, bool) {
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return s, true
}

func coverSlotsFromImages(v map[string]any, cdnBase string) map[string]any {
	if _, hasPortrait := v["portrait"]; hasPortrait {
		return nil
	}
	if !looksLikeV2Image(v["cover"]) && !looksLikeV2Image(v["banner"]) {
		return nil
	}
	portrait := imageToSlot(v["cover"], cdnBase)
	banner := imageToSlot(v["banner"], cdnBase)
	if portrait == nil && banner == nil {
		return nil
	}
	return map[string]any{"portrait": portrait, "banner": banner}
}

func looksLikeV2Image(v any) bool {
	img, ok := v.(map[string]any)
	if !ok {
		return false
	}
	if url, _ := img["url"].(string); url != "" {
		return true
	}
	hash, _ := img["hash"].(string)
	return hash != ""
}

func imageURL(img map[string]any, cdnBase string) string {
	if url, _ := img["url"].(string); url != "" {
		return url
	}
	hash, _ := img["hash"].(string)
	return bannerURLFromHash(cdnBase, hash)
}

func imageToSlot(v any, cdnBase string) map[string]any {
	img, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	url := imageURL(img, cdnBase)
	if url == "" {
		return nil
	}
	slot := map[string]any{"url": url, "source": img["source"]}
	if w, ok := img["width"]; ok {
		slot["width"] = w
	}
	if h, ok := img["height"]; ok {
		slot["height"] = h
	}
	if th, ok := img["thumbhash"]; ok {
		slot["thumbhash"] = th
	}
	if sexual, ok := img["sexual"].(string); ok {
		slot["sexual"] = json.Number(strconv.Itoa(sexualLevel(sexual)))
	}
	return slot
}

func spoilerLevel(s string) int {
	switch s {
	case "minimum", "minor":
		return 1
	case "spoilered", "major":
		return 2
	default:
		return 0
	}
}

// sexualLevel maps the image grading enum back onto v1's integer scale, which
// is what the cover gate compares against. Decoding it as a string is not an
// option: the cover block is typed int, so one "safe" made the WHOLE work
// detail fail to decode rather than just that field.
func sexualLevel(s string) int {
	switch s {
	case "suggestive":
		return 1
	case "explicit":
		return 2
	default:
		return 0
	}
}

func genderCode(s string) (int, bool) {
	switch s {
	case "male":
		return 1, true
	case "female":
		return 2, true
	default:
		return 0, false
	}
}

func searchEntityType(object string) string {
	switch object {
	case "company":
		return "label"
	case "credit_name":
		return "name"
	default:
		return object
	}
}

// v2CatalogListInclude is what a browse row has to ask for to match the row v1
// answered with. Deliberately narrower than the detail vocabulary: the company
// index pages hundreds of rows and has never rendered an intro or a link.
var v2CatalogListInclude = map[string]string{
	"companies": "aliases,logo",
}

func v2ListEntity(v2Path string) string {
	rest, ok := strings.CutPrefix(v2Path, "/v2/catalog/")
	if !ok || strings.Contains(rest, "/") {
		return ""
	}
	return rest
}

// v2DetailEntity names the entity whose single-record face this path is, if it
// is one. /v2/catalog/works is the list; /v2/catalog/works/{id} is the detail;
// /v2/catalog/works/{id}/tags is a sub-face and keeps its own vocabulary.
func v2DetailEntity(v2Path string) (string, bool) {
	rest, ok := strings.CutPrefix(v2Path, "/v2/catalog/")
	if !ok {
		return "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) == 3 && parts[2] == "graph" {
		parts = parts[:2]
	}
	if len(parts) != 2 || parts[1] == "" {
		return "", false
	}
	if _, has := v2CatalogDetailInclude[parts[0]]; !has {
		return "", false
	}
	return parts[0], true
}

func v2HasSpoilerAxis(v2Path string) bool {
	rest, ok := strings.CutPrefix(v2Path, "/v2/catalog/")
	if !ok {
		return false
	}
	switch parts := strings.Split(rest, "/"); {
	case len(parts) == 2 && parts[0] == "characters":
		return true
	case len(parts) == 2 && parts[0] == "works":
		return true
	case len(parts) == 3 && parts[0] == "works" && parts[2] == "tags":
		return true
	}
	return false
}

func v2SpoilerCeiling(n int) string {
	switch {
	case n <= 0:
		return "none"
	case n == 1:
		return "minor"
	default:
		return "major"
	}
}
