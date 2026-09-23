package apiv1

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/quiz/repository"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

const (
	maxPrompt      = 200
	maxDescription = 20000
	maxExplanation = 2000
	maxChoices     = 20
	minChoices     = 2
	maxChoiceLen   = 200
	maxWorks       = 20
)

const (
	quizTypeSingle   = "single"
	quizTypeMultiple = "multiple"
	quizTypeJudge    = "judge"
)

var sortSpecs = map[string]repository.SortSpec{
	"bumped_at_desc":    {Expr: "q.status_update_time DESC, q.id DESC"},
	"bumped_at_asc":     {Expr: "q.status_update_time ASC, q.id ASC"},
	"created_desc":      {Expr: "q.created DESC, q.id DESC"},
	"created_asc":       {Expr: "q.created ASC, q.id ASC"},
	"view_desc":         {Expr: "q.view DESC, q.id DESC"},
	"view_asc":          {Expr: "q.view ASC, q.id ASC"},
	"view_1d_desc":      {Expr: "COALESCE((SELECT SUM(d.count) FROM galgame_quiz_view_daily d WHERE d.entity_id = q.id AND d.day = CURRENT_DATE), 0) DESC, q.id DESC"},
	"view_1d_asc":       {Expr: "COALESCE((SELECT SUM(d.count) FROM galgame_quiz_view_daily d WHERE d.entity_id = q.id AND d.day = CURRENT_DATE), 0) ASC, q.id ASC"},
	"view_7d_desc":      {Expr: "q.view_7d DESC, q.id DESC"},
	"view_7d_asc":       {Expr: "q.view_7d ASC, q.id ASC"},
	"view_30d_desc":     {Expr: "q.view_30d DESC, q.id DESC"},
	"view_30d_asc":      {Expr: "q.view_30d ASC, q.id ASC"},
	"difficulty_desc":   {Expr: "q.difficulty DESC, q.id DESC"},
	"difficulty_asc":    {Expr: "q.difficulty ASC, q.id ASC"},
	"answer_count_desc": {Expr: "q.answer_count DESC, q.id DESC"},
	"answer_count_asc":  {Expr: "q.answer_count ASC, q.id ASC"},
}

var promptSpoilerRe = regexp.MustCompile(`(?s)\|\|(.*?)\|\|`)

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The token lacks the permission this decision needs.")
}

func contentRejected() *problem.Problem {
	return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
}

func selfAnswerForbidden() *problem.Problem {
	return problem.New(problem.CodeSelfAnswerForbidden, "Users cannot answer a quiz they authored.")
}

func alreadyExists() *problem.Problem {
	return problem.New(problem.CodeAlreadyExists, "The same subject already has a live record for this target.")
}

func quizAnswerRequired() *problem.Problem {
	return problem.New(problem.CodeQuizAnswerRequired, "The caller must already have answered this quiz; the author cannot answer their own quiz.")
}

func tooShort(pointer string, min int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooShort, "too short once surrounding whitespace is removed", &problem.FieldParams{MinLength: &min})
}

func tooLong(pointer string, max int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooLong, "longer than the field allows", &problem.FieldParams{MaxLength: &max})
}

func tooMany(pointer string, max int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooManyItems, "more items than the field allows", &problem.FieldParams{MaxItems: &max})
}

func tooFew(pointer string, min int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooFewItems, "fewer items than the field allows", &problem.FieldParams{MinItems: &min})
}

func duplicateItem(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonDuplicateItem, "duplicate of an earlier item", nil)
}

func immutable(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonImmutable, "this field cannot be changed", nil)
}

func inconsistent(pointer, other string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonInconsistentWith, other, nil)
}

func unknownRef(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonUnknownReference, "the referenced work is not a visible catalog work", nil)
}

func outOfRange(pointer string, min, max float64) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonOutOfRange, "outside the allowed range", &problem.FieldParams{Minimum: &min, Maximum: &max})
}

func parseID(raw string) (int, bool) {
	return repr.ParseID(repr.DecimalID(raw))
}

func pageOf(page, limit int) collect.PageNumber {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	return collect.PageNumber{Page: page, Limit: limit}
}

