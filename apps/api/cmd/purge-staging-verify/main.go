package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/pkg/communityclient"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const stagingDBName = "kungalgame_purge_staging"

type colCheck struct {
	table, col string
	mode       string
	why        string
}

var colChecks = []colCheck{
	{"topic", "user_id", "zero", ""},
	{"topic_reply", "user_id", "zero", ""},
	{"topic_comment", "user_id", "zero", ""},
	{"topic_comment", "target_user_id", "keep", "NOT NULL; 3rd-party comment kept"},
	{"topic_like", "user_id", "zero", ""},
	{"topic_dislike", "user_id", "zero", ""},
	{"topic_favorite", "user_id", "zero", ""},
	{"topic_upvote", "user_id", "zero", ""},
	{"topic_comment_like", "user_id", "zero", ""},
	{"topic_reply_like", "user_id", "zero", ""},
	{"topic_reply_dislike", "user_id", "zero", ""},
	{"topic_poll", "user_id", "zero", ""},
	{"topic_poll_vote", "user_id", "zero", ""},
	{"topic_reaction", "user_id", "zero", ""},
	{"topic_reply_reaction", "user_id", "zero", ""},
	{"topic_draft", "user_id", "zero", ""},
	{"topic_lottery", "user_id", "zero", ""},
	{"topic_lottery_entry", "user_id", "zero", ""},
	{"galgame_post_like", "user_id", "zero", ""},
	{"galgame_favorite", "user_id", "zero", ""},
	{"galgame_like", "user_id", "zero", ""},
	{"galgame_rating", "user_id", "zero", ""},
	{"galgame_rating_like", "user_id", "zero", ""},
	{"galgame_resource", "user_id", "zero", ""},
	{"galgame_resource_like", "user_id", "zero", ""},
	{"galgame_toolset", "user_id", "zero", ""},
	{"galgame_toolset_contributor", "user_id", "zero", ""},
	{"galgame_toolset_practicality", "user_id", "zero", ""},
	{"galgame_toolset_resource", "user_id", "zero", ""},
	{"galgame_website", "user_id", "zero", ""},
	{"galgame_website_favorite", "user_id", "zero", ""},
	{"galgame_website_like", "user_id", "zero", ""},
	{"galgame_collection", "user_id", "zero", ""},
	{"galgame_collection_item", "user_id", "zero", ""},
	{"galgame_collection_viewer", "user_id", "zero", ""},
	{"galgame_quiz", "user_id", "zero", ""},
	{"galgame_quiz_answer", "user_id", "zero", ""},
	{"galgame_quiz_favorite", "user_id", "zero", ""},
	{"galgame_activity", "user_id", "zero", ""},
	{"galgame_contributor", "user_id", "zero", ""},
	{"feed_activity", "user_id", "zero", ""},
	{"user_permission_override", "user_id", "zero", ""},
	{"chat_message", "sender_id", "zero", ""},
	{"chat_message", "receiver_id", "zero", ""},
	{"chat_message_reaction", "user_id", "zero", ""},
	{"chat_message_read_by", "user_id", "zero", ""},
	{"chat_room_admin", "user_id", "zero", ""},
	{"chat_room_participant", "user_id", "zero", ""},
	{"message", "sender_id", "zero", ""},
	{"message", "receiver_id", "zero", ""},
	{"system_message", "user_id", "zero", ""},
	{"system_message_read_state", "user_id", "zero", ""},
	{"kungal_user_state", "user_id", "zero", ""},
	{"user_follow", "follower_id", "zero", ""},
	{"user_follow", "followed_id", "zero", ""},
	{"user_friend", "user_id", "zero", ""},
	{"user_friend", "friend_id", "zero", ""},
	{"doc_article", "author_id", "keep", "admin content; role>1 not purgeable"},
	{"todo", "user_id", "zero", ""},
	{"todo", "claimed_user_id", "keep", "closed board entries keep who worked them"},
	{"permission_audit_log", "operator_id", "keep", "audit trail"},
	{"update_log", "user_id", "keep", "admin content"},
	{"unmoe", "user_id", "keep", "read-only logs; not a POST surface"},
}

