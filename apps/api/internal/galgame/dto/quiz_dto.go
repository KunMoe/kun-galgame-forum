package dto

import "encoding/json"

type QuizListRequest struct {
	Page       int    `query:"page" validate:"min=1"`
	Limit      int    `query:"limit" validate:"min=1,max=50"`
	SortField  string `query:"sort_field"`
	SortOrder  string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	Category   string `query:"category"`
	Type       string `query:"type"`
	Difficulty int    `query:"difficulty" validate:"omitempty,min=1,max=10"`
	WorkID     int    `query:"galgame_id"`
	UserID     int    `query:"user_id"`
}

type CreateQuizRequest struct {
	WorkIDs      []int           `json:"galgame_ids" validate:"omitempty,dive,min=1"`
	HideGalgame  bool            `json:"hide_galgame"`
	Category     string          `json:"category" validate:"required,oneof=plot character system music voice company trivia other"`
	Type         string          `json:"type" validate:"required,oneof=single multiple judge fill essay"`
	Difficulty   int             `json:"difficulty" validate:"required,min=1,max=10"`
	SpoilerLevel string          `json:"spoiler_level" validate:"omitempty,oneof=none portion serious"`
	Question     string          `json:"question" validate:"required,min=1,max=200"`
	Description  string          `json:"description" validate:"max=20000"`
	Content      json.RawMessage `json:"content" validate:"required"`
	Explanation  string          `json:"explanation" validate:"max=2000"`
}

type UpdateQuizRequest struct {
	QuizID       int             `json:"quiz_id" validate:"required,min=1"`
	WorkIDs      []int           `json:"galgame_ids" validate:"omitempty,dive,min=1"`
	HideGalgame  bool            `json:"hide_galgame"`
	Category     string          `json:"category" validate:"required,oneof=plot character system music voice company trivia other"`
	Type         string          `json:"type" validate:"required,oneof=single multiple judge fill essay"`
	Difficulty   int             `json:"difficulty" validate:"required,min=1,max=10"`
	SpoilerLevel string          `json:"spoiler_level" validate:"omitempty,oneof=none portion serious"`
	Question     string          `json:"question" validate:"required,min=1,max=200"`
	Description  string          `json:"description" validate:"max=20000"`
	Content      json.RawMessage `json:"content" validate:"required"`
	Explanation  string          `json:"explanation" validate:"max=2000"`
}

type QuizEditData struct {
	ID           int                `json:"id"`
	WorkIDs      []int              `json:"galgame_ids"`
	HideGalgame  bool               `json:"hide_galgame"`
	Category     string             `json:"category"`
	Type         string             `json:"type"`
	Difficulty   int                `json:"difficulty"`
	SpoilerLevel string             `json:"spoiler_level"`
	Question     string             `json:"question"`
	Description  string             `json:"description"`
	Content      json.RawMessage    `json:"content"`
	Explanation  string             `json:"explanation"`
	Galgames     []QuizGalgameBrief `json:"galgames"`
}

type QuizAnswererRecord struct {
	User      UserBrief       `json:"user"`
	Submitted json.RawMessage `json:"submitted,omitempty"`
	IsCorrect *bool           `json:"is_correct"`
	Created   string          `json:"created"`
}

type AnswerQuizRequest struct {
	QuizID    int             `json:"quiz_id" validate:"required,min=1"`
	Submitted json.RawMessage `json:"submitted" validate:"required"`
}

type RateQuizQualityRequest struct {
	QuizID        int `json:"quiz_id" validate:"required,min=1"`
	QualityRating int `json:"quality_rating" validate:"required,min=1,max=10"`
}

type DeleteQuizRequest struct {
	QuizID int `query:"quiz_id" validate:"required,min=1"`
}

type QuizGalgameBrief struct {
	ID           int    `json:"id"`
	ContentLimit string `json:"content_limit"`
	Name         string `json:"name"`
}

type QuizGalgameDetail struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	ContentLimit     string   `json:"content_limit"`
	AgeLimit         string   `json:"age_limit"`
	OriginalLanguage string   `json:"original_language"`
	Banner           string   `json:"banner"`
	BannerThumbhash  string   `json:"banner_thumbhash,omitempty"`
	Officials        []string `json:"officials"`
}

type QuizStats struct {
	View           int     `json:"view"`
	AnswerCount    int     `json:"answer_count"`
	CorrectCount   int     `json:"correct_count"`
	FavoriteCount  int     `json:"favorite_count"`
	QualityAverage float64 `json:"quality_average"`
	QualityCount   int     `json:"quality_count"`
	CommentCount   int     `json:"comment_count"`
}

type QuizCard struct {
	ID           int       `json:"id"`
	User         UserBrief `json:"user"`
	Category     string    `json:"category"`
	Type         string    `json:"type"`
	Difficulty   int       `json:"difficulty"`
	SpoilerLevel string    `json:"spoiler_level"`
	QuestionHtml string    `json:"question_html"`
	QuizStats
	Created          string `json:"created"`
	Updated          string `json:"updated"`
	StatusUpdateTime string `json:"status_update_time"`
	MyStatus         string `json:"my_status"`
}

type QuizListPage struct {
	QuizData []QuizCard `json:"quiz_data"`
	Total    int64      `json:"total"`
}

type QuizPlay struct {
	ID              int             `json:"id"`
	User            UserBrief       `json:"user"`
	Category        string          `json:"category"`
	Type            string          `json:"type"`
	Difficulty      int             `json:"difficulty"`
	SpoilerLevel    string          `json:"spoiler_level"`
	Question        string          `json:"question"`
	QuestionHtml    string          `json:"question_html"`
	DescriptionHtml string          `json:"description_html"`
	Content         json.RawMessage `json:"content"`
	QuizStats
	Created     string              `json:"created"`
	Updated     string              `json:"updated"`
	HideGalgame bool                `json:"hide_galgame"`
	Galgames    []QuizGalgameDetail `json:"galgames"`
	IsAuthor    bool                `json:"is_author"`
	IsFavorited bool                `json:"is_favorited"`
	MyAnswer    *QuizAnswerResult   `json:"my_answer"`
}

type QuizAnswerResult struct {
	Submitted     json.RawMessage `json:"submitted"`
	IsCorrect     *bool           `json:"is_correct"`
	Answer        json.RawMessage `json:"answer"`
	Explanation   string          `json:"explanation"`
	QualityRating *int            `json:"quality_rating"`
	RewardDelta   int             `json:"reward_delta"`
}

type CreatedQuiz struct {
	ID           int       `json:"id"`
	User         UserBrief `json:"user"`
	Category     string    `json:"category"`
	Type         string    `json:"type"`
	Difficulty   int       `json:"difficulty"`
	SpoilerLevel string    `json:"spoiler_level"`
	QuestionHtml string    `json:"question_html"`
	QuizStats
	Created string `json:"created"`
	Updated string `json:"updated"`
}

type QuizQualityResult struct {
	QualityAverage float64 `json:"quality_average"`
	QualityCount   int     `json:"quality_count"`
	QualityRating  int     `json:"quality_rating"`
}

type QuizGalgameSearchRequest struct {
	Keywords string `query:"keywords" validate:"required,min=1"`
}

type QuizGalgameOption struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Banner          string   `json:"banner"`
	BannerThumbhash string   `json:"banner_thumbhash,omitempty"`
	Officials       []string `json:"officials"`
}
