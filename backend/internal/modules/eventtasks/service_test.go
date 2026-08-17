package eventtasks

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateTaskInput(t *testing.T) {
	t.Parallel()
	now := time.Now()
	later := now.Add(time.Hour)
	input, err := validateTaskInput(TaskInput{
		Title: "  Волонтёрский отчёт  ", Description: "  Прикрепите результат ",
		Points: 150, StartsAt: &now, EndsAt: &later,
		AllowedSubmissionTypes: []string{"image", "LINK", "IMAGE", "unknown"},
	})
	if err != nil {
		t.Fatalf("validateTaskInput() error = %v", err)
	}
	if input.Title != "Волонтёрский отчёт" || input.Description != "Прикрепите результат" {
		t.Fatalf("values were not normalized: %#v", input)
	}
	if len(input.AllowedSubmissionTypes) != 2 || input.AllowedSubmissionTypes[0] != AssetImage || input.AllowedSubmissionTypes[1] != AssetLink {
		t.Fatalf("submission types = %#v", input.AllowedSubmissionTypes)
	}

	_, err = validateTaskInput(TaskInput{
		Title: "x", Description: "y", Points: 1, StartsAt: &later, EndsAt: &now,
		AllowedSubmissionTypes: []string{AssetImage},
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid schedule error = %v, want ErrValidation", err)
	}
}

func TestValidateLink(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"https://example.org/result", "http://example.org/a?b=c"} {
		if _, err := validateLink(value); err != nil {
			t.Errorf("validateLink(%q) error = %v", value, err)
		}
	}
	for _, value := range []string{"javascript:alert(1)", "//example.org", "ftp://example.org/file", "not a url"} {
		if _, err := validateLink(value); !errors.Is(err, ErrValidation) {
			t.Errorf("validateLink(%q) error = %v, want ErrValidation", value, err)
		}
	}
}

func TestValidateImage(t *testing.T) {
	t.Parallel()
	service := NewService(nil, nil, 1024)
	valid := ImageUpload{
		OriginalName: "proof.webp", ContentType: "image/webp", Size: 100,
		Reader: strings.NewReader("RIFF1234WEBPpayload"), KeySuffix: "nonce",
	}
	prepared, err := service.validateImage(valid)
	if err != nil || prepared.ContentType != "image/webp" {
		t.Fatalf("validateImage(valid) error = %v", err)
	}
	invalid := []ImageUpload{
		{OriginalName: "proof.exe", ContentType: "image/png", Size: 100, Reader: strings.NewReader("not-image"), KeySuffix: "nonce"},
		{OriginalName: "proof.png", ContentType: "application/pdf", Size: 100, Reader: strings.NewReader("not-image"), KeySuffix: "nonce"},
		{OriginalName: "proof.png", ContentType: "image/png", Size: 2048, Reader: strings.NewReader("not-image"), KeySuffix: "nonce"},
		{OriginalName: "proof.png", ContentType: "image/png", Size: 100, Reader: strings.NewReader("not-image"), KeySuffix: "nonce"},
	}
	for _, image := range invalid {
		if _, err := service.validateImage(image); !errors.Is(err, ErrValidation) {
			t.Errorf("validateImage(%#v) error = %v, want ErrValidation", image, err)
		}
	}
}

type submitRepo struct {
	Repository
	task        Task
	existing    *Submission
	submitCalls atomic.Int32
}

func (r *submitRepo) Get(context.Context, string, string) (*Task, error) {
	copy := r.task
	return &copy, nil
}

func (r *submitRepo) ParticipantSubmission(context.Context, string, string, string) (*Submission, error) {
	if r.existing == nil {
		return nil, ErrNotFound
	}
	copy := *r.existing
	return &copy, nil
}