type counterCheck struct {
	parent, countCol, child, fk string
	agg                         string
}

var counterChecks = []counterCheck{
	{"topic", "reply_count", "topic_reply", "topic_id", ""},
	{"topic", "comment_count", "topic_comment", "topic_id", ""},
	{"topic", "like_count", "topic_like", "topic_id", ""},
	{"topic", "dislike_count", "topic_dislike", "topic_id", ""},
	{"topic", "favorite_count", "topic_favorite", "topic_id", ""},
	{"topic", "upvote_count", "topic_upvote", "topic_id", ""},
	{"topic_reply", "comment_count", "topic_comment", "topic_reply_id", ""},
	{"topic_reply", "like_count", "topic_reply_like", "topic_reply_id", ""},
	{"topic_reply", "dislike_count", "topic_reply_dislike", "topic_reply_id", ""},
	{"topic_poll_option", "vote_count", "topic_poll_vote", "option_id", ""},
	{"galgame", "rating_count", "galgame_rating", "work_id", ""},
	{"galgame", "resource_count", "galgame_resource", "work_id", ""},
	{"galgame", "like_count", "galgame_like", "work_id", ""},
	{"galgame", "favorite_count", "galgame_collection_item", "work_id", "COUNT(DISTINCT user_id)"},
	{"topic_lottery", "entry_count", "topic_lottery_entry", "lottery_id", ""},
	{"galgame_collection", "item_count", "galgame_collection_item", "collection_id", ""},
	{"galgame_quiz", "favorite_count", "galgame_quiz_favorite", "quiz_id", ""},
	{"galgame_quiz", "answer_count", "galgame_quiz_answer", "quiz_id", "COUNT(*) FILTER (WHERE role = 'answerer')"},
	{"galgame_quiz", "correct_count", "galgame_quiz_answer", "quiz_id", "COUNT(*) FILTER (WHERE is_correct)"},
	{"galgame_quiz", "quality_sum", "galgame_quiz_answer", "quiz_id", "COALESCE(SUM(quality_rating), 0)"},
	{"galgame_quiz", "quality_count", "galgame_quiz_answer", "quiz_id", "COUNT(*) FILTER (WHERE quality_rating > 0)"},
	{"galgame_rating", "like_count", "galgame_rating_like", "galgame_rating_id", ""},
	{"galgame_resource", "like_count", "galgame_resource_like", "galgame_resource_id", ""},
	{"galgame_website", "like_count", "galgame_website_like", "website_id", ""},
	{"galgame_website", "favorite_count", "galgame_website_favorite", "website_id", ""},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: purge-staging-verify <userID>")
		os.Exit(2)
	}
	userID, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("invalid userID:", err)
		os.Exit(2)
	}
	dsn := os.Getenv("KUN_STAGING_URL")
	if dsn == "" {
		fmt.Println("KUN_STAGING_URL not set")
		os.Exit(2)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}

	var dbName string
	db.Raw("SELECT current_database()").Scan(&dbName)
	if dbName != stagingDBName {
		fmt.Printf("REFUSING: connected to %q, not %q\n", dbName, stagingDBName)
		os.Exit(1)
	}
	fmt.Printf("== purge-staging-verify on %q, user %d ==\n\n", dbName, userID)

	count := func(table, col string) int64 {
		var n int64
		db.Table(table).Where(col+" = ?", userID).Count(&n)
		return n
	}
	counterDrift := func(c counterCheck) int64 {
		agg := c.agg
		if agg == "" {
			agg = "COUNT(*)"
		}
		var n int64
		db.Raw(fmt.Sprintf(
			"SELECT COUNT(*) FROM %s p WHERE p.%s <> (SELECT %s FROM %s c WHERE c.%s = p.id)",
			c.parent, c.countCol, agg, c.child, c.fk)).Scan(&n)
		return n
	}

	pass := true
	fail := func(format string, a ...any) {
		pass = false
		fmt.Printf("  FAIL: "+format+"\n", a...)
	}

	fmt.Println("[1] pre-purge footprint (non-zero columns):")
	var preTotal int64
	for _, c := range colChecks {
		if n := count(c.table, c.col); n > 0 {
			preTotal += n
			fmt.Printf("    %-28s %-15s %d\n", c.table, c.col, n)
		}
	}
	preDrift := map[string]int64{}
	for _, c := range counterChecks {
		preDrift[c.parent+"."+c.countCol] = counterDrift(c)
	}
	staleLastMsg := func() int64 {
		var n int64
		db.Raw("SELECT COUNT(*) FROM chat_room cr WHERE cr.last_message_sender_id IS NOT NULL " +
			"AND NOT EXISTS (SELECT 1 FROM chat_message m WHERE m.chat_room_id=cr.id AND m.sender_id=cr.last_message_sender_id)").Scan(&n)
		return n
	}
	preStale := staleLastMsg()

	fmt.Println("\n[2] running PurgeRepository.PurgeUserContent ...")
	repo := repository.NewPurgeRepository(db)
	stats, perr := repo.PurgeUserContent(userID)
	if perr != nil {
		fail("purge returned error: %v", perr)
		fmt.Println("\nRESULT: FAIL")
		os.Exit(1)
	}
	fmt.Printf("    reported preview Total=%d (topics=%d replies=%d topicComments=%d ratings=%d resources=%d websites=%d toolsets=%d polls=%d lotteries=%d drafts=%d quizzes=%d collections=%d todos=%d chat=%d msgs=%d interactions=%d)\n",
		stats.Total, stats.Topics, stats.Replies, stats.TopicComments,
		stats.Ratings, stats.Resources, stats.Websites, stats.Toolsets,
		stats.Polls, stats.Lotteries, stats.Drafts, stats.Quizzes, stats.Collections, stats.Todos,
		stats.ChatMessages, stats.Messages, stats.Interactions)

	fmt.Println("\n[3] completeness — every traced column must be 0 (except by-design keeps):")
	for _, c := range colChecks {
		n := count(c.table, c.col)
		switch c.mode {
		case "zero":
			if n != 0 {
				fail("%s.%s still has %d rows for user %d", c.table, c.col, n, userID)
			}
		case "keep":
			if n > 0 {
				fmt.Printf("    keep  %-26s %-15s %d  (%s)\n", c.table, c.col, n, c.why)
			}
		}
	}
	if pass {
		fmt.Println("    OK — all must-zero columns are 0")
	}

	fmt.Println("\n[4] counter integrity — drift must not increase vs pre-purge:")
	for _, c := range counterChecks {
		key := c.parent + "." + c.countCol
		post := counterDrift(c)
		if post > preDrift[key] {
			fail("%s drift increased %d -> %d", key, preDrift[key], post)
		} else if post != preDrift[key] {
			fmt.Printf("    %-32s drift %d -> %d (improved)\n", key, preDrift[key], post)
		}
	}
	if pass {
		fmt.Println("    OK — no counter left worse than before")
	}

	fmt.Println("\n[5] orphan checks:")
	orphan := func(label, sql string) {
		var n int64
		db.Raw(sql).Scan(&n)
		if n != 0 {
			fail("%s: %d orphan rows", label, n)
		}
	}
	orphan("topic_reply without topic", "SELECT COUNT(*) FROM topic_reply r WHERE NOT EXISTS (SELECT 1 FROM topic t WHERE t.id=r.topic_id)")
	orphan("topic_comment without reply", "SELECT COUNT(*) FROM topic_comment c WHERE NOT EXISTS (SELECT 1 FROM topic_reply r WHERE r.id=c.topic_reply_id)")
	orphan("chat_message in deleted room", "SELECT COUNT(*) FROM chat_message m WHERE NOT EXISTS (SELECT 1 FROM chat_room cr WHERE cr.id=m.chat_room_id)")
	orphan("chat_room_participant in deleted room", "SELECT COUNT(*) FROM chat_room_participant p WHERE NOT EXISTS (SELECT 1 FROM chat_room cr WHERE cr.id=p.chat_room_id)")
	orphan("topic_reaction without topic", "SELECT COUNT(*) FROM topic_reaction r WHERE NOT EXISTS (SELECT 1 FROM topic t WHERE t.id=r.topic_id)")
	orphan("topic_reply_reaction without reply", "SELECT COUNT(*) FROM topic_reply_reaction r WHERE NOT EXISTS (SELECT 1 FROM topic_reply x WHERE x.id=r.topic_reply_id)")
	orphan("topic_lottery without topic", "SELECT COUNT(*) FROM topic_lottery l WHERE NOT EXISTS (SELECT 1 FROM topic t WHERE t.id=l.topic_id)")
	orphan("private chat_room missing a side", "SELECT COUNT(*) FROM chat_room cr WHERE cr.type='private' AND (SELECT COUNT(*) FROM chat_room_participant p WHERE p.chat_room_id=cr.id) < 2")
	if pass {
		fmt.Println("    OK — no orphans")
	}
	if postStale := staleLastMsg(); postStale > preStale {
		fail("chat_room stale last_message_sender increased %d -> %d", preStale, postStale)
	} else {
		fmt.Printf("    last_message_sender staleness %d -> %d (pre-existing drift, not increased)\n", preStale, postStale)
	}

	fmt.Println("\n[6] idempotency — re-run purge:")
	stats2, perr2 := repo.PurgeUserContent(userID)
	if perr2 != nil {
		fail("second purge errored: %v", perr2)
	} else if stats2.Total != 0 {
		fail("second purge still found Total=%d (expected 0)", stats2.Total)
	} else {
		fmt.Println("    OK — second run found nothing and errored not")
	}

	fmt.Println("\n[7] community purge — author has 0 visible posts:")
	verifyCommunityPurge(fail, userID)

	fmt.Printf("\nRESULT: %s\n", map[bool]string{true: "PASS", false: "FAIL"}[pass])
	if !pass {
		os.Exit(1)
	}
}