func trimSpace(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

func canEditQuiz(authorID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == authorID || user.Can(perm.QuizEditAny)
}

func canDeleteQuiz(authorID int, user *middleware.UserInfo) bool {
	if user == nil {
		return false
	}
	return user.ID == authorID || user.Can(perm.QuizDeleteAny)
}

func qualityAverage(sum, count int) *float64 {
	if count <= 0 {
		return nil
	}
	v := math.Round(float64(sum)/float64(count)*10) / 10
	return &v
}

func promptDocument(src string) content.ContentDocument {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	var inlines content.Inlines
	pos := 0
	for _, span := range promptSpoilerRe.FindAllStringSubmatchIndex(src, -1) {
		inlines = append(inlines, promptText(src[pos:span[0]])...)
		inlines = append(inlines, content.NewInlineSpoiler(promptText(src[span[2]:span[3]])))
		pos = span[1]
	}
	inlines = append(inlines, promptText(src[pos:])...)
	if len(inlines) == 0 {
		return content.NewDocument(nil)
	}
	return content.NewDocument(content.Blocks{content.NewParagraph(inlines)})
}

func promptText(s string) content.Inlines {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	var out content.Inlines
	for i, p := range parts {
		if i > 0 {
			out = append(out, content.NewBreak())
		}
		if p != "" {
			out = append(out, content.NewText(p))
		}
	}
	return out
}

type storedSingle struct {
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
}

type storedMultiple struct {
	Options []string `json:"options"`
	Answers []int    `json:"answers"`
}

type storedJudge struct {
	Answer bool `json:"answer"`
}

type submitSingle struct {
	Value *int `json:"value"`
}

type submitMultiple struct {
	Values []int `json:"values"`
}

type submitJudge struct {
	Value *bool `json:"value"`
}

func parseChoices(qtype string, raw json.RawMessage) ([]string, error) {
	switch qtype {
	case quizTypeSingle:
		var c storedSingle
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		if c.Options == nil {
			c.Options = []string{}
		}
		return c.Options, nil
	case quizTypeMultiple:
		var c storedMultiple
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		if c.Options == nil {
			c.Options = []string{}
		}
		return c.Options, nil
	default:
		return []string{}, nil
	}
}

func parseKey(qtype string, raw json.RawMessage) (indexes []int, judge *bool, err error) {
	switch qtype {
	case quizTypeSingle:
		var c storedSingle
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, nil, err
		}
		return []int{c.Answer}, nil, nil
	case quizTypeMultiple:
		var c storedMultiple
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, nil, err
		}
		if c.Answers == nil {
			c.Answers = []int{}
		}
		return c.Answers, nil, nil
	case quizTypeJudge:
		var c storedJudge
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, nil, err
		}
		v := c.Answer
		return []int{}, &v, nil
	default:
		return []int{}, nil, nil
	}
}

func encodeContent(qtype string, choices []string, indexes []int, judge *bool) (json.RawMessage, error) {
	switch qtype {
	case quizTypeSingle:
		ans := 0
		if len(indexes) > 0 {
			ans = indexes[0]
		}
		return json.Marshal(storedSingle{Options: choices, Answer: ans})
	case quizTypeMultiple:
		if indexes == nil {
			indexes = []int{}
		}
		return json.Marshal(storedMultiple{Options: choices, Answers: indexes})
	case quizTypeJudge:
		v := false
		if judge != nil {
			v = *judge
		}
		return json.Marshal(storedJudge{Answer: v})
	default:
		return json.RawMessage("{}"), nil
	}
}

func encodeSubmission(qtype string, sub QuizSubmission) (json.RawMessage, error) {
	switch qtype {
	case quizTypeSingle:
		v := 0
		if len(sub.ChoiceIndexes) > 0 {
			v = int(sub.ChoiceIndexes[0])
		}
		return json.Marshal(submitSingle{Value: &v})
	case quizTypeMultiple:
		return json.Marshal(submitMultiple{Values: toInts(sub.ChoiceIndexes)})
	case quizTypeJudge:
		v := false
		if sub.JudgeChoice != nil {
			v = *sub.JudgeChoice
		}
		return json.Marshal(submitJudge{Value: &v})
	default:
		return json.RawMessage("{}"), nil
	}
}

func decodeSubmission(qtype string, raw json.RawMessage) (QuizSubmission, error) {
	out := QuizSubmission{ChoiceIndexes: []ChoiceIndex{}}
	if len(raw) == 0 {
		return out, nil
	}
	switch qtype {
	case quizTypeSingle:
		var s submitSingle
		if err := json.Unmarshal(raw, &s); err != nil {
			return out, err
		}
		if s.Value != nil {
			out.ChoiceIndexes = []ChoiceIndex{ChoiceIndex(*s.Value)}
		}
	case quizTypeMultiple:
		var s submitMultiple
		if err := json.Unmarshal(raw, &s); err != nil {
			return out, err
		}
		out.ChoiceIndexes = toIndexes(s.Values)
	case quizTypeJudge:
		var s submitJudge
		if err := json.Unmarshal(raw, &s); err != nil {
			return out, err
		}
		out.JudgeChoice = s.Value
	}
	return out, nil
}

