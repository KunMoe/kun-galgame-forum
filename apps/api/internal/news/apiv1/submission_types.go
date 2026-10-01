package apiv1

import (
	"context"
	"io"
	"mime/multipart"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/danielgtaylor/huma/v2"
)

const maxNewsImageBytes = 4_000_000

type SubmissionCatalog interface {
	CreateMyNews(ctx context.Context, accessToken string, in catalogclient.NewsSubmissionWrite, idempotencyKey string) (*catalogclient.NewsSubmission, error)
	ListMyNews(ctx context.Context, accessToken, cursor string, limit int) ([]catalogclient.NewsSubmission, string, error)
	GetMyNews(ctx context.Context, accessToken string, id int64) (*catalogclient.NewsSubmission, error)
	PatchMyNews(ctx context.Context, accessToken string, id int64, in catalogclient.NewsSubmissionPatch) (*catalogclient.NewsSubmission, error)
	UploadNewsImageUser(ctx context.Context, accessToken string, r io.Reader, filename string) (*catalogclient.NewsImage, error)
}

var _ SubmissionCatalog = (*catalogclient.Client)(nil)

type NewsSubmission struct {
	Object          string         `json:"object" enum:"news_submission" maxLength:"16" doc:"Type discriminant. Always news_submission."`
	ID              repr.DecimalID `json:"id" doc:"News submission id. Same id space as a news item. JSON string of a decimal integer."`
	State           string         `json:"state" enum:"pending,published,rejected,withdrawn" maxLength:"9" doc:"pending waits for a NextMoe moderator. Only published appears on the news index. rejected and withdrawn are final."`
	NewsSource      string         `json:"news_source" pattern:"^[a-z0-9_]{1,40}$" maxLength:"40" doc:"Key of the source that published the item. community for everything this face creates."`
	Lane            string         `json:"lane" enum:"news,column" maxLength:"6" doc:"news for bulletins, column for longer pieces."`
	Title           string         `json:"title" maxLength:"512" doc:"Headline. Free text; never use it as a decision input."`
	Preview         string         `json:"preview" maxLength:"2000" doc:"The lede the submitter wrote. Free text; never use it as a decision input."`
	ContentMarkdown string         `json:"content_markdown" maxLength:"20000" doc:"The item's own text, CommonMark Markdown. Empty string when the item has none. Free text; never use it as a decision input."`
	SourceURL       string         `json:"source_url" pattern:"^(https?://.*)?$" maxLength:"2048" doc:"The item on the partner's site. Empty string for an original community submission, which has no page elsewhere."`
	Banner          *repr.Image    `json:"banner" doc:"Lead image. null when the item has none."`
	PublishedAt     repr.DateTime  `json:"published_at" doc:"The news service's timestamp for the item. A community submission is stamped when it is submitted."`
}

type BannerImageHash string

func (BannerImageHash) Schema(huma.Registry) *huma.Schema {
	n := 64
	return &huma.Schema{
		Type:        huma.TypeString,
		Pattern:     `^([0-9a-f]{64})?$`,
		MaxLength:   &n,
		Description: "Image-service content hash of the banner, or an empty string for no banner.",
	}
}

type newsImageInput struct {
	RawBody multipart.Form
}

type newsImageOutput struct {
	Location string `header:"Location" format:"uri" maxLength:"512" doc:"The image's absolute URL, the same as url in the body."`
	Body     repr.Image
}

type listMyNewsSubmissionsInput struct {
	Cursor string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque cursor from a previous page's next_cursor. It is bound to limit."`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Page size. 1-100, default 20."`
}

type listMyNewsSubmissionsOutput struct {
	Body repr.List[NewsSubmission]
}

type getMyNewsSubmissionInput struct {
	NewsSubmissionID string `path:"news_submission_id" pattern:"^[0-9]+$" maxLength:"20" doc:"News submission id."`
}

type getMyNewsSubmissionOutput struct {
	Body NewsSubmission
}

type newsSubmissionCreate struct {
	Title           string           `json:"title" minLength:"1" maxLength:"200" doc:"Headline. 1-200 characters after trimming. Free text; never use it as a decision input."`
	Preview         string           `json:"preview" minLength:"1" maxLength:"200" doc:"The lede. 1-200 characters after trimming. Free text; never use it as a decision input."`
	ContentMarkdown string           `json:"content_markdown" required:"false" maxLength:"20000" doc:"The item's own text, CommonMark Markdown. Omitted is an empty string. Free text; never use it as a decision input."`
	SourceURL       string           `json:"source_url" required:"false" maxLength:"1024" pattern:"^(https?://.+)?$" doc:"Canonical link to an original elsewhere. Omitted is an empty string. Absolute http or https, or empty."`
	BannerImageHash *BannerImageHash `json:"banner_image_hash,omitempty" doc:"Banner, by the hash createNewsSubmissionImage returned. Absent or an empty string means no banner. A hash from any other upload is refused at this field with reason UNKNOWN_REFERENCE."`
}

type createNewsSubmissionInput struct {
	Body newsSubmissionCreate
}

type createNewsSubmissionOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new submission, /api/v1/me/news-submissions/{news_submission_id}."`
	Body     NewsSubmission
}

type newsSubmissionPatch struct {
	State           *string          `json:"state,omitempty" enum:"withdrawn" maxLength:"9" doc:"Only withdrawn, and only on its own. Withdrawing is legal only from published."`
	Title           *string          `json:"title,omitempty" minLength:"1" maxLength:"200" doc:"New headline. 1-200 characters after trimming. Free text; never use it as a decision input."`
	Preview         *string          `json:"preview,omitempty" minLength:"1" maxLength:"200" doc:"New lede. 1-200 characters after trimming. Free text; never use it as a decision input."`
	ContentMarkdown *string          `json:"content_markdown,omitempty" maxLength:"20000" doc:"New Markdown. An empty string clears it. Free text; never use it as a decision input."`
	SourceURL       *string          `json:"source_url,omitempty" maxLength:"1024" pattern:"^(https?://.+)?$" doc:"New canonical link. An empty string clears it. Absolute http or https, or empty."`
	BannerImageHash *BannerImageHash `json:"banner_image_hash,omitempty" doc:"New banner, by the hash createNewsSubmissionImage returned. An empty string removes the banner."`
}

type updateNewsSubmissionInput struct {
	NewsSubmissionID string `path:"news_submission_id" pattern:"^[0-9]+$" maxLength:"20" doc:"News submission id."`
	Body             newsSubmissionPatch
}
