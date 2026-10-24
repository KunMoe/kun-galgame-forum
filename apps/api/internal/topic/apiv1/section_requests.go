package apiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	msgModel "kun-galgame-api/internal/message/model"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/topic/ai"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"
)

type TopicSectionRequest struct {
	Object      string          `json:"object" enum:"topic_section_request" maxLength:"21" doc:"Type discriminant. Always topic_section_request."`
	ID          repr.DecimalID  `json:"id" doc:"Request id."`
	TopicID     repr.DecimalID  `json:"topic_id" doc:"The topic to move."`
	TopicTitle  string          `json:"topic_title" maxLength:"233" doc:"The topic's title. Free text; never use it as a decision input."`
	Requester   *repr.UserRef   `json:"requester" doc:"Who asked. null when the automatic check filed it because the model disagreed with the section a topic was published in while the model was unreachable."`
	FromSection TopicSectionKey `json:"from_section" doc:"The topic's section when the request was filed."`
	ToSection   TopicSectionKey `json:"to_section" doc:"The section asked for."`
	Reason      string          `json:"reason" maxLength:"200" doc:"Why, in the requester's words. Free text; never use it as a decision input."`
	State       string          `json:"state" enum:"pending,approved,rejected" maxLength:"8" doc:"pending until staff decide."`
	CreatedAt   repr.DateTime   `json:"created_at" doc:"When it was filed."`
	DecidedAt   *repr.DateTime  `json:"decided_at" doc:"When staff decided. null while pending."`
}

type TopicSectionRequestCreate struct {
	Section TopicSectionKey `json:"section" doc:"The section the topic should be in."`
	Reason  string          `json:"reason" minLength:"1" maxLength:"200" doc:"Why the topic belongs there. Trimmed; blank is TOO_SHORT. Free text; never use it as a decision input."`
}

type TopicSectionRequestDecision struct {
	State string `json:"state" enum:"approved,rejected" maxLength:"8" doc:"approved moves the topic; rejected leaves it."`
}

type createSectionRequestInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    TopicSectionRequestCreate
}

type topicSectionRequestsInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type adminSectionRequestsInput struct {
	State string `query:"state" enum:"pending,approved,rejected" default:"pending" maxLength:"8" doc:"Which requests. Default pending."`
}

type sectionRequestPath struct {
	SectionRequestID string `path:"section_request_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Section request id."`
}

type decideSectionRequestInput struct {
	SectionRequestID string `path:"section_request_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Section request id."`
	Body             TopicSectionRequestDecision
}

type sectionRequestOutput struct {
	Body TopicSectionRequest
}

type sectionRequestsOutput struct {
	Body repr.List[TopicSectionRequest]
}

const (
	topicRequestLimit = 10
	adminRequestLimit = 100
)

