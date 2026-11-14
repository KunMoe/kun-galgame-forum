package repository

import "testing"

func TestInteractionDeletesRecountFromReactions(t *testing.T) {
	byTable := map[string]interactionDelete{}
	for _, it := range interactionDeletes {
		byTable[it.table] = it
	}
	topic := byTable["topic_reaction"]
	if len(topic.recounts) != 2 {
		t.Fatalf("topic_reaction recounts %d, want 2", len(topic.recounts))
	}
	if topic.recounts[0].countCol != "like_count" || topic.recounts[0].agg != "COUNT(*) FILTER (WHERE reaction = 'like')" {
		t.Errorf("topic like recount %+v", topic.recounts[0])
	}
	if topic.recounts[1].countCol != "dislike_count" || topic.recounts[1].agg != "COUNT(*) FILTER (WHERE reaction = 'dislike')" {
		t.Errorf("topic dislike recount %+v", topic.recounts[1])
	}
	reply := byTable["topic_reply_reaction"]
	if len(reply.recounts) != 2 {
		t.Fatalf("topic_reply_reaction recounts %d, want 2", len(reply.recounts))
	}
	if reply.recounts[0].parentCol != "topic_reply_id" || reply.recounts[0].agg != "COUNT(*) FILTER (WHERE reaction = 'like')" {
		t.Errorf("reply like recount %+v", reply.recounts[0])
	}
	if reply.recounts[1].countCol != "dislike_count" || reply.recounts[1].agg != "COUNT(*) FILTER (WHERE reaction = 'dislike')" {
		t.Errorf("reply dislike recount %+v", reply.recounts[1])
	}
}
