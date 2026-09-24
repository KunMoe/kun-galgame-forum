package problem

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRequestIDFormat(t *testing.T) {
	id := NewRequestID()
	if !ValidRequestID(id) {
		t.Fatalf("minted id %q failed validation", id)
	}
	if !strings.HasPrefix(id, "req_") || len(id) != 4+26 {
		t.Fatalf("shape %q", id)
	}
	ulid := strings.TrimPrefix(id, "req_")
	for i := 0; i < len(ulid); i++ {
		c := ulid[i]
		ok := (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')
		if c == 'I' || c == 'L' || c == 'O' || c == 'U' {
			ok = false
		}
		if !ok {
			t.Fatalf("ulid %q has non-crockford char %q at %d", ulid, c, i)
		}
	}
}

func TestRequestIDReusesValidInbound(t *testing.T) {
	const id = "req_01ARZ3NDEKTSV4RRFFQ69G5FAV"
	app := fiber.New()
	var got string
	app.Get("/x", func(c fiber.Ctx) error {
		got = RequestID(c)
		return c.SendStatus(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, id)
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Fatalf("got %q want %q", got, id)
	}
}

func TestRequestIDReplacesInvalidInbound(t *testing.T) {
	app := fiber.New()
	var got string
	app.Get("/x", func(c fiber.Ctx) error {
		got = RequestID(c)
		return c.SendStatus(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, "req_not-a-valid-ulid-value!!")
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	if !ValidRequestID(got) || got == "req_not-a-valid-ulid-value!!" {
		t.Fatalf("invalid inbound was kept: %q", got)
	}
}

func TestTwoMintedIDsDiffer(t *testing.T) {
	a, b := NewRequestID(), NewRequestID()
	if a == b {
		t.Fatalf("two minted ids were equal: %s", a)
	}
}