func (w *Writes) sectionRequestsReady() *problem.Problem {
	if w == nil || w.reads == nil || w.db() == nil || w.ai == nil || w.ai.repo == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (w *Writes) mapRequests(ctx context.Context, rows []repository.TopicSectionRequest) ([]TopicSectionRequest, *problem.Problem) {
	var uids, tids []int
	for _, r := range rows {
		if r.RequestedBy != nil {
			uids = append(uids, *r.RequestedBy)
		}
		tids = append(tids, r.TopicID)
	}
	users, p := w.reads.lookupUsers(ctx, uids)
	if p != nil {
		return nil, p
	}
	titles := map[int]string{}
	if len(tids) > 0 {
		var ts []struct {
			ID    int
			Title string
		}
		if err := w.db().Table("topic").Select("id, title").Where("id IN ?", tids).Scan(&ts).Error; err != nil {
			return nil, problem.Internal(err)
		}
		for _, t := range ts {
			titles[t.ID] = t.Title
		}
	}
	out := make([]TopicSectionRequest, 0, len(rows))
	for _, r := range rows {
		item := TopicSectionRequest{
			Object:      "topic_section_request",
			ID:          repr.ID(r.ID),
			TopicID:     repr.ID(r.TopicID),
			TopicTitle:  titles[r.TopicID],
			FromSection: TopicSectionKey(r.FromSection),
			ToSection:   TopicSectionKey(r.ToSection),
			Reason:      r.Reason,
			State:       r.State,
			CreatedAt:   repr.Timestamp(r.Created),
			DecidedAt:   repr.TimestampPtr(r.DecidedAt),
		}
		if r.RequestedBy != nil {
			ref := repr.DeletedUserRef(*r.RequestedBy)
			if u, ok := users[*r.RequestedBy]; ok {
				ref = repr.NewUserRef(w.reads.cdn, u)
			}
			item.Requester = &ref
		}
		out = append(out, item)
	}
	return out, nil
}

func (w *Writes) createSectionRequest(ctx context.Context, in *createSectionRequestInput) (*sectionRequestOutput, error) {
	if p := w.sectionRequestsReady(); p != nil {
		return nil, p
	}
	topic, user, p := w.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if user == nil || user.ID != topic.UserID {
		return nil, permissionRequired()
	}
	reason := strings.TrimSpace(in.Body.Reason)
	if reason == "" {
		return nil, validationFailed(tooShort("/reason"))
	}
	stored, err := w.reads.taxonomy.FindSectionNamesByTopicID(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	from := ""
	if len(stored) > 0 {
		from = stored[0]
	}
	to := string(in.Body.Section)
	if to == from {
		return nil, validationFailed(problem.AtPointer("/section", problem.ReasonNotAllowedValue, "the topic is already in this section", nil))
	}
	rules, err := w.reads.taxonomy.FindSectionRules(nil, []string{to})
	if err != nil {
		return nil, problem.Internal(err)
	}
	if rules[to].IsArchived {
		return nil, validationFailed(sectionArchived("/section"))
	}
	uid := user.ID
	req := repository.TopicSectionRequest{TopicID: topic.ID, RequestedBy: &uid, FromSection: from, ToSection: to, Reason: reason}
	err = w.db().Transaction(func(tx *gorm.DB) error {
		pending, err := w.ai.repo.HasPendingRequest(tx, topic.ID)
		if err != nil {
			return err
		}
		if pending {
			return txFail{p: problem.New(problem.CodeAlreadyExists, "This topic already has a request waiting for staff.")}
		}
		return w.ai.repo.CreateRequest(tx, &req)
	})
	if p := txProblem(err); p != nil {
		return nil, p
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := w.mapRequests(ctx, []repository.TopicSectionRequest{req})
	if p != nil {
		return nil, p
	}
	return &sectionRequestOutput{Body: items[0]}, nil
}

func (w *Writes) listTopicSectionRequests(ctx context.Context, in *topicSectionRequestsInput) (*sectionRequestsOutput, error) {
	if p := w.sectionRequestsReady(); p != nil {
		return nil, p
	}
	topic, user, p := w.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if user == nil || (user.ID != topic.UserID && !user.Can(perm.TopicEditAny)) {
		return nil, permissionRequired()
	}
	rows, err := w.ai.repo.TopicRequests(topic.ID, topicRequestLimit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := w.mapRequests(ctx, rows)
	if p != nil {
		return nil, p
	}
	return &sectionRequestsOutput{Body: repr.NewList(items, nil)}, nil
}

func (w *Writes) getAdminSectionRequest(ctx context.Context, in *sectionRequestPath) (*sectionRequestOutput, error) {
	if p := w.sectionRequestsReady(); p != nil {
		return nil, p
	}
	if _, p := w.staff(ctx); p != nil {
		return nil, p
	}
	id, ok := repr.ParseID(repr.DecimalID(in.SectionRequestID))
	if !ok {
		return nil, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
	}
	req, err := w.ai.repo.FindRequest(id)
	if errors.Is(err, repository.ErrRequestNotFound) {
		return nil, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := w.mapRequests(ctx, []repository.TopicSectionRequest{*req})
	if p != nil {
		return nil, p
	}
	return &sectionRequestOutput{Body: items[0]}, nil
}

func (w *Writes) staff(ctx context.Context) (int, *problem.Problem) {
	user := w.caller(ctx)
	if user == nil {
		return 0, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	if !user.Can(perm.TopicEditAny) {
		return 0, permissionRequired()
	}
	return user.ID, nil
}

func (w *Writes) listAdminSectionRequests(ctx context.Context, in *adminSectionRequestsInput) (*sectionRequestsOutput, error) {
	if p := w.sectionRequestsReady(); p != nil {
		return nil, p
	}
	if _, p := w.staff(ctx); p != nil {
		return nil, p
	}
	rows, err := w.ai.repo.Requests(in.State, 0, adminRequestLimit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := w.mapRequests(ctx, rows)
	if p != nil {
		return nil, p
	}
	return &sectionRequestsOutput{Body: repr.NewList(items, nil)}, nil
}

var errRequestDecided = errors.New("request already decided")

func (w *Writes) decideSectionRequest(ctx context.Context, in *decideSectionRequestInput) (*sectionRequestOutput, error) {
	if p := w.sectionRequestsReady(); p != nil {
		return nil, p
	}
	staffID, p := w.staff(ctx)
	if p != nil {
		return nil, p
	}
	id, ok := repr.ParseID(repr.DecimalID(in.SectionRequestID))
	if !ok {
		return nil, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
	}
	var awards []pendingAward
	var decided repository.TopicSectionRequest
	err := w.db().Transaction(func(tx *gorm.DB) error {
		req, err := w.ai.repo.LockRequest(tx, id)
		if err != nil {
			return err
		}
		if req.State != "pending" {
			return errRequestDecided
		}
		var authorID int
		if err := tx.Table("topic").Select("user_id").Where("id = ?", req.TopicID).Scan(&authorID).Error; err != nil {
			return err
		}
		if in.Body.State == "approved" {
			delta, err := w.moveTopicSection(tx, req.TopicID, req.FromSection, req.ToSection)
			if err != nil {
				return err
			}
			if delta != 0 {
				awards = append(awards, topicCostChangedAward(authorID, req.TopicID, delta))
			}
		}
		if err := w.ai.repo.DecideRequest(tx, req.ID, in.Body.State, staffID); err != nil {
			return err
		}
		if err := w.notifyRequestDecision(tx, staffID, authorID, req, in.Body.State); err != nil {
			return err
		}
		decided = *req
		decided.State = in.Body.State
		return nil
	})
	switch {
	case errors.Is(err, repository.ErrRequestNotFound):
		return nil, problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
	case errors.Is(err, errRequestDecided):
		return nil, problem.New(problem.CodeInvalidStateTransition, "The request was already decided.")
	case err != nil:
		return nil, problem.Internal(err)
	}
	w.flushAwards(awards)
	fresh, err := w.ai.repo.FindRequest(decided.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := w.mapRequests(ctx, []repository.TopicSectionRequest{*fresh})
	if p != nil {
		return nil, p
	}
	return &sectionRequestOutput{Body: items[0]}, nil
}

// moveTopicSection files the topic under to and returns the author's
// moemoepoint delta. A topic that left from since the request was filed is
// moved all the same: staff decided on where it should end up.
func (w *Writes) moveTopicSection(tx *gorm.DB, topicID int, from, to string) (int, error) {
	stored, err := w.reads.taxonomy.FindSectionNamesByTopicID(topicID)
	if err != nil {
		return 0, err
	}
	if len(stored) > 0 {
		from = stored[0]
	}
	rules, err := w.reads.taxonomy.FindSectionRules(tx, []string{from, to})
	if err != nil {
		return 0, err
	}
	rule, ok := rules[to]
	if !ok {
		return 0, fmt.Errorf("topic section %q is missing", to)
	}
	if err := w.reads.taxonomy.ReplaceSectionRelations(tx, topicID, []int{rule.ID}); err != nil {
		return 0, err
	}
	if err := tx.Table("topic").Where("id = ?", topicID).Update("category", rule.LegacyCategory).Error; err != nil {
		return 0, err
	}
	return topicSectionFootprint(rule.CostsMoemoepoint) - topicSectionFootprint(rules[from].CostsMoemoepoint), nil
}

func (w *Writes) notifyRequestDecision(tx *gorm.DB, staffID, authorID int, req *repository.TopicSectionRequest, state string) error {
	if w.notify == nil || req.RequestedBy == nil {
		return nil
	}
	labels := map[string]string{}
	var rows []struct{ Name, Label string }
	if err := tx.Table("topic_section").Select("name, label").Where("name IN ?", []string{req.FromSection, req.ToSection}).Scan(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		labels[r.Name] = r.Label
	}
	content := ai.RequestDecisionNotice(labels[req.FromSection], labels[req.ToSection], state == "approved")
	return w.notify.Emit(tx, msgService.Spec{
		SenderID:   staffID,
		ReceiverID: authorID,
		Kind:       msgService.NotifySystemNotice,
		Content:    content,
		TopicID:    req.TopicID,
		Ref:        msgModel.TopicRef(req.TopicID),
	})
}

func registerSectionRequests(api huma.API, w *Writes) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "createTopicSectionRequest",
		Method:        http.MethodPost,
		Path:          "/topics/{topic_id}/section-requests",
		Summary:       "Ask staff to move a topic to another section",
		DefaultStatus: http.StatusCreated,
		Description:   "The author asks staff to move their topic. Staff decide with updateAdminSectionRequest; the author is notified either way. A topic has at most one pending request.",
		Tags:          []string{"topics"},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED when the caller is not the author.",
			404: "NOT_FOUND under the same conditions as getTopic.",
			409: "ALREADY_EXISTS when a request for this topic is still pending.",
			422: "VALIDATION_FAILED for a blank reason, the topic's own section, or an archived section.",
		}),
	}), w.createSectionRequest)
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listTopicSectionRequests",
		Method:      http.MethodGet,
		Path:        "/topics/{topic_id}/section-requests",
		Summary:     "List a topic's section requests",
		Description: "The topic's 10 newest section requests, newest first. For its author and for staff with topic.edit_any. Not paged.",
		Tags:        []string{"topics"},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED for anyone else.",
			404: "NOT_FOUND under the same conditions as getTopic.",
		}),
	}), w.listTopicSectionRequests)
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listAdminSectionRequests",
		Method:      http.MethodGet,
		Path:        "/admin/section-requests",
		Summary:     "List section requests for staff",
		Description: "The 100 newest section requests in the given state, newest first. Needs topic.edit_any, which a Bearer request never carries. Not paged.",
		Tags:        []string{"topics"},
		Responses:   problemResponses(map[int]string{403: "PERMISSION_REQUIRED without topic.edit_any."}),
	}), w.listAdminSectionRequests)
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getAdminSectionRequest",
		Method:      http.MethodGet,
		Path:        "/admin/section-requests/{section_request_id}",
		Summary:     "Get a section request",
		Description: "One section request. Needs topic.edit_any.",
		Tags:        []string{"topics"},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without topic.edit_any.",
			404: "NOT_FOUND when the request does not exist.",
		}),
	}), w.getAdminSectionRequest)
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateAdminSectionRequest",
		Method:      http.MethodPatch,
		Path:        "/admin/section-requests/{section_request_id}",
		Summary:     "Decide a section request",
		Description: "approved moves the topic to to_section and charges or refunds the author the difference between paid and free sections; rejected leaves it. Either way the author is notified. Needs topic.edit_any.",
		Tags:        []string{"topics"},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without topic.edit_any.",
			404: "NOT_FOUND when the request does not exist.",
			409: "INVALID_STATE_TRANSITION when it was already decided.",
		}),
	}), w.decideSectionRequest)
}
