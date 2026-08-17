package eventtasks

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"
)

type Repository interface {
	Can(ctx context.Context, userID, contestID, permission string) (bool, error)
	List(ctx context.Context, contestID string) ([]Task, error)
	ParticipantList(ctx context.Context, contestID, participantID string) ([]Task, error)
	Get(ctx context.Context, contestID, taskID string) (*Task, error)
	Create(ctx context.Context, contestID string, input TaskInput) (*Task, error)
	Update(ctx context.Context, contestID, taskID string, input TaskInput) (*Task, error)
	Transition(ctx context.Context, contestID, taskID string, allowedFrom []string, to string) (*Task, error)
	Delete(ctx context.Context, contestID, taskID string) error
	SetImage(ctx context.Context, contestID, taskID string, imageKey *string) (*Task, *string, error)
	ParticipantSubmission(ctx context.Context, contestID, taskID, participantID string) (*Submission, error)
	SubmitAttempt(ctx context.Context, params SubmitParams) (*Submission, error)
	ModerationList(ctx context.Context, contestID, status string) ([]Submission, error)
	SubmissionByID(ctx context.Context, contestID, submissionID string) (*Submission, error)
	Approve(ctx context.Context, actor Actor, contestID, submissionID, comment string) (*ModerationResult, error)
	Reject(ctx context.Context, actor Actor, contestID, submissionID, comment string) (*ModerationResult, error)
	ParticipantAsset(ctx context.Context, participantID, assetID string) (*Asset, error)
	AdminAsset(ctx context.Context, contestID, submissionID, assetID string) (*Asset, error)
}

type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type FileStore interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, key string) error
}

type Service struct {
	repo     Repository
	audit    Auditor
	presign  func(context.Context, string) (string, error)
	now      func() time.Time
	maxImage int64
}

func NewService(repo Repository, audit Auditor, maxImageBytes int64) *Service {
	if maxImageBytes <= 0 {
		maxImageBytes = 10 << 20
	}
	return &Service{repo: repo, audit: audit, now: time.Now, maxImage: maxImageBytes}
}

func (s *Service) SetPresigner(fn func(context.Context, string) (string, error)) {
	s.presign = fn
}

func (s *Service) ensure(ctx context.Context, actor Actor, contestID, permission string) error {
	if actor.IsMega {
		return nil
	}
	allowed, err := s.repo.Can(ctx, actor.UserID, contestID, permission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) List(ctx context.Context, actor Actor, contestID string) ([]Task, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	list, err := s.repo.List(ctx, contestID)
	if err != nil {
		return nil, err
	}
	s.decorateTasks(ctx, list)
	return nonNilTasks(list), nil
}

func (s *Service) ParticipantList(ctx context.Context, contestID, participantID string) ([]Task, error) {
	list, err := s.repo.ParticipantList(ctx, contestID, participantID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	for i := range list {
		list[i].Available = list[i].Status == StatusActive &&
			(list[i].StartsAt == nil || !now.Before(*list[i].StartsAt)) &&
			(list[i].EndsAt == nil || !now.After(*list[i].EndsAt))
		if list[i].Submission != nil {
			decorateParticipantAssetPaths(list[i].Submission)
		}
	}
	s.decorateTasks(ctx, list)
	return nonNilTasks(list), nil
}

func (s *Service) Get(ctx context.Context, actor Actor, contestID, taskID string) (*Task, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	task, err := s.repo.Get(ctx, contestID, taskID)
	if err == nil {
		s.decorateTask(ctx, task)
	}
	return task, err
}

func (s *Service) Create(ctx context.Context, actor Actor, contestID string, input TaskInput) (*Task, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	input, err := validateTaskInput(input)
	if err != nil {
		return nil, err
	}
	task, err := s.repo.Create(ctx, contestID, input)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_TASK_CREATED", "event_task", task.ID,
			map[string]any{"contest_id": contestID})
	}
	return task, err
}

