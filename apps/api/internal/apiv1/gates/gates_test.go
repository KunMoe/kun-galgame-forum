package gates_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/gates"
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

func spec(t *testing.T, registrars ...func(huma.API)) *huma.OpenAPI {
	t.Helper()
	return apiv1.Setup(fiber.New(), apiv1.Deps{}, registrars...).OpenAPI()
}

func get[B any](path string) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{
			OperationID: "get" + strings.ReplaceAll(path, "/", "-"),
			Method:      http.MethodGet,
			Path:        path,
			Summary:     "Probe",
		}), func(context.Context, *struct{}) (*struct{ Body B }, error) { return nil, nil })
	}
}

func expect(t *testing.T, errs []string, want string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e, want) {
			return
		}
	}
	t.Fatalf("no violation containing %q in %q", want, errs)
}

func TestTheProbeDocumentPassesEveryGate(t *testing.T) {
	type thing struct {
		ID       repr.DecimalID `json:"id" doc:"Thing id."`
		Author   repr.UserRef   `json:"author" doc:"Author."`
		Cover    *repr.Image    `json:"cover" doc:"Cover. null when there is none."`
		IsPinned bool           `json:"is_pinned" doc:"Whether the thing is pinned."`
	}
	if errs := gates.CheckAll(spec(t, get[thing]("/things"))); len(errs) > 0 {
		t.Fatalf("a well-formed document failed:\n%s", strings.Join(errs, "\n"))
	}
}

func TestG2MissingSummary(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{OperationID: "bare", Method: http.MethodGet, Path: "/bare"}),
			func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
	})
	expect(t, gates.CheckG2(doc), "GET /bare has no description or summary")
}

func TestG2MissingPropertyDescription(t *testing.T) {
	expect(t, gates.CheckG2(spec(t, get[struct {
		Name string `json:"name" maxLength:"8" pattern:"^a$"`
	}]("/x"))), "property name has no description")
}

func TestG3OpenEnumWithoutVocabulary(t *testing.T) {
	doc := spec(t)
	doc.Components.Schemas.Map()["Thing"] = &huma.Schema{Type: huma.TypeObject, Properties: map[string]*huma.Schema{
		"source": {Type: huma.TypeString, Enum: []any{"a"}, Extensions: map[string]any{"x-vocabulary-closed": false}},
	}}
	expect(t, gates.CheckG3(doc), "schema Thing.source open enum has no x-vocabulary")
}

func TestG3EnumWithoutClosedMarker(t *testing.T) {
	doc := spec(t)
	doc.Components.Schemas.Map()["Thing"] = &huma.Schema{Type: huma.TypeObject, Properties: map[string]*huma.Schema{
		"state": {Type: huma.TypeString, Enum: []any{"open"}},
	}}
	expect(t, gates.CheckG3(doc), "schema Thing.state enum has no x-vocabulary-closed")
}

func TestG4RequiredOperationMissing401(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Required(huma.Operation{OperationID: "secret", Method: http.MethodGet, Path: "/secret", Summary: "Secret"}),
			func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
	})
	delete(doc.Paths["/secret"].Get.Responses, "401")
	expect(t, gates.CheckG4(doc), "GET /secret does not declare 401")
}

func TestG4ErrorResponseThatIsNotAProblem(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{
			OperationID: "odd", Method: http.MethodGet, Path: "/odd", Summary: "Odd",
			Responses: map[string]*huma.Response{"409": {Description: "Conflict", Content: map[string]*huma.MediaType{
				"application/json": {Schema: &huma.Schema{Type: huma.TypeObject}},
			}}},
		}), func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
	})
	expect(t, gates.CheckG4(doc), "GET /odd response 409 is not application/problem+json")
}

func TestG4ProblemMediaWithTheWrongSchema(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{
			OperationID: "odd", Method: http.MethodGet, Path: "/odd", Summary: "Odd",
			Responses: map[string]*huma.Response{"409": {Description: "Conflict", Content: map[string]*huma.MediaType{
				"application/problem+json": {Schema: &huma.Schema{Ref: "#/components/schemas/FieldError"}},
			}}},
		}), func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
	})
	expect(t, gates.CheckG4(doc), "GET /odd response 409 is not application/problem+json with $ref Problem")
}

