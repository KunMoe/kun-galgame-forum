package content

import (
	"encoding/json"
	"reflect"
	"testing"

	"kun-galgame-api/internal/apiv1/repr"
)

func everyConstructor() []any {
	one := 1
	yes := true
	lang := "go"
	left := "left"
	return []any{
		NewDocument(nil),
		NewParagraph(nil),
		NewHeading(2, "a", nil),
		NewThematicBreak(),
		NewBlockquote(nil),
		NewList(true, &one, false, nil),
		NewListItem(&yes, nil),
		NewCode(&lang, "x"),
		NewMath("x"),
		NewTable(nil),
		NewTableRow(nil),
		NewTableCell(&left, nil),
		NewSpoiler(nil),
		NewText("x"),
		NewEmphasis(nil),
		NewStrong(nil),
		NewStrikethrough(nil),
		NewInlineCode("x"),
		NewInlineMath("x"),
		NewBreak(),
		NewLink("https://example.com", nil),
		NewImage("https://example.com/a.png", "", nil),
		NewVideo("https://example.com/a.mp4"),
		NewInlineSpoiler(nil),
		NewMention(repr.DeletedUserRef(3)),
		NewReplyReference(48, 2),
	}
}

func TestConstructorsSetTheirOwnDiscriminant(t *testing.T) {
	seen := map[reflect.Type]bool{}
	for _, n := range everyConstructor() {
		v := reflect.ValueOf(n)
		want := Discriminant(v.Type())
		if want == "" {
			t.Fatalf("%s has no Object enum tag", v.Type())
		}
		if got := v.FieldByName("Object").String(); got != want {
			t.Errorf("%s: object %q, want %q", v.Type(), got, want)
		}
		seen[v.Type()] = true
	}
	for _, t2 := range append(append([]reflect.Type{}, blockMembers...), inlineMembers...) {
		if !seen[t2] {
			t.Errorf("no constructor exercised for %s", t2)
		}
	}
}

func TestUnionMembersAreExactlyTheMarkedTypes(t *testing.T) {
	blockIface := reflect.TypeFor[Block]()
	inlineIface := reflect.TypeFor[Inline]()
	tokens := map[string]reflect.Type{}
	check := func(members []reflect.Type, iface reflect.Type) {
		for _, m := range members {
			if !m.Implements(iface) {
				t.Errorf("%s is listed but does not implement %s", m, iface)
			}
			tok := Discriminant(m)
			if prev, dup := tokens[tok]; dup {
				t.Errorf("token %q used by %s and %s", tok, prev, m)
			}
			tokens[tok] = m
		}
	}
	check(blockMembers, blockIface)
	check(inlineMembers, inlineIface)

	for _, n := range everyConstructor() {
		typ := reflect.TypeOf(n)
		if typ.Implements(blockIface) && !contains(blockMembers, typ) {
			t.Errorf("%s implements Block but is missing from blockMembers", typ)
		}
		if typ.Implements(inlineIface) && !contains(inlineMembers, typ) {
			t.Errorf("%s implements Inline but is missing from inlineMembers", typ)
		}
	}
}

func contains(ts []reflect.Type, t reflect.Type) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}

func TestNilChildrenMarshalAsEmptyArrays(t *testing.T) {
	for _, n := range everyConstructor() {
		raw, err := json.Marshal(n)
		if err != nil {
			t.Fatalf("%T: %v", n, err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		if c, ok := m["children"]; ok {
			if arr, isArr := c.([]any); !isArr || len(arr) != 0 {
				t.Errorf("%T children = %v, want []", n, c)
			}
		}
	}
}

func TestDocumentWireShape(t *testing.T) {
	two := 2
	doc := NewDocument(Blocks{
		NewHeading(2, "intro", Inlines{NewText("Intro")}),
		NewParagraph(Inlines{
			NewText("hi "),
			NewMention(repr.DeletedUserRef(3)),
			NewText(" "),
			NewReplyReference(48, 2),
			NewBreak(),
			NewLink("https://example.com/", Inlines{NewStrong(Inlines{NewText("x")})}),
		}),
		NewList(true, &two, false, ListItems{NewListItem(nil, Blocks{NewParagraph(Inlines{NewText("a")})})}),
		NewParagraph(nil),
	})
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"object":"document","children":[` +
		`{"object":"heading","depth":2,"anchor":"intro","children":[{"object":"text","value":"Intro"}]},` +
		`{"object":"paragraph","children":[{"object":"text","value":"hi "},` +
		`{"object":"mention","mentioned_user":{"object":"user","id":"3","name":null,"avatar":null,"avatar_frame":null}},` +
		`{"object":"text","value":" "},{"object":"reply_reference","reply_id":"48","floor":2},{"object":"break"},` +
		`{"object":"link","url":"https://example.com/","children":[{"object":"strong","children":[{"object":"text","value":"x"}]}]}]},` +
		`{"object":"list","is_ordered":true,"start":2,"is_spread":false,"children":[{"object":"list_item","is_checked":null,"children":[{"object":"paragraph","children":[{"object":"text","value":"a"}]}]}]},` +
		`{"object":"paragraph","children":[]}]}`
	if string(raw) != want {
		t.Errorf("wire shape\n got %s\nwant %s", raw, want)
	}
}
