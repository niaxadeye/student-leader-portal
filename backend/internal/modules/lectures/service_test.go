package lectures

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeRepo struct {
	allowed       bool
	permissionErr error
	lastPerm      atomic.Value
	lecture       *Lecture
	scanMu        sync.Mutex
	scanned       bool
	created       atomic.Int32
}

func (f *fakeRepo) Can(_ context.Context, _, _, permission string) (bool, error) {
	f.lastPerm.Store(permission)
	return f.allowed, f.permissionErr
}
func (f *fakeRepo) List(context.Context, string) ([]Lecture, error) { return nil, nil }
func (f *fakeRepo) Get(context.Context, string, string) (*Lecture, error) {
	if f.lecture == nil {
		return nil, ErrNotFound
	}
	copy := *f.lecture
	return &copy, nil
}
func (f *fakeRepo) Create(_ context.Context, contestID string, input LectureInput) (*Lecture, error) {
	return &Lecture{
		ID: "lecture-1", ContestID: contestID, Title: input.Title, Points: input.Points, Status: StatusDraft,
		DirectionIDs: uniqueDirectionIDs(input.DirectionIDs), Directions: []DirectionRef{},
	}, nil
}
func (f *fakeRepo) Update(_ context.Context, contestID, lectureID string, input LectureInput) (*Lecture, error) {
	return &Lecture{
		ID: lectureID, ContestID: contestID, Title: input.Title, Points: input.Points, Status: StatusDraft,
		DirectionIDs: uniqueDirectionIDs(input.DirectionIDs), Directions: []DirectionRef{},
	}, nil
}
func (f *fakeRepo) Transition(_ context.Context, contestID, lectureID, _, to string) (*Lecture, error) {
	return &Lecture{ID: lectureID, ContestID: contestID, Status: to}, nil
}
func (f *fakeRepo) Delete(context.Context, string, string) error { return nil }
func (f *fakeRepo) CreateCode(context.Context, string, string, string, time.Time) error {
	return nil
}
func (f *fakeRepo) ScanAttendance(_ context.Context, params ScanParams) (*ScanResult, error) {
	f.scanMu.Lock()
	defer f.scanMu.Unlock()
	attendance := Attendance{
		ID: "attendance-1", ContestID: params.ContestID, LectureID: params.LectureID,
		EventParticipantID: "participant-1", ParticipantName: "Иванов Иван",
		PointsAwarded: 100, ScannerType: params.ScannerType,
	}
	if f.scanned {
		return &ScanResult{Attendance: attendance, AlreadyChecked: true}, nil
	}
	f.scanned = true
	f.created.Add(1)
	return &ScanResult{Attendance: attendance}, nil
}
func (f *fakeRepo) ListAttendance(context.Context, string, string) ([]Attendance, error) {
	return nil, nil
}
func (f *fakeRepo) ParticipantLectures(context.Context, string, string) ([]ParticipantLecture, error) {
	return nil, nil
}

func newTestCodes(now *time.Time) *CodeManager {
	codes := NewCodeManager(strings.Repeat("s", 32), 45*time.Second)
	codes.now = func() time.Time { return *now }
	codes.random = bytes.NewReader(bytes.Repeat([]byte{7}, 64))
	return codes
}

func TestCodeManagerRoundTripTamperAndExpiry(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	codes := newTestCodes(&now)
	token, nonceHash, expiresAt, err := codes.New()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(token, "participant") || strings.Contains(token, "event") {
		t.Fatal("QR token must not expose participant/event identifiers")
	}
	verified, err := codes.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if verified.NonceHash != nonceHash || !verified.ExpiresAt.Equal(expiresAt.Truncate(time.Second)) {
		t.Fatalf("unexpected verified code: %#v", verified)
	}
	if _, err := codes.Verify(token + "x"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("tampered token: got %v", err)
	}
	now = expiresAt.Add(time.Second)
	if _, err := codes.Verify(token); !errors.Is(err, ErrExpiredCode) {
		t.Fatalf("expired token: got %v", err)
	}
}

func TestLectureValidationAndPermissions(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeRepo{allowed: true}
	service := NewService(repo, newTestCodes(&now), nil)

	if _, err := service.Create(context.Background(), Actor{UserID: "staff"}, "event", LectureInput{
		Title: " ", Points: 100,
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("blank title: got %v", err)
	}
	start, end := now.Add(time.Hour), now
	if _, err := service.Create(context.Background(), Actor{UserID: "staff"}, "event", LectureInput{
		Title: "Лекция", Points: 100, StartsAt: &start, EndsAt: &end,
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid schedule: got %v", err)
	}
	repo.allowed = false
	if _, err := service.List(context.Background(), Actor{UserID: "staff"}, "event"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("permission: got %v", err)
	}
	repo.allowed = true
	if _, err := service.List(context.Background(), Actor{UserID: "staff"}, "event"); err != nil {
		t.Fatalf("scan list: %v", err)
	}
	if got, _ := repo.lastPerm.Load().(string); got != PermissionScan {
		t.Fatalf("list permission: got %q want %q", got, PermissionScan)
	}
}

func TestLectureAllowsParticipant(t *testing.T) {
	t.Parallel()
	dirA, dirB := "dir-a", "dir-b"
	cases := []struct {
		name        string
		restricted  []string
		participant *string
		want        bool
	}{
		{name: "open lecture", restricted: nil, participant: &dirA, want: true},
		{name: "open lecture without direction", restricted: nil, participant: nil, want: true},
		{name: "matching track", restricted: []string{dirA, dirB}, participant: &dirA, want: true},
		{name: "wrong track", restricted: []string{dirA}, participant: &dirB, want: false},
		{name: "restricted and no track", restricted: []string{dirA}, participant: nil, want: false},
		{name: "restricted and empty track", restricted: []string{dirA}, participant: ptr(""), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := lectureAllowsParticipant(tc.restricted, tc.participant); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func ptr(value string) *string { return &value }

func TestConcurrentScanRetriesCreateOneAttendance(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{allowed: true}
	codes := newTestCodes(&now)
	service := NewService(repo, codes, nil)
	token, _, _, err := codes.New()
	if err != nil {
		t.Fatal(err)
	}

	const workers = 32
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	created := atomic.Int32{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := service.Scan(context.Background(), Actor{UserID: "staff"}, "event", "lecture", ScanInput{
				Token: token, ScannerType: ScannerUSB,
			})
			if err != nil {
				errs <- err
				return
			}
			if !result.AlreadyChecked {
				created.Add(1)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("scan failed: %v", err)
	}
	if created.Load() != 1 || repo.created.Load() != 1 {
		t.Fatalf("created responses=%d repository creates=%d", created.Load(), repo.created.Load())
	}
}
