package content_test

import (
	"context"
	"testing"

	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"
)

func TestConvertUntrusted(t *testing.T) {
	br := `{"object":"break"}`
	strong := `{"object":"strong","children":[` + textJSON("bold") + `]}`
	link := `{"object":"link","url":` + quoteJSON("https://example.com") + `,"children":[` + textJSON("site") + `]}`
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "script block is literal",
			src:  "<script>alert(1)</script>",
			want: docJSON(paraJSON(textJSON("<script>alert(1)</script>"))),
		},
		{
			name: "inline tags are literal text",
			src:  "a <b>bold</b> word",
			want: docJSON(paraJSON(textJSON("a <b>bold</b> word"))),
		},
		{
			name: "img tag is literal text",
			src:  "<img src=x onerror=alert(1)>",
			want: docJSON(paraJSON(textJSON("<img src=x onerror=alert(1)>"))),
		},
		{
			name: "html block lines joined by breaks",
			src:  "<div>\n<b>x</b>\n</div>",
			want: docJSON(paraJSON(textJSON("<div>") + "," + br + "," + textJSON("<b>x</b>") + "," + br + "," + textJSON("</div>"))),
		},
		{
			name: "mention is plain text",
			src:  "[@admin](kungal-user:1)",
			want: docJSON(paraJSON(textJSON("@admin"))),
		},
		{
			name: "reply reference is plain text",
			src:  "[#3](kungal-reply:9)",
			want: docJSON(paraJSON(textJSON("#3"))),
		},
		{
			name: "javascript link stays unwrapped",
			src:  "[x](javascript:alert(1))",
			want: docJSON(paraJSON(textJSON("x"))),
		},
		{
			name: "markdown strong and link",
			src:  "**bold** and [site](https://example.com)",
			want: docJSON(paraJSON(strong + "," + textJSON(" and ") + "," + link)),
		},
		{name: "empty", src: "", want: docJSON("")},
		{name: "whitespace", src: "  \n\t", want: docJSON("")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := defaultConverter().ConvertUntrusted(context.Background(), tc.src)
			if err != nil {
				t.Fatal(err)
			}
			checkDocument(t, doc)
			if got := marshalDoc(t, doc); got != tc.want {
				t.Errorf("src %q\n got %s\nwant %s", tc.src, got, tc.want)
			}
		})
	}
}

func TestConvertUntrustedDoesNotLookUpUsers(t *testing.T) {
	conv := defaultConverter()
	conv.Users = func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
		t.Fatalf("Users called with %v", ids)
		return nil, nil
	}
	doc, err := conv.ConvertUntrusted(context.Background(), "[@admin](kungal-user:1)\n\n[#3](kungal-reply:9)")
	if err != nil {
		t.Fatal(err)
	}
	checkDocument(t, doc)
	got := marshalDoc(t, doc)
	want := docJSON(paraJSON(textJSON("@admin")) + "," + paraJSON(textJSON("#3")))
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestConvertUntrustedImageToken(t *testing.T) {
	var hashes []string
	conv := defaultConverter()
	conv.Images = func(h []string) map[string]imageclient.ImageMeta {
		hashes = append(hashes, h...)
		return map[string]imageclient.ImageMeta{}
	}
	conv.Users = func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
		t.Fatalf("Users called with %v", ids)
		return nil, nil
	}
	doc, err := conv.ConvertUntrusted(context.Background(), "![](/image/"+testHash+")")
	if err != nil {
		t.Fatal(err)
	}
	checkDocument(t, doc)
	main := "https://cdn.example/aa/aa/" + testHash + ".webp"
	nullRec := `{"url":` + quoteJSON(main) + `,"hash":` + quoteJSON(testHash) + `,"width":null,"height":null,"thumbhash":null,"sexual":null}`
	img := `{"object":"image","url":` + quoteJSON(main) + `,"alt":"","image":` + nullRec + `,"is_sticker":false}`
	if got := marshalDoc(t, doc); got != docJSON(paraJSON(img)) {
		t.Errorf("got %s", got)
	}
	if len(hashes) != 1 || hashes[0] != testHash {
		t.Errorf("hashes %v, want [%s]", hashes, testHash)
	}
}
