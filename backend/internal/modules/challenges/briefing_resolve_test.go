package challenges

import (
	"testing"
	"time"
)

func TestResolveBriefingDefaultPublished(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	def := &Briefing{BodyText: "ТЗ", PublishAt: &past, Files: []BriefingFile{}}
	got := resolveBriefing(def, nil, now)
	if !got.Visible || got.BodyText != "ТЗ" || got.Scheduled {
		t.Fatalf("%+v", got)
	}
}

func TestResolveBriefingScheduled(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	def := &Briefing{BodyText: "ТЗ", PublishAt: &future}
	got := resolveBriefing(def, nil, now)
	if got.Visible || !got.Scheduled || got.PublishAt == nil {
		t.Fatalf("%+v", got)
	}
}

func TestResolveBriefingUnpublished(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	def := &Briefing{BodyText: "черновик"}
	got := resolveBriefing(def, nil, now)
	if got.Visible || got.Scheduled {
		t.Fatalf("%+v", got)
	}
}

func TestResolveBriefingHiddenOverride(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	def := &Briefing{BodyText: "всем", PublishAt: &past}
	ov := &BriefingOverride{Hidden: true}
	got := resolveBriefing(def, ov, now)
	if got.Visible || !got.Hidden || got.BodyText != "" {
		t.Fatalf("%+v", got)
	}
}

func TestResolveBriefingEarlyPersonal(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	future := now.Add(2 * time.Hour)
	past := now.Add(-time.Minute)
	def := &Briefing{BodyText: "общее", PublishAt: &future}
	ov := &BriefingOverride{
		CustomPublish: true, PublishAt: &past,
		CustomText: true, BodyText: "Вы выступаете третьим",
	}
	got := resolveBriefing(def, ov, now)
	if !got.Visible || got.BodyText != "Вы выступаете третьим" || !got.Personalized {
		t.Fatalf("%+v", got)
	}
}

func TestResolveBriefingPersonalLater(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	def := &Briefing{BodyText: "общее", PublishAt: &past}
	ov := &BriefingOverride{CustomPublish: true, PublishAt: &future}
	got := resolveBriefing(def, ov, now)
	if got.Visible || !got.Scheduled {
		t.Fatalf("%+v", got)
	}
}

func TestResolveBriefingReplaceFiles(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	def := &Briefing{
		BodyText: "x", PublishAt: &past,
		Files: []BriefingFile{{FileID: "def", OriginalName: "a.pdf"}},
	}
	ov := &BriefingOverride{
		ReplaceFiles: true,
		Files:        []BriefingFile{{FileID: "mine", OriginalName: "b.pdf"}},
	}
	got := resolveBriefing(def, ov, now)
	if len(got.Files) != 1 || got.Files[0].FileID != "mine" {
		t.Fatalf("%+v", got.Files)
	}
}

func TestContestantSeesBriefing(t *testing.T) {
	if contestantSeesBriefing(ResolvedBriefing{}) {
		t.Fatal("empty should hide")
	}
	if !contestantSeesBriefing(ResolvedBriefing{Visible: true}) {
		t.Fatal("visible")
	}
	if !contestantSeesBriefing(ResolvedBriefing{Scheduled: true}) {
		t.Fatal("scheduled")
	}
}
