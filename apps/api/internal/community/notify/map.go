package notify

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/message/model"
	"kun-galgame-api/pkg/communityclient"
)

func MapNotification(n communityclient.NotificationView, post *communityclient.PostView, target *anchor.Target) *model.Message {
	if n.Kind == communityclient.InboxFollowed {
		return mapFollowed(n)
	}
	if n.Kind == communityclient.InboxFolloweeActivity {
		return mapFolloweeActivity(n)
	}
	if n.Kind == communityclient.InboxNotice {
		return mapNotice(n)
	}
	if target == nil {
		return nil
	}
	msgType, ok := mapKind(n.Kind, post)
	if !ok {
		return nil
	}

	link := target.Link
	if n.AnchorKind == communityclient.AnchorSiteGame && n.PostID != nil && *n.PostID > 0 {
		link = "/galgame/" + n.AnchorID + "?comment=" + strconv.FormatInt(*n.PostID, 10)
	}
	if len(link) > 100 {
		link = link[:100]
	}

	// Quiz walls are spoiler-gated; a preview in the inbox would bypass the gate.
	preview := ""
	if post != nil && !strings.HasPrefix(n.AnchorID, "quiz:") {
		preview = content.PlainText(markdown.StripReferenceTokens(post.ContentRaw), constants.TextPreviewLength)
	}

	senderID := 0
	if n.ActorID != nil {
		senderID = int(*n.ActorID)
	}
	status := "unread"
	if n.ReadAt != nil && *n.ReadAt != "" {
		status = "read"
	}

	id := n.ID
	seq := n.Seq
	threadID := n.ThreadID
	msg := &model.Message{
		Content:                 preview,
		Link:                    link,
		Status:                  status,
		Type:                    msgType,
		SenderID:                senderID,
		ReceiverID:              int(n.UserID),
		CommunityNotificationID: &id,
		CommunitySeq:            &seq,
		CommunityThreadID:       &threadID,
		ItemCount:               int(n.ItemCount),
		ActorCount:              int(n.ActorCount),
		CreatedAt:               parseTime(n.UpdatedAt),
	}
	if n.PostNumber != nil {
		pn := int(*n.PostNumber)
		msg.CommunityPostNumber = &pn
	}
	return msg
}

func mapKind(kind int32, post *communityclient.PostView) (string, bool) {
	switch kind {
	case communityclient.InboxReplied:
		if post != nil && post.ReplyToPostID == 0 {
			return "commented", true
		}
		return "replied", true
	case communityclient.InboxMentioned:
		return "mentioned", true
	case communityclient.InboxPosted:
		return "followed", true
	case communityclient.InboxLiked:
		return "liked", true
	case communityclient.InboxFollowed:
		return "user-followed", true
	case communityclient.InboxFolloweeActivity:
		return "followee-activity", true
	case communityclient.InboxNotice:
		return "notice", true
	default:
		return "", false
	}
}

func mapFollowed(n communityclient.NotificationView) *model.Message {
	msgType, ok := mapKind(n.Kind, nil)
	if !ok {
		return nil
	}
	senderID := 0
	if n.ActorID != nil {
		senderID = int(*n.ActorID)
	}
	link := "/user/" + strconv.Itoa(senderID)
	if len(link) > 100 {
		link = link[:100]
	}
	status := "unread"
	if n.ReadAt != nil && *n.ReadAt != "" {
		status = "read"
	}
	id := n.ID
	seq := n.Seq
	return &model.Message{
		Link:                    link,
		Status:                  status,
		Type:                    msgType,
		SenderID:                senderID,
		ReceiverID:              int(n.UserID),
		CommunityNotificationID: &id,
		CommunitySeq:            &seq,
		ItemCount:               int(n.ItemCount),
		ActorCount:              int(n.ActorCount),
		CreatedAt:               parseTime(n.UpdatedAt),
	}
}

func mapFolloweeActivity(n communityclient.NotificationView) *model.Message {
	msgType, ok := mapKind(n.Kind, nil)
	if !ok {
		return nil
	}
	senderID := 0
	if n.ActorID != nil {
		senderID = int(*n.ActorID)
	}
	preview := ""
	if n.Activity != nil {
		preview = content.PlainText(n.Activity.Title, constants.TextPreviewLength)
	}
	status := "unread"
	if n.ReadAt != nil && *n.ReadAt != "" {
		status = "read"
	}
	id := n.ID
	seq := n.Seq
	return &model.Message{
		Content:                 preview,
		Link:                    followeeActivityLink(n, senderID),
		Status:                  status,
		Type:                    msgType,
		SenderID:                senderID,
		ReceiverID:              int(n.UserID),
		CommunityNotificationID: &id,
		CommunitySeq:            &seq,
		ItemCount:               int(n.ItemCount),
		ActorCount:              int(n.ActorCount),
		CreatedAt:               parseTime(n.UpdatedAt),
	}
}

func mapNotice(n communityclient.NotificationView) *model.Message {
	if n.Notice == nil {
		return nil
	}
	msgType, ok := mapKind(n.Kind, nil)
	if !ok {
		return nil
	}
	link, ok := inSitePath(n.Notice.URL)
	if !ok {
		link = noticeFallbackPath(n.Notice.Topic)
	}
	status := "unread"
	if n.ReadAt != nil && *n.ReadAt != "" {
		status = "read"
	}
	id := n.ID
	seq := n.Seq
	return &model.Message{
		Content:                 content.PlainText(n.Notice.Title, constants.TextPreviewLength),
		Link:                    link,
		Status:                  status,
		Type:                    msgType,
		ReceiverID:              int(n.UserID),
		CommunityNotificationID: &id,
		CommunitySeq:            &seq,
		ItemCount:               int(n.ItemCount),
		ActorCount:              int(n.ActorCount),
		CreatedAt:               parseTime(n.UpdatedAt),
	}
}

func noticeFallbackPath(topic string) string {
	if topic == "shop_restock" {
		return "/shop"
	}
	return "/message/notice"
}

func followeeActivityLink(n communityclient.NotificationView, senderID int) string {
	if n.Activity != nil {
		if link, ok := inSitePath(n.Activity.URL); ok {
			return link
		}
	}
	return "/user/" + strconv.Itoa(senderID)
}

func inSitePath(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(u.Hostname()) {
	case "www.kungal.com", "kungal.com":
	default:
		return "", false
	}
	link := u.EscapedPath()
	if link == "" {
		link = "/"
	}
	if u.RawQuery != "" {
		link += "?" + u.RawQuery
	}
	if len(link) > 100 {
		return "", false
	}
	return link, true
}

func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}
