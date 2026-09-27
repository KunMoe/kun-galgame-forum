package repr

import "kun-galgame-api/pkg/userclient"

type UserRef struct {
	Object      string       `json:"object" enum:"user" maxLength:"4" doc:"Type discriminant. Always user."`
	ID          DecimalID    `json:"id" doc:"User id. JSON string of a decimal integer."`
	Name        *string      `json:"name" maxLength:"64" doc:"Display name. null when the account no longer exists; show a localized label. Free text; never use it as a decision input."`
	Avatar      *Image       `json:"avatar" doc:"Avatar image. null when the account has no image-service hash."`
	AvatarFrame *AvatarFrame `json:"avatar_frame" doc:"The avatar frame the user wears on this forum; a change can take up to ten minutes to show. null when none."`
}

type AvatarFrame struct {
	StaticURL   string  `json:"static_url" format:"uri" maxLength:"512" doc:"Still PNG on a square canvas 1.2 times the avatar, the avatar circle centred in it and transparent. Draw it centred over the avatar, outside the layout and ignoring pointer events, and not below 32 px."`
	AnimatedURL *string `json:"animated_url" format:"uri" maxLength:"512" doc:"Animated WebP of the same size. Play it only while the avatar is hovered or focused, and never under a reduced-motion preference. null when the frame has no animated version."`
}

func NewAvatarFrame(d *userclient.Decoration) *AvatarFrame {
	if d == nil || d.StaticURL == "" {
		return nil
	}
	f := &AvatarFrame{StaticURL: d.StaticURL}
	if d.AnimatedURL != "" {
		animated := d.AnimatedURL
		f.AnimatedURL = &animated
	}
	return f
}

func NewUserRef(cdnBase string, u userclient.User) UserRef {
	name := u.Name
	return UserRef{
		Object:      "user",
		ID:          ID(u.ID),
		Name:        &name,
		Avatar:      NewImage(cdnBase, u.AvatarImageHash, nil),
		AvatarFrame: NewAvatarFrame(u.Cosmetics.AvatarFrame),
	}
}

// userclient.Placeholder gives a missing account a fixed Chinese display name,
// which a client that localizes cannot tell from a user who picked that name.
func DeletedUserRef(id int) UserRef {
	return UserRef{Object: "user", ID: ID(id)}
}
