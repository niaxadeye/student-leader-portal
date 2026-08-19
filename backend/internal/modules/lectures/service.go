package lectures

import (
	"context"
	"strings"
	"time"
)

type Repository interface {
	Can(ctx context.Context, userID, contestID, permission string) (bool, error)
	List(ctx context.Context, contestID string) ([]Lecture, error)
	Get(ctx context.Context, contestID, lectureID string) (*Lecture, error)
	Create(ctx context.Context, contestID string, input LectureInput) (*Lecture, error)
	Update(ctx context.Context, contestID, lectureID string, input LectureInput) (*Lecture, error)
	Transition(ctx context.Context, contestID, lectureID, from, to string) (*Lecture, error)
	Delete(ctx context.Context, contestID, lectureID string) error
	CreateCode(ctx context.Context, contestID, participantID, nonceHash string, expiresAt time.Time) error
	ScanAttendance(ctx context.Context, params ScanParams) (*ScanResult, error)
	ListAttendance(ctx context.Context, contestID, lectureID string) ([]Attendance, error)
	ParticipantLectures(ctx context.Context, contestID, participantID string) ([]ParticipantLecture, error)
}

type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type Service struct {
	repo  Repository
	codes *CodeManager
	audit Auditor
	now   func() time.Time
}

func NewService(repo Repository, codes *CodeManager, audit Auditor) *Service {
	return &Service{repo: repo, codes: codes, audit: audit, now: time.Now}
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

func (s *Service) List(ctx context.Context, actor Actor, contestID string) ([]Lecture, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionScan); err != nil {
		return nil, err
	}
	list, err := s.repo.List(ctx, contestID)
	if list == nil {
		list = []Lecture{}
	}
	return list, err
}

func (s *Service) Get(ctx context.Context, actor Actor, contestID, lectureID string) (*Lecture, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionScan); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, contestID, lectureID)
}

func (s *Service) Create(ctx context.Context, actor Actor, contestID string, input LectureInput) (*Lecture, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	input, err := validateInput(input)
	if err != nil {
		return nil, err
	}
	lecture, err := s.repo.Create(ctx, contestID, input)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_LECTURE_CREATED", "lecture", lecture.ID,
			map[string]any{"contest_id": contestID})
	}
	return lecture, err
}

func (s *Service) Update(ctx context.Context, actor Actor, contestID, lectureID string, input LectureInput) (*Lecture, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	current, err := s.repo.Get(ctx, contestID, lectureID)
	if err != nil {
		return nil, err
	}
	if current.Status == StatusFinished {
		return nil, ErrInvalidTransition
	}
	input, err = validateInput(input)
	if err != nil {
		return nil, err
	}
	lecture, err := s.repo.Update(ctx, contestID, lectureID, input)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_LECTURE_UPDATED", "lecture", lectureID,
			map[string]any{"contest_id": contestID})
	}
	return lecture, err
}

func (s *Service) Activate(ctx context.Context, actor Actor, contestID, lectureID string) (*Lecture, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	current, err := s.repo.Get(ctx, contestID, lectureID)
	if err != nil {
		return nil, err
	}
	if current.Status != StatusDraft && current.Status != StatusFinished {
		return nil, ErrInvalidTransition
	}
	return s.applyTransition(ctx, actor, contestID, lectureID, current.Status, StatusActive)
}

func (s *Service) Finish(ctx context.Context, actor Actor, contestID, lectureID string) (*Lecture, error) {
	return s.transition(ctx, actor, contestID, lectureID, StatusActive, StatusFinished)
}

func (s *Service) transition(ctx context.Context, actor Actor, contestID, lectureID, from, to string) (*Lecture, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	return s.applyTransition(ctx, actor, contestID, lectureID, from, to)
}

func (s *Service) applyTransition(ctx context.Context, actor Actor, contestID, lectureID, from, to string) (*Lecture, error) {
	lecture, err := s.repo.Transition(ctx, contestID, lectureID, from, to)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_LECTURE_STATUS_CHANGED", "lecture", lectureID,
			map[string]any{"contest_id": contestID, "from": from, "to": to})
	}
	return lecture, err
}

func (s *Service) Delete(ctx context.Context, actor Actor, contestID, lectureID string) error {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return err
	}
	err := s.repo.Delete(ctx, contestID, lectureID)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "EVENT_LECTURE_DELETED", "lecture", lectureID,
			map[string]any{"contest_id": contestID})
	}
	return err
}

func (s *Service) IssueCode(ctx context.Context, contestID, participantID string) (*QRCode, error) {
	token, hash, expiresAt, err := s.codes.New()
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateCode(ctx, contestID, participantID, hash, expiresAt); err != nil {
		return nil, err
	}
	return &QRCode{Token: token, ExpiresAt: expiresAt, TTL: int(s.codes.TTL().Seconds())}, nil
}

func (s *Service) Scan(ctx context.Context, actor Actor, contestID, lectureID string, input ScanInput) (*ScanResult, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionScan); err != nil {
		return nil, err
	}
	input.Token = strings.TrimSpace(input.Token)
	input.ScannerType = strings.ToUpper(strings.TrimSpace(input.ScannerType))
	if input.Token == "" || !validScanner(input.ScannerType) {
		return nil, ErrValidation
	}
	verified, err := s.codes.Verify(input.Token)
	if err != nil {
		return nil, err
	}
	return s.repo.ScanAttendance(ctx, ScanParams{
		Actor: actor, ContestID: contestID, LectureID: lectureID,
		ScannerType: input.ScannerType, Code: verified, Now: s.now().UTC(),
	})
}

func (s *Service) ListAttendance(ctx context.Context, actor Actor, contestID, lectureID string) ([]Attendance, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionScan); err != nil {
		return nil, err
	}
	list, err := s.repo.ListAttendance(ctx, contestID, lectureID)
	if list == nil {
		list = []Attendance{}
	}
	return list, err
}

func (s *Service) ParticipantLectures(ctx context.Context, contestID, participantID string) ([]ParticipantLecture, error) {
	list, err := s.repo.ParticipantLectures(ctx, contestID, participantID)
	if list == nil {
		list = []ParticipantLecture{}
	}
	return list, err
}

func validateInput(input LectureInput) (LectureInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		if value == "" {
			input.Description = nil
		} else {
			input.Description = &value
		}
	}
	if input.Title == "" || input.Points <= 0 || input.Points > 1_000_000 {
		return LectureInput{}, ErrValidation
	}
	if input.StartsAt != nil && input.EndsAt != nil && !input.StartsAt.Before(*input.EndsAt) {
		return LectureInput{}, ErrValidation
	}
	if input.AttendanceStartsAt != nil && input.AttendanceEndsAt != nil &&
		!input.AttendanceStartsAt.Before(*input.AttendanceEndsAt) {
		return LectureInput{}, ErrValidation
	}
	cleaned := make([]string, 0, len(input.DirectionIDs))
	for _, id := range input.DirectionIDs {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	input.DirectionIDs = uniqueDirectionIDs(cleaned)
	speakers, err := normalizePeopleNames(input.Speakers)
	if err != nil {
		return LectureInput{}, err
	}
	moderators, err := normalizePeopleNames(input.Moderators)
	if err != nil {
		return LectureInput{}, err
	}
	input.Speakers = speakers
	input.Moderators = moderators
	location, err := normalizeOptionalText(input.Location, maxLocationRunes)
	if err != nil {
		return LectureInput{}, err
	}
	input.Location = location
	return input, nil
}

func validScanner(scanner string) bool {
	return scanner == ScannerCamera || scanner == ScannerUSB || scanner == ScannerManual
}
