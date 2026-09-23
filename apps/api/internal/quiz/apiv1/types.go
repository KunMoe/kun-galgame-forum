package apiv1

import (
	"github.com/danielgtaylor/huma/v2"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
)

const defaultLimit = 50

type QuizChoice string

func (QuizChoice) Schema(huma.Registry) *huma.Schema {
	n := 200
	return &huma.Schema{Type: huma.TypeString, MaxLength: &n, Description: "One option of a single or multiple choice quiz. Free text; never use it as a decision input."}
}

type QuizSummary struct {
	Object         string                  `json:"object" enum:"quiz" maxLength:"4" doc:"Type discriminant. Always quiz."`
	ID             repr.DecimalID          `json:"id" doc:"Quiz id."`
	Prompt         content.ContentDocument `json:"prompt" doc:"The question as a restricted document of paragraph, text, break and inline_spoiler nodes."`
	QuizType       string                  `json:"quiz_type" enum:"single,multiple,judge" maxLength:"8" doc:"Kind of quiz."`
	QuizCategory   string                  `json:"quiz_category" enum:"plot,character,system,music,voice,company,trivia,other" maxLength:"9" doc:"Subject of the quiz."`
	Difficulty     int                     `json:"difficulty" minimum:"1" maximum:"10" doc:"Difficulty, 1–10."`
	SpoilerLevel   string                  `json:"spoiler_level" enum:"none,portion,serious" maxLength:"7" doc:"How much of a work the prompt spoils."`
	Author         repr.UserRef            `json:"author" doc:"Author of the quiz."`
	ViewCount      int                     `json:"view_count" minimum:"0" doc:"Lifetime view count."`
	AnswerCount    int                     `json:"answer_count" minimum:"0" doc:"Number of answerers, excluding the author row."`
	CorrectCount   int                     `json:"correct_count" minimum:"0" doc:"Number of correct answerers."`
	FavoriteCount  int                     `json:"favorite_count" minimum:"0" doc:"Number of favorites."`
	QualityAverage *float64                `json:"quality_average" minimum:"1" maximum:"10" doc:"Mean rating, one decimal place. null when nobody has rated."`
	QualityCount   int                     `json:"quality_count" minimum:"0" doc:"Number of ratings."`
	CommentCount   int                     `json:"comment_count" minimum:"0" doc:"Comments on the quiz's wall."`
	CreatedAt      repr.DateTime           `json:"created_at" doc:"Creation time."`
	UpdatedAt      repr.DateTime           `json:"updated_at" doc:"Time of the latest write to the row."`
	BumpedAt       repr.DateTime           `json:"bumped_at" doc:"Time of the latest bump: an answer inside the three-day window, or an edit."`
	Viewer         *QuizSummaryViewer      `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type Quiz struct {
	Object         string                  `json:"object" enum:"quiz" maxLength:"4" doc:"Type discriminant. Always quiz."`
	ID             repr.DecimalID          `json:"id" doc:"Quiz id."`
	Prompt         content.ContentDocument `json:"prompt" doc:"The question as a restricted document of paragraph, text, break and inline_spoiler nodes."`
	QuizType       string                  `json:"quiz_type" enum:"single,multiple,judge" maxLength:"8" doc:"Kind of quiz."`
	QuizCategory   string                  `json:"quiz_category" enum:"plot,character,system,music,voice,company,trivia,other" maxLength:"9" doc:"Subject of the quiz."`
	Difficulty     int                     `json:"difficulty" minimum:"1" maximum:"10" doc:"Difficulty, 1–10."`
	SpoilerLevel   string                  `json:"spoiler_level" enum:"none,portion,serious" maxLength:"7" doc:"How much of a work the prompt spoils."`
	Author         repr.UserRef            `json:"author" doc:"Author of the quiz."`
	ViewCount      int                     `json:"view_count" minimum:"0" doc:"Lifetime view count. Each read of this operation adds one."`
	AnswerCount    int                     `json:"answer_count" minimum:"0" doc:"Number of answerers, excluding the author row."`
	CorrectCount   int                     `json:"correct_count" minimum:"0" doc:"Number of correct answerers."`
	FavoriteCount  int                     `json:"favorite_count" minimum:"0" doc:"Number of favorites."`
	QualityAverage *float64                `json:"quality_average" minimum:"1" maximum:"10" doc:"Mean rating, one decimal place. null when nobody has rated."`
	QualityCount   int                     `json:"quality_count" minimum:"0" doc:"Number of ratings."`
	CommentCount   int                     `json:"comment_count" minimum:"0" doc:"Comments on the quiz's wall."`
	CreatedAt      repr.DateTime           `json:"created_at" doc:"Creation time."`
	UpdatedAt      repr.DateTime           `json:"updated_at" doc:"Time of the latest write to the row."`
	BumpedAt       repr.DateTime           `json:"bumped_at" doc:"Time of the latest bump: an answer inside the three-day window, or an edit."`
	Content        content.ContentDocument `json:"content" doc:"Description as a Markdown document. An empty document when there is no description."`
	Choices        []QuizChoice            `json:"choices" maxItems:"20" doc:"Option texts of a single or multiple choice quiz, without the answer key. Empty array for a judge quiz. Never null."`
	Solution       *QuizSolution           `json:"solution" doc:"The answer key and explanation. null when the caller may not see them."`
	IsWorkHidden   bool                    `json:"is_work_hidden" doc:"Whether linked works stay hidden until the caller may see the answer key."`
	Works          []repr.WorkRef          `json:"works" maxItems:"20" doc:"Linked works. Empty when is_work_hidden is true and the caller may not see the answer key. Never null."`
	Viewer         *QuizViewer             `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type QuizSolution struct {
	Object               string                  `json:"object" enum:"quiz_solution" maxLength:"13" doc:"Type discriminant. Always quiz_solution."`
	CorrectChoiceIndexes []int                   `json:"correct_choice_indexes" maxItems:"20" doc:"0-based indexes of the correct choices. Empty array for a judge quiz."`
	JudgeAnswer          *bool                   `json:"judge_answer" doc:"The correct true/false for a judge quiz. null otherwise."`
	Explanation          content.ContentDocument `json:"explanation" doc:"Explanation as a Markdown document. An empty document when there is none."`
}

type QuizSource struct {
	Object               string           `json:"object" enum:"quiz_source" maxLength:"11" doc:"Type discriminant. Always quiz_source."`
	QuizID               repr.DecimalID   `json:"quiz_id" doc:"Id of the quiz."`
	QuizType             string           `json:"quiz_type" enum:"single,multiple,judge" maxLength:"8" doc:"Kind of quiz."`
	QuizCategory         string           `json:"quiz_category" enum:"plot,character,system,music,voice,company,trivia,other" maxLength:"9" doc:"Subject of the quiz."`
	Difficulty           int              `json:"difficulty" minimum:"1" maximum:"10" doc:"Difficulty, 1–10."`
	SpoilerLevel         string           `json:"spoiler_level" enum:"none,portion,serious" maxLength:"7" doc:"How much of a work the prompt spoils."`
	PromptText           string           `json:"prompt_text" maxLength:"200" doc:"Stored prompt text. Free text; never use it as a decision input."`
	DescriptionMarkdown  string           `json:"description_markdown" maxLength:"20000" doc:"Stored Markdown of the description. Free text; never use it as a decision input."`
	ExplanationMarkdown  string           `json:"explanation_markdown" maxLength:"2000" doc:"Stored Markdown of the explanation. Free text; never use it as a decision input."`
	Choices              []QuizChoice     `json:"choices" maxItems:"20" doc:"Option texts. Empty array for a judge quiz. Never null."`
	CorrectChoiceIndexes []int            `json:"correct_choice_indexes" maxItems:"20" doc:"0-based indexes of the correct choices. Empty array for a judge quiz."`
	JudgeAnswer          *bool            `json:"judge_answer" doc:"The correct true/false for a judge quiz. null otherwise."`
	WorkIDs              []repr.DecimalID `json:"work_ids" maxItems:"20" doc:"Linked work ids. Empty array when there are none. Never null."`
	IsWorkHidden         bool             `json:"is_work_hidden" doc:"Whether linked works stay hidden until the caller may see the answer key."`
}

type QuizAnswer struct {
	Object     string          `json:"object" enum:"quiz_answer" maxLength:"11" doc:"Type discriminant. Always quiz_answer."`
	ID         repr.DecimalID  `json:"id" doc:"Answer row id."`
	QuizID     repr.DecimalID  `json:"quiz_id" doc:"Id of the quiz."`
	Answerer   repr.UserRef    `json:"answerer" doc:"User who answered."`
	Submission *QuizSubmission `json:"submission" doc:"What they submitted. null when the caller may not see the answer key."`
	IsCorrect  *bool           `json:"is_correct" doc:"Whether they were correct. null when the caller may not see the answer key."`
	AnsweredAt repr.DateTime   `json:"answered_at" doc:"Time they answered."`
}

type QuizSubmission struct {
	ChoiceIndexes []int `json:"choice_indexes" maxItems:"20" doc:"0-based indexes of the chosen options. Empty array for a judge quiz."`
	JudgeChoice   *bool `json:"judge_choice" doc:"The true/false chosen on a judge quiz. null otherwise."`
}

type QuizAnswerResult struct {
	Object   string       `json:"object" enum:"quiz_answer_result" maxLength:"18" doc:"Type discriminant. Always quiz_answer_result."`
	Answer   QuizAnswer   `json:"answer" doc:"The row just written."`
	Solution QuizSolution `json:"solution" doc:"The answer key and explanation."`
}

type QuizViewer struct {
	HasAnswered   bool              `json:"has_answered" doc:"Whether the caller has an answerer row. The author's author row does not count."`
	Answer        *QuizViewerAnswer `json:"answer" doc:"The caller's own answer. null when they have not answered."`
	CanEdit       bool              `json:"can_edit" doc:"Whether the caller may edit this quiz. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete     bool              `json:"can_delete" doc:"Whether the caller may delete this quiz. Requests authenticated with a Bearer token never carry staff powers."`
	HasFavorited  bool              `json:"has_favorited" doc:"Whether the caller favorited the quiz."`
	QualityRating *int              `json:"quality_rating" minimum:"1" maximum:"10" doc:"The caller's rating, 1–10. null when they have not rated."`
}

type QuizViewerAnswer struct {
	Submission QuizSubmission `json:"submission" doc:"What the caller submitted."`
	IsCorrect  bool           `json:"is_correct" doc:"Whether the caller was correct."`
	AnsweredAt repr.DateTime  `json:"answered_at" doc:"Time the caller answered."`
}

type QuizSummaryViewer struct {
	HasAnswered bool  `json:"has_answered" doc:"Whether the caller has an answerer row. The author's author row does not count."`
	IsCorrect   *bool `json:"is_correct" doc:"Whether the caller was correct. null when they have not answered."`
}

type QuizEngagement struct {
	Object        string            `json:"object" enum:"quiz_engagement" maxLength:"15" doc:"Type discriminant. Always quiz_engagement."`
	QuizID        repr.DecimalID    `json:"quiz_id" doc:"Id of the quiz."`
	FavoriteCount int               `json:"favorite_count" minimum:"0" doc:"Number of favorites after this request."`
	Viewer        *EngagementViewer `json:"viewer" doc:"The caller's favorite state after this request."`
}

type EngagementViewer struct {
	HasFavorited bool `json:"has_favorited" doc:"Whether the caller favorited the quiz."`
}

type QuizQuality struct {
	Object         string         `json:"object" enum:"quiz_quality" maxLength:"12" doc:"Type discriminant. Always quiz_quality."`
	QuizID         repr.DecimalID `json:"quiz_id" doc:"Id of the quiz."`
	QualityAverage *float64       `json:"quality_average" minimum:"1" maximum:"10" doc:"Mean rating after this request, one decimal place."`
	QualityCount   int            `json:"quality_count" minimum:"0" doc:"Number of ratings after this request."`
	Viewer         *QualityViewer `json:"viewer" doc:"The caller's rating after this request."`
}

type QualityViewer struct {
	QualityRating *int `json:"quality_rating" minimum:"1" maximum:"10" doc:"The rating just written. Never null in this response."`
}

type QuizState struct {
	Object       string         `json:"object" enum:"quiz_state" maxLength:"10" doc:"Type discriminant. Always quiz_state."`
	QuizID       repr.DecimalID `json:"quiz_id" doc:"Id of the quiz this state is about."`
	HasFavorited bool           `json:"has_favorited" doc:"Whether the caller favorited the quiz."`
}

type QuizCreate struct {
	PromptText           string           `json:"prompt_text" minLength:"1" maxLength:"200" doc:"The question. Length is checked on the raw value; only whitespace is TOO_SHORT. Free text; never use it as a decision input."`
	DescriptionMarkdown  string           `json:"description_markdown" required:"false" maxLength:"20000" doc:"Markdown description. May be empty. Free text; never use it as a decision input."`
	ExplanationMarkdown  string           `json:"explanation_markdown" required:"false" maxLength:"2000" doc:"Markdown explanation shown after answering. May be empty. Free text; never use it as a decision input."`
	QuizType             string           `json:"quiz_type" enum:"single,multiple,judge" maxLength:"8" doc:"Kind of quiz. Immutable after create."`
	QuizCategory         string           `json:"quiz_category" enum:"plot,character,system,music,voice,company,trivia,other" maxLength:"9" doc:"Subject of the quiz."`
	Difficulty           int              `json:"difficulty" minimum:"1" maximum:"10" doc:"Difficulty, 1–10."`
	SpoilerLevel         string           `json:"spoiler_level" enum:"none,portion,serious" required:"false" maxLength:"7" doc:"How much of a work the prompt spoils. Omitted is none."`
	Choices              []QuizChoice     `json:"choices" required:"false" maxItems:"20" doc:"Option texts. Required with 2–20 unique items for single and multiple; inconsistent on judge."`
	CorrectChoiceIndexes []int            `json:"correct_choice_indexes" required:"false" maxItems:"20" doc:"0-based indexes of the correct choices. Exactly one for single, at least one unique in range for multiple, empty for judge."`
	JudgeAnswer          *bool            `json:"judge_answer,omitempty" doc:"The correct true/false. Required for judge; must be null or absent otherwise."`
	WorkIDs              []repr.DecimalID `json:"work_ids" required:"false" maxItems:"20" doc:"Linked catalog work ids, unique, at most 20. Each must exist in catalog."`
	IsWorkHidden         bool             `json:"is_work_hidden" required:"false" doc:"Whether linked works stay hidden until the caller may see the answer key."`
}

type QuizPatch struct {
	PromptText           *string          `json:"prompt_text,omitempty" minLength:"1" maxLength:"200" doc:"New prompt. Free text; never use it as a decision input."`
	DescriptionMarkdown  *string          `json:"description_markdown,omitempty" maxLength:"20000" doc:"New Markdown description. Free text; never use it as a decision input."`
	ExplanationMarkdown  *string          `json:"explanation_markdown,omitempty" maxLength:"2000" doc:"New Markdown explanation. Free text; never use it as a decision input."`
	QuizType             *string          `json:"quiz_type,omitempty" enum:"single,multiple,judge" maxLength:"8" doc:"Kind of quiz. A value different from the stored type is IMMUTABLE."`
	QuizCategory         *string          `json:"quiz_category,omitempty" enum:"plot,character,system,music,voice,company,trivia,other" maxLength:"9" doc:"New subject."`
	Difficulty           *int             `json:"difficulty,omitempty" minimum:"1" maximum:"10" doc:"New difficulty, 1–10."`
	SpoilerLevel         *string          `json:"spoiler_level,omitempty" enum:"none,portion,serious" maxLength:"7" doc:"New spoiler level."`
	Choices              []QuizChoice     `json:"choices" required:"false" maxItems:"20" doc:"When present, replaces every option."`
	CorrectChoiceIndexes []int            `json:"correct_choice_indexes" required:"false" maxItems:"20" doc:"When present, replaces the answer key indexes."`
	JudgeAnswer          *bool            `json:"judge_answer,omitempty" doc:"When present, replaces the judge answer."`
	WorkIDs              []repr.DecimalID `json:"work_ids" required:"false" maxItems:"20" doc:"When present, replaces every linked work."`
	IsWorkHidden         *bool            `json:"is_work_hidden,omitempty" doc:"When present, replaces the hidden-works flag."`
}

type QualityPut struct {
	Rating int `json:"rating" minimum:"1" maximum:"10" doc:"Quality rating, 1–10."`
}

type listQuizzesInput struct {
	Page         int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit        int    `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`
	WorkID       string `query:"work_id" required:"false" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"When set, only quizzes linked to this work."`
	AuthorID     string `query:"author_id" required:"false" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"When set, only quizzes this user authored."`
	QuizType     string `query:"quiz_type" enum:"single,multiple,judge" required:"false" maxLength:"8" doc:"When set, only this type. Omitted means every type."`
	QuizCategory string `query:"quiz_category" enum:"plot,character,system,music,voice,company,trivia,other" required:"false" maxLength:"9" doc:"When set, only this category. Omitted means every category."`
	Difficulty   int    `query:"difficulty" minimum:"1" maximum:"10" doc:"When set, only this difficulty. Omitted means every difficulty; 0 is out of range."`
	SpoilerLevel string `query:"spoiler_level" enum:"none,portion,serious" required:"false" maxLength:"7" doc:"When set, only this spoiler level. Omitted means every level."`
	IncludeNSFW  bool   `query:"include_nsfw" default:"false" doc:"When true, quizzes linked to an NSFW work are included. Default false."`
	Sort         string `query:"sort" enum:"bumped_at_desc,bumped_at_asc,created_desc,created_asc,view_desc,view_asc,view_1d_desc,view_1d_asc,view_7d_desc,view_7d_asc,view_30d_desc,view_30d_asc,difficulty_desc,difficulty_asc,answer_count_desc,answer_count_asc" default:"bumped_at_desc" maxLength:"17" doc:"Sort token. Default bumped_at_desc."`
}

type listQuizzesOutput struct {
	Body repr.PageList[QuizSummary]
}

type quizIDInput struct {
	QuizID string `path:"quiz_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Quiz id."`
}

type quizOutput struct {
	Body Quiz
}

type sourceOutput struct {
	Body QuizSource
}

type listAnswersInput struct {
	QuizID string `path:"quiz_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Quiz id."`
	collect.Page
	collect.Total
}

type listAnswersOutput struct {
	Body repr.CountedList[QuizAnswer]
}

type listMyAnsweredInput struct {
	Page  int `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"50" doc:"Page size. 1–100, default 50. Values above 100 are rejected, not clamped."`
}

type listMyQuizStatesInput struct {
	QuizIDs []repr.DecimalID `query:"quiz_ids" required:"true" maxItems:"100" doc:"Quiz ids to answer for, comma-separated. 1 to 100 of them."`
}

type listMyQuizStatesOutput struct {
	Body repr.BatchList[QuizState]
}

type listWorkSuggestionsInput struct {
	Q           string `query:"q" minLength:"1" maxLength:"100" doc:"Search text. Length is checked on the raw value; only whitespace is TOO_SHORT. Free text; never use it as a decision input."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, NSFW works are included. Default false."`
}

type listWorkSuggestionsOutput struct {
	Body repr.List[repr.WorkRef]
}

type createQuizInput struct {
	Body QuizCreate
}

type createQuizOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new quiz, such as /api/v1/quizzes/12."`
	Body     Quiz
}

type patchQuizInput struct {
	QuizID string `path:"quiz_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Quiz id."`
	Body   QuizPatch
}

type noContentOutput struct{}

type createAnswerInput struct {
	QuizID string `path:"quiz_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Quiz id."`
	Body   QuizSubmission
}

type createAnswerOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the quiz, such as /api/v1/quizzes/12."`
	Body     QuizAnswerResult
}

type quizFavoriteInput struct {
	QuizID string `path:"quiz_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Quiz id."`
}

type engagementOutput struct {
	Body QuizEngagement
}

type putQualityInput struct {
	QuizID string `path:"quiz_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Quiz id."`
	Body   QualityPut
}

type qualityOutput struct {
	Body QuizQuality
}
