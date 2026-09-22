package content

import "github.com/gofiber/fiber/v3"

type Stance string

const (
	StanceHide Stance = "hide"
	StanceBlur Stance = "blur"
	StanceShow Stance = "show"
)

func ParseStance(s string) Stance {
	switch Stance(s) {
	case StanceBlur:
		return StanceBlur
	case StanceShow:
		return StanceShow
	default:
		return StanceHide
	}
}

// Fold is the only place this repo turns the account's two content-preference
// claims into a stance. Upstream retired the age attestation on 2026-09-23 and
// adult_confirmed is constant true from then on, but the claim is still sent
// and the upstream formula still reads it, so this keeps folding on the pair
// rather than trusting nsfw_display alone. Unknown values fold to hide.
func Fold(adultConfirmed bool, nsfwDisplay string) Stance {
	if !adultConfirmed {
		return StanceHide
	}
	return ParseStance(nsfwDisplay)
}

func (s Stance) AllowsNSFW() bool { return s != StanceHide }

const stanceLocal = "contentStance"

// Attach records the account stance for this request. It is deliberately left
// off anonymous requests: that lane still decides from the KUNGalgameSettings
// cookie.
func Attach(c fiber.Ctx, s Stance) { c.Locals(stanceLocal, s) }

func FromCtx(c fiber.Ctx) (Stance, bool) {
	s, ok := c.Locals(stanceLocal).(Stance)
	return s, ok
}
