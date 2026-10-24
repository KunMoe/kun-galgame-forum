package apiv1

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"kun-galgame-api/internal/setting"
	"kun-galgame-api/internal/topic/ai"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/aiclient"
)

const (
	classificationTTL = 30 * time.Minute
	summaryTimeout    = 2 * time.Minute
)

// TopicAI is nil-safe: every method on a nil receiver reports the model off.
type TopicAI struct {
	client   *aiclient.Client
	settings *setting.Store
	repo     *repository.AIRepository
	// Tokens only live for one publish, so a key per process is enough: after
	// a restart an old token fails to verify and the topic is classified again.
	key []byte
}

func NewTopicAI(client *aiclient.Client, settings *setting.Store, repo *repository.AIRepository) *TopicAI {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return &TopicAI{client: client, settings: settings, repo: repo, key: key}
}

func (a *TopicAI) ClassifyEnabled() bool {
	return a != nil && a.client != nil && a.settings.Bool(setting.TopicAIClassifyEnabled)
}

func (a *TopicAI) SummaryEnabled() bool {
	return a != nil && a.client != nil && a.settings.Bool(setting.TopicAISummaryEnabled)
}

func (a *TopicAI) model() string { return a.settings.String(setting.TopicAIModel) }

func (a *TopicAI) choices() ([]ai.SectionChoice, error) {
	rows, err := a.repo.SectionChoices()
	if err != nil {
		return nil, err
	}
	out := make([]ai.SectionChoice, 0, len(rows))
	for _, r := range rows {
		if !r.IsArchived {
			out = append(out, ai.SectionChoice{Key: r.Name, Label: r.Label, Description: r.Description})
		}
	}
	return out, nil
}

func (a *TopicAI) Classify(ctx context.Context, title, body string) (ai.Verdict, error) {
	choices, err := a.choices()
	if err != nil {
		return ai.Verdict{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(a.settings.Int(setting.TopicAITimeoutMs))*time.Millisecond)
	defer cancel()
	return ai.Classify(ctx, a.client, a.model(), choices, title, body)
}

func textDigest(title, body string) []byte {
	sum := sha256.Sum256([]byte(title + "\x00" + body))
	return sum[:]
}

func (a *TopicAI) sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, a.key)
	mac.Write(payload)
	return mac.Sum(nil)[:16]
}

// Token binds a verdict to the caller and to the exact text it was given.
func (a *TopicAI) Token(userID int, title, body, section string, expires time.Time) string {
	payload := make([]byte, 0, 16+32+len(section))
	payload = binary.BigEndian.AppendUint64(payload, uint64(userID))
	payload = binary.BigEndian.AppendUint64(payload, uint64(expires.Unix()))
	payload = append(payload, textDigest(title, body)...)
	payload = append(payload, section...)
	return base64.RawURLEncoding.EncodeToString(append(payload, a.sign(payload)...))
}

var errBadToken = errors.New("classification token does not verify")

func (a *TopicAI) Verify(token string, userID int, title, body string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) < 16+32+16 {
		return "", errBadToken
	}
	payload, mac := raw[:len(raw)-16], raw[len(raw)-16:]
	if !hmac.Equal(mac, a.sign(payload)) {
		return "", errBadToken
	}
	if int(binary.BigEndian.Uint64(payload[0:8])) != userID {
		return "", errBadToken
	}
	if time.Now().Unix() > int64(binary.BigEndian.Uint64(payload[8:16])) {
		return "", errBadToken
	}
	if !hmac.Equal(payload[16:48], textDigest(title, body)) {
		return "", errBadToken
	}
	return string(payload[48:]), nil
}

func (a *TopicAI) wantsSummary(body string) bool {
	runes, _ := ai.Signals(body)
	return runes >= a.settings.Int(setting.TopicSummaryMinLength)
}

// Summarize runs outside the request: publishing never waits for a summary.
func (a *TopicAI) Summarize(topicID int, title, body string) {
	if !a.SummaryEnabled() || !a.wantsSummary(body) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), summaryTimeout)
		defer cancel()
		if err := a.summarizeNow(ctx, topicID, title, body); err != nil {
			slog.Warn("topic summary: model failed", "topic_id", topicID, "error", err)
		}
	}()
}

// AfterEdit writes a summary for a body that now qualifies but has none, or
// rewrites one once the edits since it add up to the threshold.
func (a *TopicAI) AfterEdit(topic *model.Topic, drift int, title, body string) {
	if !a.SummaryEnabled() {
		return
	}
	if topic.AISummary == nil || topic.AISummaryDrift+drift >= a.settings.Int(setting.TopicSummaryRegenerateThreshold) {
		a.Summarize(topic.ID, title, body)
	}
}

func (a *TopicAI) summarizeNow(ctx context.Context, topicID int, title, body string) error {
	summary, err := ai.Summarize(ctx, a.client, a.model(), title, body)
	if err != nil {
		return err
	}
	return a.repo.SetSummary(topicID, &summary, a.model())
}

// editDrift counts the runes an edit inserted plus the runes it deleted,
// line by line, so a rewritten paragraph counts as both and an untouched one
// as neither.
func editDrift(before, after string) int {
	a := strings.Split(before, "\n")
	b := strings.Split(after, "\n")
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		a, b = a[1:], b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	runes := func(lines []string) int {
		n := 0
		for _, l := range lines {
			n += utf8.RuneCountInString(l)
		}
		return n
	}
	if len(a)*len(b) > 4_000_000 {
		return runes(a) + runes(b)
	}
	// Runes of the longest common subsequence of lines.
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			switch {
			case a[i-1] == b[j-1]:
				cur[j] = prev[j-1] + utf8.RuneCountInString(a[i-1])
			case prev[j] >= cur[j-1]:
				cur[j] = prev[j]
			default:
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
	}
	common := prev[len(b)]
	return runes(a) + runes(b) - 2*common
}