func declaring(status, description string) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{
			OperationID: "odd", Method: http.MethodGet, Path: "/odd", Summary: "Odd",
			Responses: map[string]*huma.Response{status: {Description: description, Content: map[string]*huma.MediaType{
				"application/problem+json": {Schema: &huma.Schema{Ref: apiv1.ProblemRef}},
			}}},
		}), func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
	}
}

func TestG4StatusTheOperationCannotDerive(t *testing.T) {
	expect(t, gates.CheckG4(spec(t, declaring("404", "Not Found"))),
		"GET /odd response 404 is not derived from the operation and names no registry code with that status")
}

func TestG4StatusNamedByItsCodePasses(t *testing.T) {
	if errs := gates.CheckG4(spec(t, declaring("404", "NOT_FOUND when the odd thing is gone."))); len(errs) > 0 {
		t.Fatalf("a named status failed: %q", errs)
	}
}

func TestG4DescriptionNamesACodeOfAnotherStatus(t *testing.T) {
	expect(t, gates.CheckG4(spec(t, declaring("404", "NOT_FOUND or INVALID_PARAMETER."))),
		"GET /odd response 404 names INVALID_PARAMETER, whose status is 400")
}

func TestG4DescriptionNamesAnUnknownCode(t *testing.T) {
	expect(t, gates.CheckG4(spec(t, declaring("404", "NOT_FOUND or TOPIC_GONE."))),
		"GET /odd response 404 names TOPIC_GONE, which is not in the code registry")
}

func TestG4CatchAllResponse(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, huma.Operation{OperationID: "untiered", Method: http.MethodGet, Path: "/untiered", Summary: "Untiered"},
			func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })
	})
	expect(t, gates.CheckG4(doc), "GET /untiered response default is a catch-all")
}

func TestG6EnvelopeKey(t *testing.T) {
	expect(t, gates.CheckG6(spec(t, get[struct {
		Code int `json:"code" minimum:"0" doc:"Legacy code."`
	}]("/x"))), "has envelope key code")
}

func TestG7NumericID(t *testing.T) {
	expect(t, gates.CheckG7(spec(t, get[struct {
		TopicID int `json:"topic_id" minimum:"1" doc:"Topic."`
	}]("/x"))), ".topic_id must be a JSON string")
}

func TestG7IDList(t *testing.T) {
	expect(t, gates.CheckG7(spec(t, get[struct {
		TagIDs []int `json:"tag_ids" maxItems:"4" doc:"Tags."`
	}]("/x"))), ".tag_ids must be an array of strings")
}

func TestG8ForbiddenName(t *testing.T) {
	expect(t, gates.CheckG8(spec(t, get[struct {
		Gid repr.DecimalID `json:"gid" doc:"Galgame."`
	}]("/x"))), ".gid is a forbidden name")
}

func TestG8UserIsAForbiddenName(t *testing.T) {
	expect(t, gates.CheckG8(spec(t, get[struct {
		User repr.UserRef `json:"user" doc:"Author."`
	}]("/x"))), ".user is a forbidden name")
}

func TestG8SameNameDifferentObject(t *testing.T) {
	doc := spec(t,
		get[struct {
			Author repr.UserRef `json:"author" doc:"Author."`
		}]("/a"),
		get[struct {
			Author repr.Image `json:"author" doc:"Author."`
		}]("/b"),
	)
	expect(t, gates.CheckG8(doc), "property author has diverging schemas")
}

func TestG8SameNameDifferentNullability(t *testing.T) {
	doc := spec(t,
		get[struct {
			Cover repr.Image `json:"cover" doc:"Cover."`
		}]("/a"),
		get[struct {
			Cover *repr.Image `json:"cover" doc:"Cover or null."`
		}]("/b"),
	)
	expect(t, gates.CheckG8(doc), "property cover has diverging schemas")
}

