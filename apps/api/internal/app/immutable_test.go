package app

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRequestStringsSurviveTheNextRequest(t *testing.T) {
	f := newFiber()
	var kept []string
	f.Get("/", func(c fiber.Ctx) error {
		kept = append(kept, c.Query("face"), c.Cookies("sid"), c.Get("X-Key"))
		return c.SendStatus(fiber.StatusNoContent)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = f.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = f.Shutdown() })

	client := &http.Client{}
	for i := range 3 {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/?face=face%d", ln.Addr(), i), nil)
		req.Header.Set("Cookie", fmt.Sprintf("sid=session%d", i))
		req.Header.Set("X-Key", fmt.Sprintf("key%d", i))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	for i := range 3 {
		want := []string{fmt.Sprintf("face%d", i), fmt.Sprintf("session%d", i), fmt.Sprintf("key%d", i)}
		for j, w := range want {
			if got := kept[i*3+j]; got != w {
				t.Errorf("request %d value %d = %q after later requests, want %q", i, j, got, w)
			}
		}
	}
}
