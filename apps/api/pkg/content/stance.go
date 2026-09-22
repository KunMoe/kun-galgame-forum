package content

import "github.com/gofiber/fiber/v3"

type Stance string

const (
	StanceHide Stance = "hide"
	StanceBlur Stance = "blur"
	StanceShow Stance = "show"
)

// Fold is the only place this repo turns the account's two content-preference
// claims into a stance. The upstream migration backfilled nsfw_display to
// 'blur' for every existing account while leaving adult_confirmed_at null, so
// folding on nsfw_display alone shows adult content to every reader who has
// never attested their age. Unknown values fold to hide.
func Fold(adultConfirmed bool, nsfwDisplay string) Stance {
	if !adultConfirmed {
		return StanceHide
	}
	switch Stance(nsfwDisplay) {
	case StanceBlur:
		return StanceBlur
	case StanceShow:
		return StanceShow
	default:
		return StanceHide
	}
}

func (s Stance) AllowsNSFW() bool { return s != StanceHide }

const stanceLocal = "contentStance"

// Attach records the account stance for this request. It is deliberately left
// off Bearer and anonymous requests: those two lanes still decide from the
// X-Kungal-Nsfw header and the KUNGalgameSettings cookie respectively.
func Attach(c fiber.Ctx, s Stance) { c.Locals(stanceLocal, s) }

func FromCtx(c fiber.Ctx) (Stance, bool) {
	s, ok := c.Locals(stanceLocal).(Stance)
	return s, ok
}