func grade(qtype string, content, submitted json.RawMessage) (bool, error) {
	switch qtype {
	case quizTypeSingle:
		var c storedSingle
		var s submitSingle
		if err := json.Unmarshal(content, &c); err != nil {
			return false, err
		}
		if err := json.Unmarshal(submitted, &s); err != nil {
			return false, err
		}
		return s.Value != nil && *s.Value == c.Answer, nil
	case quizTypeMultiple:
		var c storedMultiple
		var s submitMultiple
		if err := json.Unmarshal(content, &c); err != nil {
			return false, err
		}
		if err := json.Unmarshal(submitted, &s); err != nil {
			return false, err
		}
		return intSetEqual(c.Answers, s.Values), nil
	case quizTypeJudge:
		var c storedJudge
		var s submitJudge
		if err := json.Unmarshal(content, &c); err != nil {
			return false, err
		}
		if err := json.Unmarshal(submitted, &s); err != nil {
			return false, err
		}
		return s.Value != nil && *s.Value == c.Answer, nil
	default:
		return false, errors.New("unknown quiz type")
	}
}

func intSetEqual(a, b []int) bool {
	set := make(map[int]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	seen := make(map[int]struct{}, len(b))
	for _, v := range b {
		if _, ok := set[v]; !ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return len(seen) == len(set)
}

func typedChoices(in []string) []QuizChoice {
	if in == nil {
		in = []string{}
	}
	out := make([]QuizChoice, len(in))
	for i, v := range in {
		out[i] = QuizChoice(v)
	}
	return out
}

func workIDStrings(ids []int) []repr.DecimalID {
	out := make([]repr.DecimalID, len(ids))
	for i, id := range ids {
		out[i] = repr.ID(id)
	}
	return out
}

func parseWorkIDs(raw []repr.DecimalID, pointer string) ([]int, []problem.FieldError) {
	if raw == nil {
		return nil, nil
	}
	if len(raw) > maxWorks {
		return nil, []problem.FieldError{tooMany(pointer, maxWorks)}
	}
	var errs []problem.FieldError
	out := make([]int, 0, len(raw))
	seen := map[int]int{}
	for i, part := range raw {
		ptr := pointer + "/" + strconv.Itoa(i)
		id, ok := repr.ParseID(repr.DecimalID(strings.TrimSpace(string(part))))
		if !ok {
			errs = append(errs, problem.AtPointer(ptr, problem.ReasonInvalidFormat, "must be a positive decimal integer", nil))
			continue
		}
		if prev, ok := seen[id]; ok {
			errs = append(errs, duplicateItem(ptr))
			_ = prev
			continue
		}
		seen[id] = i
		out = append(out, id)
	}
	return out, errs
}

func validateChoices(qtype string, raw []QuizChoice, required bool) ([]string, []problem.FieldError) {
	if raw == nil && !required {
		return nil, nil
	}
	if qtype == quizTypeJudge {
		if len(raw) > 0 {
			return nil, []problem.FieldError{inconsistent("/choices", "/quiz_type")}
		}
		return []string{}, nil
	}
	if len(raw) > maxChoices {
		return nil, []problem.FieldError{tooMany("/choices", maxChoices)}
	}
	if len(raw) < minChoices {
		return nil, []problem.FieldError{tooFew("/choices", minChoices)}
	}
	var errs []problem.FieldError
	out := make([]string, 0, len(raw))
	seen := map[string]int{}
	for i, c := range raw {
		ptr := "/choices/" + strconv.Itoa(i)
		if len(c) > maxChoiceLen {
			errs = append(errs, tooLong(ptr, maxChoiceLen))
			continue
		}
		name := trimSpace(string(c))
		if name == "" {
			errs = append(errs, tooShort(ptr, 1))
			continue
		}
		if prev, ok := seen[name]; ok {
			errs = append(errs, duplicateItem(ptr))
			_ = prev
			continue
		}
		seen[name] = i
		out = append(out, name)
	}
	return out, errs
}

func validateIndexes(qtype string, indexes []int, nChoices int, present bool) []problem.FieldError {
	if qtype == quizTypeJudge {
		if present && len(indexes) > 0 {
			return []problem.FieldError{inconsistent("/correct_choice_indexes", "/quiz_type")}
		}
		return nil
	}
	if !present {
		return []problem.FieldError{tooFew("/correct_choice_indexes", 1)}
	}
	if qtype == quizTypeSingle && len(indexes) != 1 {
		if len(indexes) == 0 {
			return []problem.FieldError{tooFew("/correct_choice_indexes", 1)}
		}
		return []problem.FieldError{tooMany("/correct_choice_indexes", 1)}
	}
	if qtype == quizTypeMultiple && len(indexes) < 1 {
		return []problem.FieldError{tooFew("/correct_choice_indexes", 1)}
	}
	var errs []problem.FieldError
	seen := map[int]int{}
	maxIdx := float64(nChoices - 1)
	for i, idx := range indexes {
		ptr := "/correct_choice_indexes/" + strconv.Itoa(i)
		if idx < 0 || idx >= nChoices {
			errs = append(errs, outOfRange(ptr, 0, max(0, maxIdx)))
			continue
		}
		if prev, ok := seen[idx]; ok {
			errs = append(errs, duplicateItem(ptr))
			_ = prev
			continue
		}
		seen[idx] = i
	}
	return errs
}

func validateJudge(qtype string, judge *bool, required bool) []problem.FieldError {
	if qtype == quizTypeJudge {
		if judge == nil && required {
			return []problem.FieldError{problem.AtPointer("/is_statement_true", problem.ReasonRequired, "required", nil)}
		}
		return nil
	}
	if judge != nil {
		return []problem.FieldError{inconsistent("/is_statement_true", "/quiz_type")}
	}
	return nil
}

func validateSubmission(qtype string, nChoices int, sub QuizSubmission) []problem.FieldError {
	var errs []problem.FieldError
	if qtype == quizTypeJudge {
		if len(sub.ChoiceIndexes) > 0 {
			errs = append(errs, inconsistent("/choice_indexes", "/is_statement_true"))
		}
		if sub.JudgeChoice == nil {
			errs = append(errs, problem.AtPointer("/is_statement_true", problem.ReasonRequired, "required", nil))
		}
		return errs
	}
	if sub.JudgeChoice != nil {
		errs = append(errs, inconsistent("/is_statement_true", "/choice_indexes"))
	}
	if qtype == quizTypeSingle && len(sub.ChoiceIndexes) != 1 {
		if len(sub.ChoiceIndexes) == 0 {
			errs = append(errs, tooFew("/choice_indexes", 1))
		} else {
			errs = append(errs, tooMany("/choice_indexes", 1))
		}
		return errs
	}
	if qtype == quizTypeMultiple && len(sub.ChoiceIndexes) < 1 {
		errs = append(errs, tooFew("/choice_indexes", 1))
		return errs
	}
	seen := map[int]int{}
	maxIdx := float64(nChoices - 1)
	for i, raw := range sub.ChoiceIndexes {
		idx := int(raw)
		ptr := "/choice_indexes/" + strconv.Itoa(i)
		if idx < 0 || idx >= nChoices {
			errs = append(errs, outOfRange(ptr, 0, max(0, maxIdx)))
			continue
		}
		if _, ok := seen[idx]; ok {
			errs = append(errs, duplicateItem(ptr))
			continue
		}
		seen[idx] = i
	}
	return errs
}

func fromRow(r model.GalgameQuiz) QuizSummary {
	return QuizSummary{
		Object: "quiz", ID: repr.ID(r.ID),
		Prompt: promptDocument(r.Question), QuizType: r.Type, QuizCategory: r.Category,
		Difficulty: r.Difficulty, SpoilerLevel: r.SpoilerLevel,
		ViewCount: r.View, AnswerCount: r.AnswerCount, CorrectCount: r.CorrectCount,
		FavoriteCount: r.FavoriteCount, QualityAverage: qualityAverage(r.QualitySum, r.QualityCount),
		QualityCount: r.QualityCount, CommentCount: r.CommentCount,
		CreatedAt: repr.Timestamp(r.CreatedAt), UpdatedAt: repr.Timestamp(r.UpdatedAt),
		BumpedAt: repr.Timestamp(r.StatusUpdateTime),
	}
}

func bytesEqual(a, b json.RawMessage) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	var xa, xb any
	if json.Unmarshal(a, &xa) != nil || json.Unmarshal(b, &xb) != nil {
		return string(a) == string(b)
	}
	ga, _ := json.Marshal(xa)
	gb, _ := json.Marshal(xb)
	return string(ga) == string(gb)
}

func intPtr(n int) *int { return &n }

func toInts(xs []ChoiceIndex) []int {
	out := make([]int, len(xs))
	for i, x := range xs {
		out[i] = int(x)
	}
	return out
}

func toIndexes(xs []int) []ChoiceIndex {
	out := make([]ChoiceIndex, len(xs))
	for i, x := range xs {
		out[i] = ChoiceIndex(x)
	}
	return out
}