func (r *submitRepo) SubmitAttempt(_ context.Context, params SubmitParams) (*Submission, error) {
	r.submitCalls.Add(1)
	return &Submission{
		ID: "submission", ContestID: params.ContestID, TaskID: params.TaskID,
		EventParticipantID: params.EventParticipantID, Status: SubmissionPending,
		CurrentAttempt: 2,
	}, nil
}

func TestSubmitAllowsOnlyNewOrRejected(t *testing.T) {
	t.Parallel()
	repo := &submitRepo{task: Task{
		ID: "task", Status: StatusActive, AllowedSubmissionTypes: []string{AssetLink},
	}}
	service := NewService(repo, nil, 1024)
	service.now = func() time.Time { return time.Unix(100, 0) }

	if _, err := service.Submit(context.Background(), "contest", "participant", "task", SubmitInput{
		Links: []string{"https://example.org/proof"},
	}, nil); err != nil {
		t.Fatalf("first Submit() error = %v", err)
	}

	repo.existing = &Submission{Status: SubmissionPending}
	if _, err := service.Submit(context.Background(), "contest", "participant", "task", SubmitInput{
		Links: []string{"https://example.org/proof"},
	}, nil); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("pending Submit() error = %v, want ErrInvalidTransition", err)
	}

	repo.existing = &Submission{Status: SubmissionRejected}
	if _, err := service.Submit(context.Background(), "contest", "participant", "task", SubmitInput{
		Links: []string{"https://example.org/fixed"},
	}, nil); err != nil {
		t.Fatalf("rejected Submit() error = %v", err)
	}
	if calls := repo.submitCalls.Load(); calls != 2 {
		t.Fatalf("SubmitAttempt calls = %d, want 2", calls)
	}
}

type concurrentApproveRepo struct {
	Repository
	mu       sync.Mutex
	approved bool
	fresh    int
}

func (r *concurrentApproveRepo) Can(context.Context, string, string, string) (bool, error) {
	return true, nil
}

func (r *concurrentApproveRepo) Approve(_ context.Context, _ Actor, contestID, submissionID, _ string) (*ModerationResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	replayed := r.approved
	if !r.approved {
		r.approved = true
		r.fresh++
	}
	return &ModerationResult{Submission: Submission{
		ID: submissionID, ContestID: contestID, Status: SubmissionApproved,
	}, Replayed: replayed}, nil
}

func TestConcurrentApprovalIsIdempotent(t *testing.T) {
	t.Parallel()
	repo := &concurrentApproveRepo{}
	service := NewService(repo, nil, 1024)
	const requests = 32
	var wg sync.WaitGroup
	var failures atomic.Int32
	var replayed atomic.Int32
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := service.Approve(context.Background(), Actor{UserID: "moderator"}, "contest", "submission", ModerationInput{})
			if err != nil {
				failures.Add(1)
				return
			}
			if result.Replayed {
				replayed.Add(1)
			}
		}()
	}
	wg.Wait()
	if failures.Load() != 0 || repo.fresh != 1 || replayed.Load() != requests-1 {
		t.Fatalf("failures=%d fresh=%d replayed=%d", failures.Load(), repo.fresh, replayed.Load())
	}
}

func TestAssetPathsAreScoped(t *testing.T) {
	t.Parallel()
	participant := Submission{ID: "submission", Attempts: []Attempt{{Assets: []Asset{{ID: "asset", Type: AssetImage}}}}}
	decorateParticipantAssetPaths(&participant)
	if got := *participant.Attempts[0].Assets[0].DownloadPath; got != "/participant/task-assets/asset" {
		t.Fatalf("participant path = %q", got)
	}
	admin := Submission{ID: "submission", Attempts: []Attempt{{Assets: []Asset{{ID: "asset", Type: AssetImage}}}}}
	decorateAdminAssetPaths(&admin, "contest")
	if got := *admin.Attempts[0].Assets[0].DownloadPath; got != "/admin/contests/contest/task-submissions/submission/assets/asset" {
		t.Fatalf("admin path = %q", got)
	}
}
