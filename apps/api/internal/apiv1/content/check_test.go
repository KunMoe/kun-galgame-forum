package content_test

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/app"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	schemaOnce sync.Once
	schemaVal  *jsonschema.Schema
	schemaErr  error
)

func documentSchema(t testing.TB) *jsonschema.Schema {
	t.Helper()
	schemaOnce.Do(func() {
		raw, err := apiv1.MarshalOpenAPI(app.V1Spec())
		if err != nil {
			schemaErr = err
			return
		}
		decoded, err := jsonschema.UnmarshalJSON(strings.NewReader(string(raw)))
		if err != nil {
			schemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		c.DefaultDraft(jsonschema.Draft2020)
		c.AssertFormat()
		if err := c.AddResource("https://kungal.local/openapi.json", decoded); err != nil {
			schemaErr = err
			return
		}
		schemaVal, schemaErr = c.Compile("https://kungal.local/openapi.json#/components/schemas/ContentDocument")
	})
	if schemaErr != nil {
		t.Fatal(schemaErr)
	}
	return schemaVal
}

func checkDocument(t testing.TB, doc content.ContentDocument) {
	t.Helper()
	for _, issue := range documentIssues(doc) {
		t.Error(issue)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if err := documentSchema(t).Validate(v); err != nil {
		t.Errorf("schema: %v\n json: %s", err, raw)
	}
}

func documentIssues(doc content.ContentDocument) []string {
	var issues []string
	if doc.Object != "document" {
		issues = append(issues, "document.object="+doc.Object)
	}
	checkBlocks(&issues, "/children", doc.Children)
	return issues
}

func checkBlocks(issues *[]string, path string, blocks content.Blocks) {
	for i, n := range blocks {
		p := path + "/" + strconv.Itoa(i)
		switch x := n.(type) {
		case content.ParagraphNode:
			checkInlines(issues, p+"/children", x.Children)
		case content.HeadingNode:
			if x.Depth < 2 || x.Depth > 6 {
				*issues = append(*issues, p+" heading.depth")
			}
			if x.Anchor == "" || strings.ContainsAny(x.Anchor, " \t\n") || utf8.RuneCountInString(x.Anchor) > 128 {
				*issues = append(*issues, p+" heading.anchor")
			}
			checkInlines(issues, p+"/children", x.Children)
		case content.ThematicBreakNode:
		case content.BlockquoteNode:
			checkBlocks(issues, p+"/children", x.Children)
		case content.ListNode:
			if x.Start != nil && *x.Start < 0 {
				*issues = append(*issues, p+" list.start")
			}
			for j, item := range x.Children {
				checkBlocks(issues, p+"/children/"+strconv.Itoa(j)+"/children", item.Children)
			}
		case content.CodeNode:
			checkValue(issues, p+"/value", x.Value)
			if x.Lang != nil && (*x.Lang == "" || utf8.RuneCountInString(*x.Lang) > 32 || strings.ContainsAny(*x.Lang, " \t\n")) {
				*issues = append(*issues, p+" code.lang")
			}
		case content.MathNode:
			checkValue(issues, p+"/value", x.Value)
		case content.TableNode:
			for j, row := range x.Children {
				for k, cell := range row.Children {
					checkInlines(issues, p+"/children/"+strconv.Itoa(j)+"/children/"+strconv.Itoa(k)+"/children", cell.Children)
				}
			}
		case content.SpoilerNode:
			checkBlocks(issues, p+"/children", x.Children)
		case content.EntityCardNode:
			para, ok := onlyChild(x.Children).(content.ParagraphNode)
			if !ok {
				*issues = append(*issues, p+" entity_card.children")
				break
			}
			if _, ok := onlyChild(para.Children).(content.LinkNode); !ok {
				*issues = append(*issues, p+" entity_card.children")
			}
			checkBlocks(issues, p+"/children", x.Children)
		default:
			*issues = append(*issues, p+" unknown block")
		}
	}
}

func checkInlines(issues *[]string, path string, in content.Inlines) {
	prevText := false
	for i, n := range in {
		p := path + "/" + strconv.Itoa(i)
		switch x := n.(type) {
		case content.TextNode:
			if x.Value == "" {
				*issues = append(*issues, p+" empty text")
			}
			if prevText {
				*issues = append(*issues, p+" adjacent text")
			}
			prevText = true
			checkValue(issues, p+"/value", x.Value)
		case content.EmphasisNode:
			prevText = false
			checkInlines(issues, p+"/children", x.Children)
		case content.StrongNode:
			prevText = false
			checkInlines(issues, p+"/children", x.Children)
		case content.StrikethroughNode:
			prevText = false
			checkInlines(issues, p+"/children", x.Children)
		case content.InlineCodeNode:
			prevText = false
			checkValue(issues, p+"/value", x.Value)
		case content.InlineMathNode:
			prevText = false
			checkValue(issues, p+"/value", x.Value)
		case content.BreakNode:
			prevText = false
		case content.LinkNode:
			prevText = false
			checkURL(issues, p+"/url", x.URL)
			checkInlines(issues, p+"/children", x.Children)
		case content.ImageNode:
			prevText = false
			checkURL(issues, p+"/url", x.URL)
			if utf8.RuneCountInString(x.Alt) > 512 {
				*issues = append(*issues, p+" image.alt")
			}
		case content.VideoNode:
			prevText = false
			checkURL(issues, p+"/url", x.URL)
		case content.InlineSpoilerNode:
			prevText = false
			checkInlines(issues, p+"/children", x.Children)
		case content.MentionNode:
			prevText = false
		case content.ReplyReferenceNode:
			prevText = false
			if x.Floor < 1 {
				*issues = append(*issues, p+" reply_reference.floor")
			}
		case content.EntityLinkNode:
			prevText = false
			if _, ok := onlyChild(x.Children).(content.LinkNode); !ok {
				*issues = append(*issues, p+" entity_link.children")
			}
			checkInlines(issues, p+"/children", x.Children)
		default:
			prevText = false
			*issues = append(*issues, p+" unknown inline")
		}
	}
}

func checkValue(issues *[]string, path, v string) {
	if utf8.RuneCountInString(v) > 100007 {
		*issues = append(*issues, path+" longer than 100007")
	}
}

func checkURL(issues *[]string, path, s string) {
	if utf8.RuneCountInString(s) > 2048 {
		*issues = append(*issues, path+" longer than 2048")
		return
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" {
		*issues = append(*issues, path+" not a URI: "+s)
		return
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto":
	default:
		*issues = append(*issues, path+" scheme "+u.Scheme)
	}
}

func onlyChild[T any](s []T) any {
	if len(s) != 1 {
		return nil
	}
	return s[0]
}
