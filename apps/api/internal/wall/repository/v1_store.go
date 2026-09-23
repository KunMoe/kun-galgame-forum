package repository

import (
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *gorm.DB { return s.db }

var ErrNotFound = errors.New("wall: row not found")

type Owned struct {
	UserID int `gorm:"column:user_id"`
}

func (s *Store) ownerOf(table string, id int) (int, error) {
	var rows []Owned
	if err := s.db.Table(table).Select("user_id").Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, ErrNotFound
	}
	return rows[0].UserID, nil
}

func (s *Store) RatingAuthor(id int) (int, error)  { return s.ownerOf("galgame_rating", id) }
func (s *Store) ResourceOwner(id int) (int, error) { return s.ownerOf("galgame_resource", id) }
func (s *Store) ToolsetOwner(id int) (int, error)  { return s.ownerOf("galgame_toolset", id) }

type Quiz struct {
	UserID       int    `gorm:"column:user_id"`
	HideGalgame  bool   `gorm:"column:hide_galgame"`
	SpoilerLevel string `gorm:"column:spoiler_level"`
}

func (s *Store) Quiz(id int) (*Quiz, error) {
	var rows []Quiz
	if err := s.db.Table("galgame_quiz").Select("user_id, hide_galgame, spoiler_level").
		Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

func (s *Store) HasAnswered(quizID, userID int) (bool, error) {
	var n int64
	err := s.db.Table("galgame_quiz_answer").Where("quiz_id = ? AND user_id = ?", quizID, userID).Limit(1).Count(&n).Error
	return n > 0, err
}

type Website struct {
	URL      string `gorm:"column:url"`
	AgeLimit string `gorm:"column:age_limit"`
}

func (s *Store) Website(id int) (*Website, error) {
	var rows []Website
	if err := s.db.Table("galgame_website").Select("url, age_limit").Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

func (s *Store) LikeCounts(postIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(postIDs))
	if len(postIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		PostID int64 `gorm:"column:post_id"`
		N      int   `gorm:"column:n"`
	}
	if err := s.db.Table("galgame_post_like").Select("post_id, COUNT(*) AS n").
		Where("post_id IN ?", postIDs).Group("post_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.PostID] = r.N
	}
	return out, nil
}

func (s *Store) LikedSet(userID int, postIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool, len(postIDs))
	if userID <= 0 || len(postIDs) == 0 {
		return out, nil
	}
	var ids []int64
	if err := s.db.Table("galgame_post_like").Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Pluck("post_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func (s *Store) EnsureLike(postID int64, userID int) error {
	return s.db.Exec(`INSERT INTO galgame_post_like (post_id, user_id, created) VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING`, postID, userID, time.Now()).Error
}

func (s *Store) RemoveLike(postID int64, userID int) error {
	return s.db.Exec(`DELETE FROM galgame_post_like WHERE post_id = ? AND user_id = ?`, postID, userID).Error
}

func (s *Store) BumpGalgame(galgameID, delta int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if delta > 0 {
			if err := tx.Exec(`INSERT INTO galgame (id, updated) VALUES (?, now()) ON CONFLICT (id) DO NOTHING`, galgameID).Error; err != nil {
				return err
			}
		}
		return tx.Exec(`UPDATE galgame SET comment_count = GREATEST(comment_count + ?, 0) WHERE id = ?`, delta, galgameID).Error
	})
}

func (s *Store) BumpCount(table string, id, delta int) error {
	return s.db.Table(table).Where("id = ?", id).
		Update("comment_count", gorm.Expr("GREATEST(comment_count + ?, 0)", delta)).Error
}

type Message struct {
	SenderID   int
	ReceiverID int
	Type       string
	Content    string
	Link       string
}

// InsertMessageOnce skips a notification identical to one already sent, as the
// legacy walls did: a commenter who repeats themselves notifies the owner once.
func (s *Store) InsertMessageOnce(m Message) error {
	return s.db.Exec(`INSERT INTO message (sender_id, receiver_id, type, content, link, status, created, updated)
		SELECT ?, ?, ?, ?, ?, 'unread', now(), now()
		WHERE NOT EXISTS (SELECT 1 FROM message WHERE sender_id = ? AND receiver_id = ? AND type = ? AND content = ? AND link = ?)`,
		m.SenderID, m.ReceiverID, m.Type, m.Content, m.Link,
		m.SenderID, m.ReceiverID, m.Type, m.Content, m.Link).Error
}

// The feed keys rows by an int4 source id; a community post id beyond it
// cannot be represented, so the row is skipped rather than truncated.
func (s *Store) FeedUpsert(feedType string, postID int64, userID, galgameID int, content, link string, nsfw bool, created time.Time) error {
	if postID > math.MaxInt32 {
		return nil
	}
	return s.db.Exec("SELECT feed_upsert(?, ?, ?, ?, ?, ?, ?, ?)",
		feedType, postID, userID, galgameID, content, link, nsfw, created).Error
}

func (s *Store) FeedDelete(feedType string, postID int64) error {
	if postID > math.MaxInt32 {
		return nil
	}
	return s.db.Exec("SELECT feed_delete(?, ?)", feedType, postID).Error
}

// Posts imported from the pre-community tables kept their old ids in the feed.
func (s *Store) LegacyFeedID(galgame bool, source string, postID int64) (int, error) {
	var ids []int
	var err error
	if galgame {
		err = s.db.Raw(`SELECT old_comment_id FROM galgame_comment_community_map WHERE post_id = ?`, postID).Scan(&ids).Error
	} else {
		err = s.db.Raw(`SELECT old_id FROM resource_comment_community_map WHERE source = ? AND post_id = ?`, source, postID).Scan(&ids).Error
	}
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	return ids[0], nil
}