func (s *Service) Update(ctx context.Context, actor Actor, contestID, taskID string, input TaskInput) (*Task, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	input, err := validateTaskInput(input)
	if err != nil {
		return nil, err
	}
	task, err := s.repo.Update(ctx, contestID, taskID, input)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_TASK_UPDATED", "event_task", taskID,
			map[string]any{"contest_id": contestID})
	}
	return task, err
}

func (s *Service) Transition(ctx context.Context, actor Actor, contestID, taskID, action string) (*Task, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	var from []string
	var to string
	switch action {
	case "activate":
		from, to = []string{StatusDraft, StatusDisabled}, StatusActive
	case "disable":
		from, to = []string{StatusActive}, StatusDisabled
	case "archive":
		from, to = []string{StatusDraft, StatusActive, StatusDisabled}, StatusArchived
	default:
		return nil, ErrValidation
	}
	task, err := s.repo.Transition(ctx, contestID, taskID, from, to)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_TASK_STATUS_CHANGED", "event_task", taskID,
			map[string]any{"contest_id": contestID, "to": to})
	}
	return task, err
}

func (s *Service) Delete(ctx context.Context, actor Actor, contestID, taskID string) error {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return err
	}
	err := s.repo.Delete(ctx, contestID, taskID)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_TASK_DELETED", "event_task", taskID,
			map[string]any{"contest_id": contestID})
	}
	return err
}

func (s *Service) ParticipantGet(ctx context.Context, contestID, taskID, participantID string) (*Task, error) {
	task, err := s.repo.Get(ctx, contestID, taskID)
	if err != nil || task.Status != StatusActive {
		if err == nil {
			err = ErrNotFound
		}
		return nil, err
	}
	now := s.now()
	task.Available = (task.StartsAt == nil || !now.Before(*task.StartsAt)) &&
		(task.EndsAt == nil || !now.After(*task.EndsAt))
	submission, subErr := s.repo.ParticipantSubmission(ctx, contestID, taskID, participantID)
	if subErr == nil {
		decorateParticipantAssetPaths(submission)
		task.Submission = submission
	} else if subErr != ErrNotFound {
		return nil, subErr
	}
	s.decorateTask(ctx, task)
	return task, nil
}

func (s *Service) ModerationList(ctx context.Context, actor Actor, contestID, status string) ([]Submission, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionModerate); err != nil {
		return nil, err
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		status = SubmissionPending
	}
	if status != SubmissionPending && status != SubmissionApproved && status != SubmissionRejected {
		return nil, ErrValidation
	}
	list, err := s.repo.ModerationList(ctx, contestID, status)
	return nonNilSubmissions(list), err
}

func (s *Service) ModerationGet(ctx context.Context, actor Actor, contestID, submissionID string) (*Submission, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionModerate); err != nil {
		return nil, err
	}
	submission, err := s.repo.SubmissionByID(ctx, contestID, submissionID)
	if err == nil {
		decorateAdminAssetPaths(submission, contestID)
	}
	return submission, err
}

func (s *Service) Approve(ctx context.Context, actor Actor, contestID, submissionID string, input ModerationInput) (*ModerationResult, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionModerate); err != nil {
		return nil, err
	}
	comment := strings.TrimSpace(input.Comment)
	if len(comment) > 2000 {
		return nil, ErrValidation
	}
	result, err := s.repo.Approve(ctx, actor, contestID, submissionID, comment)
	if result != nil {
		decorateAdminAssetPaths(&result.Submission, contestID)
	}
	return result, err
}

func (s *Service) Reject(ctx context.Context, actor Actor, contestID, submissionID string, input ModerationInput) (*ModerationResult, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionModerate); err != nil {
		return nil, err
	}
	comment := strings.TrimSpace(input.Comment)
	if len(comment) < 3 || len(comment) > 2000 {
		return nil, ErrValidation
	}
	result, err := s.repo.Reject(ctx, actor, contestID, submissionID, comment)
	if result != nil {
		decorateAdminAssetPaths(&result.Submission, contestID)
	}
	return result, err
}