func TestG8SameNameDifferentScalarNullability(t *testing.T) {
	doc := spec(t,
		get[struct {
			EditedAt repr.DateTime `json:"edited_at" doc:"Edited."`
		}]("/a"),
		get[struct {
			EditedAt *repr.DateTime `json:"edited_at" doc:"Edited, or null."`
		}]("/b"),
	)
	expect(t, gates.CheckG8(doc), "property edited_at has diverging schemas")
}

func TestG9NullableArray(t *testing.T) {
	expect(t, gates.CheckG9(spec(t, get[struct {
		Tags []string `json:"tags" nullable:"true" maxItems:"4" doc:"Tags."`
	}]("/x"))), "array or map schema is nullable")
}

func TestG9ClosedObject(t *testing.T) {
	expect(t, gates.CheckG9(spec(t, get[struct {
		_    struct{} `additionalProperties:"false"`
		Name string   `json:"name" maxLength:"8" pattern:"^a$" doc:"Name."`
	}]("/x"))), "additionalProperties: false")
}

func TestG9NullableObjectInARequestBody(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Required(huma.Operation{OperationID: "post", Method: http.MethodPost, Path: "/things", Summary: "Post"}),
			func(context.Context, *struct {
				Body struct {
					Cover *repr.Image `json:"cover" doc:"Cover."`
				}
			}) (*struct{}, error) {
				return nil, nil
			})
	})
	expect(t, gates.CheckG9(doc), "nullable object in a request body")
}

func TestG14BareString(t *testing.T) {
	errs := gates.CheckG14(spec(t, get[struct {
		Title string `json:"title" doc:"Title."`
	}]("/x")))
	expect(t, errs, ".title string has no maxLength")
	expect(t, errs, ".title string has no enum, format, pattern, or free-text marker")
}

func TestG14UnboundedNumber(t *testing.T) {
	expect(t, gates.CheckG14(spec(t, get[struct {
		Score float64 `json:"score" doc:"Score."`
	}]("/x"))), ".score number has no minimum")
}

func TestG16LooseRequestID(t *testing.T) {
	expect(t, gates.CheckG16(spec(t, get[struct {
		RequestID string `json:"request_id" maxLength:"30" pattern:"^req_.*$" doc:"Request."`
	}]("/x"))), "request_id pattern")
}

func TestG16CursorWithoutPrefix(t *testing.T) {
	expect(t, gates.CheckG16(spec(t, get[struct {
		NextCursor *string `json:"next_cursor,omitempty" maxLength:"512" pattern:"^[a-z]+$" doc:"Next."`
	}]("/x"))), "next_cursor pattern")
}

func TestG17(t *testing.T) {
	put := func(path string) func(huma.API) {
		return func(api huma.API) {
			huma.Register(api, apiv1.Required(huma.Operation{OperationID: "put" + strings.ReplaceAll(path, "/", "-"), Method: http.MethodPut, Path: path, Summary: "Put"}),
				func(context.Context, *struct {
					ID repr.DecimalID `path:"thing_id" doc:"Thing."`
				}) (*struct{}, error) {
					return nil, nil
				})
		}
	}
	type thing struct {
		ID repr.DecimalID `json:"id" doc:"Thing id."`
	}
	t.Run("an unreadable target", func(t *testing.T) {
		expect(t, gates.CheckG17(spec(t, put("/things/{thing_id}/like"))), "addresses {thing_id}, but GET /things/{thing_id} has no 200 object with an id")
	})
	t.Run("a read without an id", func(t *testing.T) {
		doc := spec(t, put("/things/{thing_id}"), get[struct {
			Name string `json:"name" maxLength:"8" pattern:"^a$" doc:"Name."`
		}]("/things/{thing_id}"))
		expect(t, gates.CheckG17(doc), "GET /things/{thing_id} has no 200 object with an id")
	})
	t.Run("a readable target", func(t *testing.T) {
		if errs := gates.CheckG17(spec(t, put("/things/{thing_id}/like"), get[thing]("/things/{thing_id}"))); len(errs) > 0 {
			t.Fatal(errs)
		}
	})
}

