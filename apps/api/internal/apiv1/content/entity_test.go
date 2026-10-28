package content_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/userclient"
)

var testTopic = content.TopicPreview{
	Object: "topic", ID: "1", Title: "T", CoverImages: []repr.Image{},
	Author: repr.DeletedUserRef(9), CreatedAt: repr.Timestamp(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
}

var testWork = content.WorkPreview{WorkRef: repr.NewWorkRef(3, repr.NewCatalogName("W", "", nil), nil, false), TopTags: []workrepr.TagRef{}}

var testResource = content.GalgameResourcePreview{
	Object: "galgame_resource", ID: "4", Work: &testWork.WorkRef, Author: repr.DeletedUserRef(9),
	ResourceType: "game", ResourceLanguages: []workrepr.ResourceLanguage{"ja-jp"},
	ResourcePlatforms: []workrepr.ResourcePlatform{"win"}, State: "valid",
	CreatedAt: repr.Timestamp(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
}

func entityConverter(asked *[]content.EntityRef, fail bool) *content.Converter {
	conv := defaultConverter()
	conv.Entities = func(_ context.Context, refs []content.EntityRef) (map[content.EntityRef]content.EntityTarget, error) {
		if asked != nil {
			*asked = append(*asked, refs...)
		}
		if fail {
			return nil, errors.New("down")
		}
		return map[content.EntityRef]content.EntityTarget{
			{Type: content.EntityTopic, ID: 1}:           content.TopicTarget(testTopic),
			{Type: content.EntityWork, ID: 3}:            content.WorkTarget(testWork),
			{Type: content.EntityGalgameResource, ID: 4}: content.GalgameResourceTarget(testResource),
		}, nil
	}
	return conv
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func linkJSON(url, text string) string {
	return `{"object":"link","url":` + quoteJSON(url) + `,"children":[` + textJSON(text) + `]}`
}

func cardJSON(target any, url, text string) string {
	return `{"object":"entity_card","target":` + mustJSON(target) + `,"children":[` + paraJSON(linkJSON(url, text)) + `]}`
}

func chipJSON(target any, url, text string) string {
	return `{"object":"entity_link","target":` + mustJSON(target) + `,"children":[` + linkJSON(url, text) + `]}`
}

func TestEntityLinks(t *testing.T) {
	topicURL := testSite + "/topic/1"
	unavailable := content.NewUnavailableEntity(content.EntityRef{Type: content.EntityTopic, ID: 2})
	user := repr.NewUserRef(testCDN, userclient.User{ID: 7, Name: "u7"})
	cases := []convertCase{
		{name: "bare url alone is a card", src: topicURL, want: docJSON(cardJSON(testTopic, topicURL, topicURL))},
		{name: "autolink alone is a card", src: "<" + topicURL + ">", want: docJSON(cardJSON(testTopic, topicURL, topicURL))},
		{name: "link whose text is its url is a card", src: "[" + topicURL + "](" + topicURL + ")", want: docJSON(cardJSON(testTopic, topicURL, topicURL))},
		{name: "apex host renders the canonical url", src: "https://kungal.com/topic/1", want: docJSON(cardJSON(testTopic, topicURL, "https://kungal.com/topic/1"))},
		{name: "rooted bare link", src: "[/topic/1](/topic/1)", want: docJSON(cardJSON(testTopic, topicURL, "/topic/1"))},
		{name: "trailing slash", src: topicURL + "/", want: docJSON(cardJSON(testTopic, topicURL, topicURL+"/"))},
		{name: "url among text is a chip", src: "看 " + topicURL + " 这个", want: docJSON(paraJSON(
			textJSON("看 ") + "," + chipJSON(testTopic, topicURL, topicURL) + "," + textJSON(" 这个")))},
		{name: "two urls in one paragraph are chips", src: topicURL + " " + topicURL, want: docJSON(paraJSON(
			chipJSON(testTopic, topicURL, topicURL) + "," + textJSON(" ") + "," + chipJSON(testTopic, topicURL, topicURL)))},
		{name: "www literal is not bare", src: "www.kungal.com/topic/1", want: docJSON(paraJSON(linkJSON("http://www.kungal.com/topic/1", "www.kungal.com/topic/1")))},
		{name: "strong around the url is a chip", src: "**" + topicURL + "**", want: docJSON(paraJSON(
			`{"object":"strong","children":[` + chipJSON(testTopic, topicURL, topicURL) + `]}`))},
		{name: "heading holds a chip", src: "## " + topicURL, want: docJSON(
			`{"object":"heading","depth":2,"anchor":` + `"httpswwwkungalcomtopic1"` + `,"children":[` + chipJSON(testTopic, topicURL, topicURL) + `]}`)},
		{name: "hard break is content", src: topicURL + "\\\nx", want: docJSON(paraJSON(
			chipJSON(testTopic, topicURL, topicURL) + `,{"object":"break"},` + textJSON("x")))},
		{name: "custom text stays a link", src: "[这里](" + topicURL + ")", want: docJSON(paraJSON(linkJSON(topicURL, "这里")))},
		{name: "query stays a link", src: topicURL + "?page=2", want: docJSON(paraJSON(linkJSON(topicURL+"?page=2", topicURL+"?page=2")))},
		{name: "fragment stays a link", src: topicURL + "#k3", want: docJSON(paraJSON(linkJSON(topicURL+"#k3", topicURL+"#k3")))},
		{name: "other host stays a link", src: "https://example.com/topic/1", want: docJSON(paraJSON(linkJSON("https://example.com/topic/1", "https://example.com/topic/1")))},
		{name: "explicit port stays a link", src: "https://www.kungal.com:443/topic/1", want: docJSON(paraJSON(linkJSON("https://www.kungal.com:443/topic/1", "https://www.kungal.com:443/topic/1")))},
		{name: "encoded path stays a link", src: "<" + testSite + "/topic/%31>", want: docJSON(paraJSON(linkJSON(testSite+"/topic/%31", testSite+"/topic/%31")))},
		{name: "unknown path stays a link", src: testSite + "/topic/1/edit", want: docJSON(paraJSON(linkJSON(testSite+"/topic/1/edit", testSite+"/topic/1/edit")))},
		{name: "missing target is unavailable", src: testSite + "/topic/2", want: docJSON(cardJSON(unavailable, testSite+"/topic/2", testSite+"/topic/2"))},
		{name: "work", src: testSite + "/galgame/3", want: docJSON(cardJSON(testWork, testSite+"/galgame/3", testSite+"/galgame/3"))},
		{name: "resource", src: testSite + "/galgame/resource/4", want: docJSON(cardJSON(testResource, testSite+"/galgame/resource/4", testSite+"/galgame/resource/4"))},
		{name: "user", src: testSite + "/user/7", want: docJSON(cardJSON(user, testSite+"/user/7", testSite+"/user/7"))},
		{name: "card in a list item", src: "- " + topicURL, want: docJSON(listJSON(false, "null", false,
			itemJSON("null", cardJSON(testTopic, topicURL, topicURL))))},
		{name: "table cell holds a chip", src: "| a |\n| - |\n| " + topicURL + " |", want: docJSON(
			`{"object":"table","children":[{"object":"table_row","children":[{"object":"table_cell","align":null,"children":[` + textJSON("a") +
				`]}]},{"object":"table_row","children":[{"object":"table_cell","align":null,"children":[` + chipJSON(testTopic, topicURL, topicURL) + `]}]}]}`)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := marshalDoc(t, convertOne(t, entityConverter(nil, false), c.src))
			if got != c.want {
				t.Errorf("src %q\n got %s\nwant %s", c.src, got, c.want)
			}
		})
	}
}

func TestCatalogAndRatingURLsAreEntities(t *testing.T) {
	cases := map[string]content.EntityType{
		"/galgame/character/5": content.EntityCharacter,
		"/galgame/official/5":  content.EntityCompany,
		"/galgame/tag/5":       content.EntityTag,
		"/galgame/series/5":    content.EntitySeries,
		"/galgame/staff/5":     content.EntityCreditName,
		"/galgame-rating/5":    content.EntityRating,
	}
	for path, typ := range cases {
		url := testSite + path
		want := docJSON(cardJSON(content.NewUnavailableEntity(content.EntityRef{Type: typ, ID: 5}), url, url))
		if got := marshalDoc(t, convertOne(t, entityConverter(nil, false), url)); got != want {
			t.Errorf("%s\n got %s\nwant %s", path, got, want)
		}
	}
	for _, path := range []string{"/galgame/official/5/game", "/galgame/trait/5", "/galgame/engine/5"} {
		url := testSite + path
		if got, want := marshalDoc(t, convertOne(t, entityConverter(nil, false), url)), docJSON(paraJSON(linkJSON(url, url))); got != want {
			t.Errorf("%s\n got %s\nwant %s", path, got, want)
		}
	}
}

func TestEntityLinksStayLinksWithoutAResolver(t *testing.T) {
	url := testSite + "/topic/1"
	got := marshalDoc(t, convertOne(t, defaultConverter(), url))
	if want := docJSON(paraJSON(linkJSON(url, url))); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestEntityResolverFailureRendersLinks(t *testing.T) {
	url := testSite + "/topic/1"
	got := marshalDoc(t, convertOne(t, entityConverter(nil, true), url))
	if want := docJSON(paraJSON(linkJSON(url, url))); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestEntityRefsAreLookedUpOnceAcrossABatch(t *testing.T) {
	var asked []content.EntityRef
	conv := entityConverter(&asked, false)
	_, err := conv.Convert(context.Background(), []string{
		testSite + "/topic/1 " + testSite + "/topic/1",
		testSite + "/topic/1\n\n" + testSite + "/galgame/3\n\n" + testSite + "/user/7",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []content.EntityRef{{Type: content.EntityTopic, ID: 1}, {Type: content.EntityWork, ID: 3}}
	if !slices.Equal(asked, want) {
		t.Errorf("asked %v, want %v", asked, want)
	}
}

func TestUntrustedContentHasNoEntityLinks(t *testing.T) {
	url := testSite + "/topic/1"
	doc, err := entityConverter(nil, false).ConvertUntrusted(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := marshalDoc(t, doc), docJSON(paraJSON(linkJSON(url, url))); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}
