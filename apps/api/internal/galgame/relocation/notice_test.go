package relocation

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

const announcer = 940_340_003

func TestNoticeFitsTheNotificationColumnWhateverTheWorkIsCalled(t *testing.T) {
	name := strings.Repeat("とても長い作品名", 40)
	for _, m := range []Recipient{
		{ResourceID: resEnglishA, WorkID: workEnglish, DestinationID: 9_000_000_000_001, DestinationPublic: true},
		{ResourceID: resEnglishA, WorkID: workEnglish, DestinationID: 9_000_000_000_001, DestinationPublic: true, Liked: true},
		{ResourceID: resEnglishA, WorkID: workEnglish, DestinationID: 9_000_000_000_001},
		{ResourceID: resEnglishA, WorkID: workEnglish, DestinationID: 9_000_000_000_001, Liked: true},
	} {
		got := Notice(name, m)
		if n := utf8.RuneCountInString(got); n > 233 {
			t.Errorf("public=%v liked=%v: %d runes, message.content holds 233", m.DestinationPublic, m.Liked, n)
		}
		if !strings.Contains(got, m.URL()+" ") {
			t.Errorf("public=%v liked=%v: %q does not carry %s followed by a space", m.DestinationPublic, m.Liked, got, m.URL())
		}
	}
}

func TestNoticeSendsOnlyTheUploaderToAResourceLetMoeHoldsBack(t *testing.T) {
	for _, c := range []struct {
		m    Recipient
		want string
	}{
		{Recipient{WorkID: 148, DestinationID: 7001, DestinationPublic: true}, "https://www.letmoe.com/resource/7001"},
		{Recipient{WorkID: 148, DestinationID: 7001, DestinationPublic: true, Liked: true}, "https://www.letmoe.com/resource/7001"},
		{Recipient{WorkID: 148, DestinationID: 7002}, "https://www.letmoe.com/resource/7002"},
		{Recipient{WorkID: 148, DestinationID: 7002, Liked: true}, "https://www.letmoe.com/game/148"},
	} {
		if got := c.m.URL(); got != c.want {
			t.Errorf("public=%v liked=%v: URL = %q, want %q", c.m.DestinationPublic, c.m.Liked, got, c.want)
		}
	}
}

func TestNotifyTellsTheUploaderAndTheLikersOfRetiredResourcesOnce(t *testing.T) {
	db, store := seed(t)
	clean := func() {
		db.Exec("DELETE FROM message WHERE receiver_id IN ?", []int{uploader, liker, announcer})
	}
	clean()
	t.Cleanup(clean)
	if err := db.Exec(`INSERT INTO galgame_resource_like (galgame_resource_id, user_id, updated) VALUES (?, ?, now())`,
		resEnglishA, uploader).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	for id, dest := range map[int]int64{resEnglishA: 7001, resEnglishB: 7002} {
		if err := store.SaveReceipt(id, dest, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Retire(resEnglishA); err != nil {
		t.Fatal(err)
	}

	recipients, err := store.Recipients()
	if err != nil {
		t.Fatal(err)
	}
	var mine []Recipient
	for _, r := range recipients {
		if r.ResourceID >= resEnglishA && r.ResourceID <= resKorean {
			mine = append(mine, r)
		}
	}
	if len(mine) != 2 || mine[0].ReceiverID != uploader || mine[0].Liked || mine[1].ReceiverID != liker || !mine[1].Liked {
		t.Fatalf("recipients = %+v, want the uploader then the liker of the retired resource; a resource still on the forum and the uploader's own like tell nobody", mine)
	}

	for round := 1; round <= 2; round++ {
		for _, r := range mine {
			created, err := store.Notify(announcer, r, Notice("Doki Doki", r))
			if err != nil {
				t.Fatal(err)
			}
			if created != (round == 1) {
				t.Errorf("round %d, receiver %d: created = %v", round, r.ReceiverID, created)
			}
		}
	}
	var rows []struct {
		ReceiverID int
		SenderID   int
		Type       string
		Link       string
		Content    string
		Status     string
	}
	db.Raw(`SELECT receiver_id, sender_id, type, link, content, status FROM message
		WHERE receiver_id IN ? ORDER BY receiver_id`, []int{uploader, liker}).Scan(&rows)
	if len(rows) != 2 {
		t.Fatalf("message rows = %d, want one per reader", len(rows))
	}
	for _, row := range rows {
		if row.SenderID != announcer || row.Type != "resource-relocated" || row.Status != "unread" ||
			row.Link != "/galgame/resource/2000340101" || !strings.Contains(row.Content, "https://www.letmoe.com/resource/7001") {
			t.Errorf("row = %+v", row)
		}
	}
	if !strings.Contains(rows[0].Content, "您发布的《Doki Doki》") || !strings.Contains(rows[1].Content, "您点赞过的《Doki Doki》") {
		t.Errorf("contents = %q / %q", rows[0].Content, rows[1].Content)
	}

	self := mine[0]
	self.ReceiverID = announcer
	if created, err := store.Notify(announcer, self, "x"); err != nil || created {
		t.Errorf("the sender notified itself: created=%v err=%v", created, err)
	}
}

func TestAnnouncePostsOneTopicInTheForumSection(t *testing.T) {
	db, store := seed(t)
	clean := func() {
		db.Exec("DELETE FROM topic WHERE user_id = ?", announcer)
	}
	clean()
	t.Cleanup(clean)
	if err := db.Exec(`INSERT INTO topic_section (name, updated)
		SELECT 'o-forum', now() WHERE NOT EXISTS (SELECT 1 FROM topic_section WHERE name = 'o-forum')`).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := store.Announce(announcer, "  ", "body"); err == nil {
		t.Error("an announcement without a title was posted")
	}
	id, err := store.Announce(announcer, " 资源搬迁公告 ", "正文\n\n第二段")
	if err != nil {
		t.Fatal(err)
	}
	var topic struct {
		Title       string
		Content     string
		Category    string
		AccessScope string
		Status      int
	}
	db.Raw(`SELECT title, content, category, access_scope, status FROM topic WHERE id = ?`, id).Scan(&topic)
	if topic.Title != "资源搬迁公告" || topic.Content != "正文\n\n第二段" || topic.Category != "others" || topic.AccessScope != "public" || topic.Status != 0 {
		t.Errorf("topic = %+v", topic)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM topic_section_relation r JOIN topic_section s ON s.id = r.topic_section_id
		WHERE r.topic_id = ? AND s.name = 'o-forum'`, id); n != 1 {
		t.Errorf("o-forum relations = %d, want 1", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM topic_subscription WHERE topic_id = ? AND user_id = ? AND notification_level = 'watching'`,
		id, announcer); n != 1 {
		t.Errorf("author watch rows = %d, want 1", n)
	}

	if _, err := store.Announce(announcer, "资源搬迁公告", "again"); !errors.Is(err, ErrAnnounced) {
		t.Errorf("second announce: err = %v, want ErrAnnounced", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM topic WHERE user_id = ?`, announcer); n != 1 {
		t.Errorf("topics by the announcer = %d, want 1", n)
	}
}