func TestF1(t *testing.T) {
	cases := []struct {
		name string
		doc  func(t *testing.T) *huma.OpenAPI
		want string
	}{
		{"boolean without a prefix", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				Liked bool `json:"liked" doc:"Liked."`
			}]("/x"))
		}, ".liked boolean must start with"},
		{"timestamp not ending in _at", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				Created repr.DateTime `json:"create_time" doc:"Created."`
			}]("/x"))
		}, ".create_time a name ending in _at and format date-time go together"},
		{"a count that can go negative", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				ReplyCount int `json:"reply_count" minimum:"-1" doc:"Replies."`
			}]("/x"))
		}, ".reply_count must be an integer with minimum 0"},
		{"an enum value that is not snake_case", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				Sort string `json:"sort" enum:"newest,Hot" doc:"Sort."`
			}]("/x"))
		}, `enum value "Hot" is not snake_case`},
		{"a language tag in a property that is not a language", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				Region string `json:"region" enum:"zh-cn,ja-jp" maxLength:"5" doc:"Region."`
			}]("/x"))
		}, `enum value "zh-cn" is not snake_case`},
		{"a language property whose value is not a lowercase tag", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				Languages []string `json:"resource_languages" enum:"zh-cn,zh_TW" maxItems:"2" doc:"Languages."`
			}]("/x"))
		}, `enum value "zh_TW" is not a lowercase BCP 47 tag`},
		{"an include_ property", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				IncludeTotal bool `json:"include_total" doc:"Included."`
			}]("/x"))
		}, ".include_total boolean must start with"},
		{"an array of enum values that are not snake_case", func(t *testing.T) *huma.OpenAPI {
			return spec(t, get[struct {
				Badges []string `json:"badges" enum:"new,Hot" maxItems:"4" doc:"Badges."`
			}]("/x"))
		}, `enum value "Hot" is not snake_case`},
		{"a query flag without a prefix", func(t *testing.T) *huma.OpenAPI {
			return spec(t, func(api huma.API) {
				huma.Register(api, apiv1.Public(huma.Operation{OperationID: "q", Method: http.MethodGet, Path: "/q", Summary: "Q"}),
					func(context.Context, *struct {
						Liked bool `query:"liked" doc:"Liked."`
					}) (*struct{}, error) {
						return nil, nil
					})
			})
		}, "param.liked boolean must start with"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { expect(t, gates.CheckF1(c.doc(t)), c.want) })
	}
}

func TestF1AcceptsLanguageTags(t *testing.T) {
	doc := spec(t, get[struct {
		Language  string   `json:"language" enum:"zh-cn,zh-tw,ja-jp,en-us,others" maxLength:"6" doc:"Language."`
		Languages []string `json:"languages" enum:"zh-cn,other" maxItems:"2" doc:"Languages."`
	}]("/x"))
	if errs := gates.CheckF1(doc); len(errs) > 0 {
		t.Fatal(errs)
	}
}

func TestF1AcceptsAnIncludeFlag(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{OperationID: "q", Method: http.MethodGet, Path: "/q", Summary: "Q"}),
			func(context.Context, *struct {
				IncludeTotal bool `query:"include_total" doc:"Include total."`
			}) (*struct{}, error) {
				return nil, nil
			})
	})
	if errs := gates.CheckF1(doc); len(errs) > 0 {
		t.Fatal(errs)
	}
}

type countedIn struct {
	collect.Total
}

func listing[In, B any](api huma.API) {
	huma.Register(api, apiv1.Public(huma.Operation{OperationID: "listThings", Method: http.MethodGet, Path: "/things", Summary: "Things"}),
		func(context.Context, *In) (*struct{ Body B }, error) { return nil, nil })
}

