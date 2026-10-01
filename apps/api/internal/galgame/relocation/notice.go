package relocation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/message/notifytype"
	topicModel "kun-galgame-api/internal/topic/model"
	topicRepo "kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

const (
	letmoeOrigin = "https://www.letmoe.com"
	// message.content is varchar(233), and the longest sentence around the name
	// runs to 184 runes.
	noticeNameLimit = 40

	announcementCategory = "others"
	announcementSection  = "o-forum"
)

var ErrAnnounced = errors.New("relocation: this author already has a topic with that title")

type Recipient struct {
	ResourceID        int   `gorm:"column:resource_id"`
	WorkID            int   `gorm:"column:work_id"`
	DestinationID     int64 `gorm:"column:destination_id"`
	DestinationPublic bool  `gorm:"column:destination_public"`
	ReceiverID        int   `gorm:"column:receiver_id"`
	Liked             bool  `gorm:"column:liked"`
}

// LetMoe shows a held-back resource (its link was reported dead here) to its
// uploader alone; everyone else gets a 404 there, so they are sent to the game.
func (m Recipient) URL() string {
	if m.DestinationPublic || !m.Liked {
		return letmoeOrigin + "/resource/" + strconv.FormatInt(m.DestinationID, 10)
	}
	return letmoeOrigin + "/game/" + strconv.Itoa(m.WorkID)
}

// Path is the resource's old page: it answers with a redirect to URL, so the
// notification opens the new address and stays an in-site path.
func (m Recipient) Path() string {
	return "/galgame/resource/" + strconv.Itoa(m.ResourceID)
}

func (s *Store) Recipients() ([]Recipient, error) {
	var rows []Recipient
	err := s.db.Raw(`SELECT m.resource_id, m.work_id, m.destination_id, m.destination_public,
			m.uploader_id AS receiver_id, false AS liked
		FROM galgame_resource_relocation m
		WHERE m.retired_at IS NOT NULL AND m.destination_id IS NOT NULL
		UNION
		SELECT m.resource_id, m.work_id, m.destination_id, m.destination_public,
			(k->>'user_id')::int, true
		FROM galgame_resource_relocation m, jsonb_array_elements(m.payload->'likes') k
		WHERE m.retired_at IS NOT NULL AND m.destination_id IS NOT NULL
		  AND (k->>'user_id')::int <> m.uploader_id
		ORDER BY resource_id, liked, receiver_id`).Scan(&rows).Error
	return rows, err
}

const noticeReason = "鲲 Galgame 论坛今后只负责 Galgame，3D SLG 等同人游戏的资源改由 一起萌·LetMoe（letmoe.com）负责。"

func Notice(workName string, m Recipient) string {
	name := []rune(strings.TrimSpace(workName))
	if len(name) > noticeNameLimit {
		name = append(name[:noticeNameLimit-1], '…')
	}
	subject := fmt.Sprintf("您发布的《%s》资源", string(name))
	if m.Liked {
		subject = fmt.Sprintf("您点赞过的《%s》资源", string(name))
	}
	var where string
	switch {
	case m.DestinationPublic && m.Liked:
		where = "新地址：" + m.URL() + " ，您的点赞已转为那边的收藏。"
	case m.DestinationPublic:
		where = "新地址：" + m.URL() + " 。"
	case m.Liked:
		where = "它的链接此前被报告失效，发布者修复并通过审核后会重新公开，游戏页：" + m.URL() + " 。"
	default:
		where = "它的链接此前被报告失效，请登录 LetMoe 打开 " + m.URL() + " 修改链接，保存后重新送审，通过后公开。"
	}
	return subject + "已搬迁到 一起萌·LetMoe，" + where + noticeReason
}

// Notify reports whether the notification was new.
func (s *Store) Notify(senderID int, m Recipient, content string) (bool, error) {
	if m.ReceiverID <= 0 || m.ReceiverID == senderID {
		return false, nil
	}
	kind := notifytype.ToDB(notifytype.ResourceRelocated)
	var existing int64
	err := s.db.Model(&msgModel.Message{}).
		Where("receiver_id = ? AND type = ? AND link = ?", m.ReceiverID, kind, m.Path()).
		Count(&existing).Error
	if err != nil || existing > 0 {
		return false, err
	}
	return true, s.db.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: m.ReceiverID,
		Type:       kind,
		Content:    content,
		Link:       m.Path(),
	}).Error
}

// Announce posts the announcement as an ordinary topic of authorID. It goes
// around createTopic on purpose: that path spends the author's daily quota and
// pays a publish reward, and neither belongs to an operator's notice.
func (s *Store) Announce(authorID int, title, body string) (int, error) {
	title = strings.TrimSpace(title)
	body = markdown.NormalizeStoredContent(strings.TrimSpace(body))
	if title == "" || body == "" {
		return 0, errors.New("relocation: the announcement needs a title and a body")
	}
	var topicID int
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&topicModel.Topic{}).Where("user_id = ? AND title = ?", authorID, title).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return ErrAnnounced
		}
		topic := topicModel.Topic{
			AccessScope: "public",
			Title:       title,
			Content:     body,
			Category:    announcementCategory,
			UserID:      authorID,
			CoverImages: topicModel.ImageTokens{},
		}
		if err := tx.Create(&topic).Error; err != nil {
			return err
		}
		var section topicModel.TopicSection
		if err := tx.Where("name = ?", announcementSection).First(&section).Error; err != nil {
			return fmt.Errorf("topic section %q: %w", announcementSection, err)
		}
		rel := topicModel.TopicSectionRelation{TopicID: topic.ID, TopicSectionID: section.ID}
		if err := tx.Create(&rel).Error; err != nil {
			return err
		}
		topicID = topic.ID
		return topicRepo.WatchOwnTopic(tx, authorID, topic.ID)
	})
	return topicID, err
}