func verifyCommunityPurge(fail func(string, ...any), userID int) {
	base := os.Getenv("KUN_COMMUNITY_API_BASE")
	clientID := firstNonEmpty(os.Getenv("KUN_COMMUNITY_CLIENT_ID"), os.Getenv("OAUTH_CLIENT_ID"))
	clientSecret := firstNonEmpty(os.Getenv("KUN_COMMUNITY_CLIENT_SECRET"), os.Getenv("OAUTH_CLIENT_SECRET"))
	if base == "" || clientID == "" || clientSecret == "" {
		fmt.Println("    SKIP — community S2S not configured (set KUN_COMMUNITY_API_BASE + client creds)")
		return
	}

	cli := communityclient.New(communityclient.Config{BaseURL: base, ClientID: clientID, ClientSecret: clientSecret})
	ctx := context.Background()
	if _, err := cli.AuthorPurge(ctx, int64(userID)); err != nil {
		fail("community AuthorPurge errored: %v", err)
		return
	}
	stats, err := cli.AuthorStats(ctx, []int64{int64(userID)})
	if err != nil {
		fail("community AuthorStats errored: %v", err)
		return
	}
	var visible int64
	for _, st := range stats.Stats {
		if st.AuthorID == int64(userID) {
			visible = st.VisiblePosts
		}
	}
	if visible != 0 {
		fail("community author %d still has %d visible posts after purge", userID, visible)
		return
	}
	fmt.Println("    OK — author has 0 visible community posts")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
