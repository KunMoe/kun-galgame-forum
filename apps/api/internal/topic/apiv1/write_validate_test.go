package apiv1

import (
	"reflect"
	"testing"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/model"
)

func TestAccessErrors(t *testing.T) {
	roles := []AccessRole{"creator"}
	users := []repr.DecimalID{"2"}
	cases := []struct {
		name   string
		fields accessFields
		want   []string
	}{
		{"public clean", accessFields{scope: "public"}, nil},
		{"login with roles", accessFields{scope: "login", roles: &roles}, []string{"/access_roles"}},
		{"login with users", accessFields{scope: "login", users: &users}, []string{"/access_user_ids"}},
		{"public both", accessFields{scope: "public", roles: &roles, users: &users}, []string{"/access_roles", "/access_user_ids"}},
		{"role missing", accessFields{scope: "role"}, []string{"/access_roles"}},
		{"role with users", accessFields{scope: "role", roles: &roles, users: &users}, []string{"/access_user_ids"}},
		{"role only users", accessFields{scope: "role", users: &users}, []string{"/access_roles", "/access_user_ids"}},
		{"users missing", accessFields{scope: "users"}, []string{"/access_user_ids"}},
		{"users with roles", accessFields{scope: "users", roles: &roles, users: &users}, []string{"/access_roles"}},
		{"users only roles", accessFields{scope: "users", roles: &roles}, []string{"/access_user_ids", "/access_roles"}},
		{"role ok", accessFields{scope: "role", roles: &roles}, nil},
		{"users ok", accessFields{scope: "users", users: &users}, nil},
	}
	for _, c := range cases {
		got := accessErrors(c.fields)
		var pointers []string
		for _, f := range got {
			if f.Pointer != nil {
				pointers = append(pointers, *f.Pointer)
			}
		}
		if !reflect.DeepEqual(pointers, c.want) {
			t.Errorf("%s: got %v want %v", c.name, pointers, c.want)
		}
	}
}

func TestGrantsDropAuthorKeepOrder(t *testing.T) {
	users := []repr.DecimalID{"7", "3", "7", "9"}
	got := grantsFromAccess(accessFields{scope: "users", users: &users}, 7)
	want := []model.TopicAccessGrant{
		{SubjectType: "user", SubjectValue: "3"},
		{SubjectType: "user", SubjectValue: "9"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	onlyAuthor := []repr.DecimalID{"7"}
	if got := grantsFromAccess(accessFields{scope: "users", users: &onlyAuthor}, 7); len(got) != 0 {
		t.Fatalf("author-only grants %+v", got)
	}
}

func TestDeriveCovers(t *testing.T) {
	h1 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	h2 := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	body := "see /image/" + h1 + " then /image/" + h2 + " and again /image/" + h1
	got := deriveCovers(body)
	want := model.ImageTokens{"/image/" + h1, "/image/" + h2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("derived %+v", got)
	}
	empty := coversFromHashes(nil)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty hashes %+v", empty)
	}
	exact := coversFromHashes([]ImageHash{ImageHash(h2)})
	if !reflect.DeepEqual(exact, model.ImageTokens{"/image/" + h2}) {
		t.Fatalf("exact %+v", exact)
	}
}