func TestF9TotalWithoutIncludeTotal(t *testing.T) {
	expect(t, gates.CheckF9(spec(t, listing[struct{}, repr.CountedList[repr.UserRef]])),
		"GET /things 200 application/json declares total but the operation has no include_total parameter")
}

func TestF9IncludeTotalWithoutTotal(t *testing.T) {
	expect(t, gates.CheckF9(spec(t, listing[countedIn, repr.List[repr.UserRef]])),
		"GET /things 200 application/json does not declare total but the operation accepts include_total")
}

func TestF9PairedTotalPasses(t *testing.T) {
	if errs := gates.CheckAll(spec(t, listing[countedIn, repr.CountedList[repr.UserRef]])); len(errs) > 0 {
		t.Fatalf("a counted collection failed:\n%s", strings.Join(errs, "\n"))
	}
	if errs := gates.CheckAll(spec(t, listing[struct{}, repr.List[repr.UserRef]])); len(errs) > 0 {
		t.Fatalf("an uncounted collection failed:\n%s", strings.Join(errs, "\n"))
	}
}

type pagedIn struct {
	collect.PageNumber
}

type pagedCountedIn struct {
	collect.PageNumber
	collect.Total
}

func TestF9PageNumberCollectionPasses(t *testing.T) {
	if errs := gates.CheckAll(spec(t, listing[pagedIn, repr.PageList[repr.UserRef]])); len(errs) > 0 {
		t.Fatalf("a page-number collection failed:\n%s", strings.Join(errs, "\n"))
	}
}

func TestF9PageNumberCollectionWithoutPage(t *testing.T) {
	expect(t, gates.CheckF9(spec(t, listing[struct{}, repr.PageList[repr.UserRef]])),
		"GET /things 200 application/json is a page-number collection but the operation has no page parameter")
}

func TestF9PageNumberCollectionWithIncludeTotal(t *testing.T) {
	expect(t, gates.CheckF9(spec(t, listing[pagedCountedIn, repr.PageList[repr.UserRef]])),
		"GET /things 200 application/json is a page-number collection, which always sends total, but the operation accepts include_total")
}

func TestClosedEnumSetsExtension(t *testing.T) {
	s := repr.ClosedEnum("safe", "suggestive", "explicit")
	if s.Extensions["x-vocabulary-closed"] != true {
		t.Fatalf("closed %v", s.Extensions)
	}
	open := repr.OpenEnum("sources", 64)
	if open.Extensions["x-vocabulary-closed"] != false || open.Extensions["x-vocabulary"] != "sources" {
		t.Fatalf("open %v", open.Extensions)
	}
}

func TestG13FieldErrorWithACode(t *testing.T) {
	doc := spec(t)
	doc.Components.Schemas.Map()["FieldError"].Properties["code"] = &huma.Schema{Type: huma.TypeString}
	expect(t, gates.CheckG5G13(doc), "FieldError schema has a code property")
}

type unexportedPath struct {
	ThingID string `path:"thing_id" pattern:"^[0-9]+$" maxLength:"19" doc:"Thing id."`
}

func TestF10PathVariableWithoutParameter(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{OperationID: "getThing", Method: http.MethodGet, Path: "/things/{thing_id}", Summary: "Get"}),
			func(context.Context, *struct{ unexportedPath }) (*struct{}, error) {
				return nil, nil
			})
	})
	expect(t, gates.CheckF10(doc), "F10: GET /things/{thing_id} has no path parameter thing_id")
}

func TestF10DeclaredPathParameterPasses(t *testing.T) {
	doc := spec(t, func(api huma.API) {
		huma.Register(api, apiv1.Public(huma.Operation{OperationID: "getThing", Method: http.MethodGet, Path: "/things/{thing_id}", Summary: "Get"}),
			func(context.Context, *struct {
				ThingID string `path:"thing_id" pattern:"^[0-9]+$" maxLength:"19" doc:"Thing id."`
			}) (*struct{}, error) {
				return nil, nil
			})
	})
	if errs := gates.CheckF10(doc); len(errs) > 0 {
		t.Fatalf("declared path parameter flagged: %v", errs)
	}
}