func (s *Service) ParticipantAssetURL(ctx context.Context, participantID, assetID string) (string, error) {
	asset, err := s.repo.ParticipantAsset(ctx, participantID, assetID)
	return s.assetURL(ctx, asset, err)
}

func (s *Service) AdminAssetURL(ctx context.Context, actor Actor, contestID, submissionID, assetID string) (string, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionModerate); err != nil {
		return "", err
	}
	asset, err := s.repo.AdminAsset(ctx, contestID, submissionID, assetID)
	return s.assetURL(ctx, asset, err)
}

func (s *Service) assetURL(ctx context.Context, asset *Asset, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if asset.Type != AssetImage || asset.ObjectKey == nil || s.presign == nil {
		return "", ErrNotFound
	}
	return s.presign(ctx, *asset.ObjectKey)
}

func (s *Service) decorateTasks(ctx context.Context, tasks []Task) {
	for i := range tasks {
		s.decorateTask(ctx, &tasks[i])
	}
}

func (s *Service) decorateTask(ctx context.Context, task *Task) {
	if task.ImageKey != nil && s.presign != nil {
		if value, err := s.presign(ctx, *task.ImageKey); err == nil {
			task.ImageURL = &value
		}
	}
}

func decorateParticipantAssetPaths(submission *Submission) {
	for i := range submission.Attempts {
		for j := range submission.Attempts[i].Assets {
			asset := &submission.Attempts[i].Assets[j]
			if asset.Type == AssetImage {
				path := fmt.Sprintf("/participant/task-assets/%s", asset.ID)
				asset.DownloadPath = &path
			}
		}
	}
}

func decorateAdminAssetPaths(submission *Submission, contestID string) {
	for i := range submission.Attempts {
		for j := range submission.Attempts[i].Assets {
			asset := &submission.Attempts[i].Assets[j]
			if asset.Type == AssetImage {
				value := fmt.Sprintf("/admin/contests/%s/task-submissions/%s/assets/%s",
					contestID, submission.ID, asset.ID)
				asset.DownloadPath = &value
			}
		}
	}
}

func validateTaskInput(input TaskInput) (TaskInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Icon != nil {
		icon := strings.TrimSpace(*input.Icon)
		if icon == "" {
			input.Icon = nil
		} else {
			input.Icon = &icon
		}
	}
	if input.Title == "" || input.Description == "" || len(input.Title) > 300 ||
		len(input.Description) > 20_000 || input.Points <= 0 || input.Points > 1_000_000 ||
		(input.Icon != nil && len(*input.Icon) > 64) {
		return TaskInput{}, ErrValidation
	}
	if input.StartsAt != nil && input.EndsAt != nil && !input.StartsAt.Before(*input.EndsAt) {
		return TaskInput{}, ErrValidation
	}
	seen := map[string]bool{}
	types := make([]string, 0, len(input.AllowedSubmissionTypes))
	for _, raw := range input.AllowedSubmissionTypes {
		value := strings.ToUpper(strings.TrimSpace(raw))
		if (value != AssetImage && value != AssetLink) || seen[value] {
			continue
		}
		seen[value] = true
		types = append(types, value)
	}
	if len(types) == 0 {
		return TaskInput{}, ErrValidation
	}
	input.AllowedSubmissionTypes = types
	return input, nil
}

func validateLink(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if len(value) == 0 || len(value) > 2048 {
		return "", ErrValidation
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrValidation
	}
	return parsed.String(), nil
}

func safeName(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	if name == "" || name == "." {
		return "image"
	}
	return name
}

func nonNilTasks(list []Task) []Task {
	if list == nil {
		return []Task{}
	}
	return list
}

func nonNilSubmissions(list []Submission) []Submission {
	if list == nil {
		return []Submission{}
	}
	return list
}
