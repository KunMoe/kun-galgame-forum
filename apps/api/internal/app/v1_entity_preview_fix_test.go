package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"kun-galgame-api/internal/apiv1/content"
)

func TestEntityPreviewsShowOnlyWhatAnyReaderMaySee(t *testing.T) {
	f := newDetailFix(t)
	resolve := f.entityResolver(f.Config.NextMoeAPI.ImageCDNBase)
	ids := []int{d1TopicPublic, d1TopicHidden, d1TopicLogin, d1TopicRole, d1TopicUsers, d1TopicBanned, d1TopicGone, d1TopicMax}
	refs := make([]content.EntityRef, len(ids))
	for i, id := range ids {
		refs[i] = content.EntityRef{Type: content.EntityTopic, ID: id}
	}
	got, err := resolve(context.Background(), refs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("resolved %d topics, want only the public one: %v", len(got), got)
	}
	raw, err := json.Marshal(got[refs[0]])
	if err != nil {
		t.Fatal(err)
	}
	var preview struct {
		Object      string `json:"object"`
		ID          string `json:"id"`
		Title       string `json:"title"`
		ReplyCount  int    `json:"reply_count"`
		CoverImages []struct {
			Width     *int    `json:"width"`
			Thumbhash *string `json:"thumbhash"`
		} `json:"cover_images"`
		Author struct {
			Name *string `json:"name"`
		} `json:"author"`
	}
	if err := json.Unmarshal(raw, &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Object != "topic" || preview.ID != strconv.Itoa(d1TopicPublic) || preview.Title != "d1-public" || preview.ReplyCount != 5 {
		t.Errorf("preview %s", raw)
	}
	if len(preview.CoverImages) != 1 || preview.CoverImages[0].Width == nil || *preview.CoverImages[0].Width != 640 {
		t.Errorf("cover metadata missing: %s", raw)
	}
	if preview.Author.Name == nil || *preview.Author.Name != "alice" {
		t.Errorf("author %s", raw)
	}
}

func TestTopicBodyLinksBecomeEntityNodes(t *testing.T) {
	f := newDetailFix(t)
	body := "https://www.kungal.com/topic/" + strconv.Itoa(d1TopicPublic) +
		"\n\n见 https://kungal.com/topic/" + strconv.Itoa(d1TopicHidden) + " 和 [这里](https://www.kungal.com/topic/" + strconv.Itoa(d1TopicPublic) + ")"
	if err := f.db.Exec(`UPDATE topic SET content = ? WHERE id = ?`, body, d1TopicList).Error; err != nil {
		t.Fatal(err)
	}
	resp, raw := f.do(t, http.MethodGet, "/api/v1/topics/"+strconv.Itoa(d1TopicList), "", "/topics/{topic_id}", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, raw)
	}
	var topic struct {
		Content struct {
			Children []struct {
				Object string `json:"object"`
				Target struct {
					Object   string `json:"object"`
					ID       string `json:"id"`
					EntityID string `json:"entity_id"`
				} `json:"target"`
				Children []struct {
					Object string `json:"object"`
					Target struct {
						Object   string `json:"object"`
						EntityID string `json:"entity_id"`
					} `json:"target"`
				} `json:"children"`
			} `json:"children"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &topic); err != nil {
		t.Fatal(err)
	}
	blocks := topic.Content.Children
	if len(blocks) != 2 {
		t.Fatalf("blocks %s", raw)
	}
	if blocks[0].Object != "entity_card" || blocks[0].Target.Object != "topic" || blocks[0].Target.ID != strconv.Itoa(d1TopicPublic) {
		t.Errorf("first block is not the public topic's card: %s", raw)
	}
	var objects []string
	for _, n := range blocks[1].Children {
		objects = append(objects, n.Object)
	}
	want := []string{"text", "entity_link", "text", "link"}
	if len(objects) != len(want) {
		t.Fatalf("second paragraph %v, want %v", objects, want)
	}
	for i := range want {
		if objects[i] != want[i] {
			t.Fatalf("second paragraph %v, want %v", objects, want)
		}
	}
	chip := blocks[1].Children[1].Target
	if chip.Object != "unavailable_entity" || chip.EntityID != strconv.Itoa(d1TopicHidden) {
		t.Errorf("hidden topic leaked or misrendered: %+v", chip)
	}
}

func TestListEntityPreviews(t *testing.T) {
	f := newDetailFix(t)
	url := "/api/v1/entity-previews?topic_ids=" + strconv.Itoa(d1TopicPublic) + "," + strconv.Itoa(d1TopicHidden) +
		"&user_ids=" + strconv.Itoa(d1UserAlice)
	resp, raw := f.do(t, http.MethodGet, url, "", "/entity-previews", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, raw)
	}
	var list struct {
		Items []struct {
			Object   string  `json:"object"`
			ID       string  `json:"id"`
			EntityID string  `json:"entity_id"`
			Name     *string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 3 ||
		list.Items[0].Object != "topic" || list.Items[0].ID != strconv.Itoa(d1TopicPublic) ||
		list.Items[1].Object != "unavailable_entity" || list.Items[1].EntityID != strconv.Itoa(d1TopicHidden) ||
		list.Items[2].Object != "user" || list.Items[2].Name == nil || *list.Items[2].Name != "alice" {
		t.Errorf("items %s", raw)
	}

	resp, raw = f.do(t, http.MethodGet, "/api/v1/entity-previews", "", "/entity-previews", nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("no ids: status %d: %s", resp.StatusCode, raw)
	}
}
