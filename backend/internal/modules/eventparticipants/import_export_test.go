package eventparticipants

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestParseCSVWithRussianHeadersAndSemicolon(t *testing.T) {
	t.Parallel()
	data := "ФИО;Дата рождения;Номер профсоюзного билета;Штрихкод СКС\n" +
		"Иванов Иван;02.01.2000;U-001;S-001\n" +
		"Петров Пётр;1999-12-31;;S-002\n"
	records, err := ParseImportFile("participants.csv", bytes.NewBufferString(data))
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	if records[0].Line != 2 || records[0].UnionCardNumber != "U-001" || records[1].SKSBarcode != "S-002" {
		t.Fatalf("unexpected records: %#v", records)
	}
	if _, err := parseBirthDate(records[0].BirthDate); err != nil {
		t.Fatalf("russian date: %v", err)
	}
}

func TestParseCSVWithSocialLinks(t *testing.T) {
	t.Parallel()
	data := "ФИО;Дата рождения;ВК;ТГ\nИванов Иван;02.01.2000;vk.com/durov;@durov\n"
	records, err := ParseImportFile("participants.csv", bytes.NewBufferString(data))
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(records) != 1 || records[0].VKURL != "vk.com/durov" || records[0].TelegramURL != "@durov" {
		t.Fatalf("social records: %#v", records)
	}
}

func TestParseCSVWithDirectionColumn(t *testing.T) {
	t.Parallel()
	data := "ФИО;Дата рождения;Направление подготовки\nИванов Иван;02.01.2000;IT\n"
	records, err := ParseImportFile("participants.csv", bytes.NewBufferString(data))
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(records) != 1 || records[0].Direction != "IT" {
		t.Fatalf("direction record: %#v", records)
	}
}

func TestParseCSVWithMultilineDirectionHeader(t *testing.T) {
	t.Parallel()
	data := "ФИО,Дата рождения,\"Направление\nподготовки\"\nИванов Иван,2000-01-02,Медиа\n"
	records, err := ParseImportFile("participants.csv", bytes.NewBufferString(data))
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(records) != 1 || records[0].Direction != "Медиа" {
		t.Fatalf("multiline header: %#v", records)
	}
}

func TestXLSXExportImportRoundTrip(t *testing.T) {
	t.Parallel()
	union := "U-001"
	direction := "IT"
	vk := "https://vk.com/durov"
	tg := "https://t.me/durov"
	participants := []Participant{{
		ID: "p1", ContestID: "c1", FullName: "Иванов Иван",
		BirthDate:       time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
		UnionCardNumber: &union, DirectionName: &direction, Status: StatusActive,
		VKURL: &vk, TelegramURL: &tg,
	}}
	file, err := exportXLSX(participants)
	if err != nil {
		t.Fatalf("exportXLSX: %v", err)
	}
	records, err := ParseImportFile(file.Name, bytes.NewReader(file.Data))
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(records) != 1 || records[0].FullName != "Иванов Иван" || records[0].BirthDate != "2000-01-02" {
		t.Fatalf("round-trip records: %#v", records)
	}
	if records[0].Direction != "IT" {
		t.Fatalf("round-trip direction: %#v", records[0])
	}
	if records[0].VKURL != vk || records[0].TelegramURL != tg {
		t.Fatalf("round-trip social: %#v", records[0])
	}
}

func TestImportSummaryAndMatchingRules(t *testing.T) {
	t.Parallel()
	birth := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	uExact, sExact := "U-EXACT", "S-EXACT"
	exact := &Participant{
		ID: "exact", ContestID: "contest-1", FullName: "Иванов Иван",
		FullNameNormalized: "иванов иван", BirthDate: birth,
		UnionCardNumber: &uExact, SKSBarcode: &sExact, Status: StatusActive,
	}
	uUpdate := "U-UPDATE"
	update := &Participant{
		ID: "update", ContestID: "contest-1", FullName: "Старое Имя",
		FullNameNormalized: "старое имя", BirthDate: birth,
		UnionCardNumber: &uUpdate, Status: StatusBlocked,
	}
	uConflict, sConflict := "U-CONFLICT", "S-CONFLICT"
	conflictA := &Participant{ID: "a", ContestID: "contest-1", Status: StatusActive}
	conflictB := &Participant{ID: "b", ContestID: "contest-1", Status: StatusActive}
	dupA := Participant{ID: "dup-a", ContestID: "contest-1", Status: StatusActive}
	dupB := Participant{ID: "dup-b", ContestID: "contest-1", Status: StatusActive}

	repo := &fakeRepo{
		unionByValue: map[string]*Participant{uExact: exact, uUpdate: update, uConflict: conflictA},
		sksByValue:   map[string]*Participant{sExact: exact, sConflict: conflictB},
		nameMatchesByNormalized: map[string][]Participant{
			"дубль человек": {dupA, dupB},
		},
	}
	svc := testService(repo, &fakeAudit{})
	result, err := svc.Import(context.Background(), Actor{UserID: "owner"}, "contest-1", []ImportRecord{
		{Line: 2, FullName: "Новый Человек", BirthDate: "01.01.2001"},
		{Line: 3, FullName: exact.FullName, BirthDate: "2000-01-02", UnionCardNumber: uExact, SKSBarcode: sExact},
		{Line: 4, FullName: "Новое Имя", BirthDate: "2000-01-02", UnionCardNumber: uUpdate},
		{Line: 5, FullName: "Ошибка Даты", BirthDate: "not-a-date"},
		{Line: 6, FullName: "Конфликт", BirthDate: "2000-01-02", UnionCardNumber: uConflict, SKSBarcode: sConflict},
		{Line: 7, FullName: "Дубль Человек", BirthDate: "2000-01-02"},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Added != 1 || result.Updated != 1 || result.Errors != 2 || result.Duplicates != 2 {
		t.Fatalf("summary = added %d updated %d errors %d duplicates %d",
			result.Added, result.Updated, result.Errors, result.Duplicates)
	}
	if len(repo.created) != 1 || len(repo.updated) != 1 || repo.updated[0].ID != update.ID {
		t.Fatalf("writes: created=%d updated=%#v", len(repo.created), repo.updated)
	}
	if repo.updated[0].UnionCardNumber == nil || *repo.updated[0].UnionCardNumber != uUpdate {
		t.Fatal("existing identifier was not preserved")
	}
}

func TestImportAssignsDirectionFromName(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	svc := testService(repo, &fakeAudit{})
	result, err := svc.Import(context.Background(), Actor{UserID: "owner"}, "contest-1", []ImportRecord{
		{Line: 2, FullName: "Новый Человек", BirthDate: "01.01.2001", Direction: "IT"},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Added != 1 || len(repo.created) != 1 || repo.created[0].DirectionID == nil {
		t.Fatalf("created=%#v result=%#v", repo.created, result)
	}
	if *repo.created[0].DirectionID != "dir-it" {
		t.Fatalf("direction id = %q", *repo.created[0].DirectionID)
	}
	if result.Rows[0].Direction != "IT" {
		t.Fatalf("result direction = %q", result.Rows[0].Direction)
	}
}

func TestImportRejectsInvalidSocialURL(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{}
	svc := testService(repo, &fakeAudit{})
	result, err := svc.Import(context.Background(), Actor{UserID: "owner"}, "contest-1", []ImportRecord{
		{Line: 2, FullName: "Новый Человек", BirthDate: "01.01.2001", VKURL: "https://evil.com/durov"},
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Errors != 1 || result.Rows[0].Message != "Некорректная ссылка ВКонтакте" {
		t.Fatalf("invalid vk import: %#v", result)
	}
}

func TestParseImportRejectsMissingRequiredHeader(t *testing.T) {
	t.Parallel()
	_, err := ParseImportFile("participants.csv", bytes.NewBufferString("full_name,barcode\nИванов,1\n"))
	if err == nil {
		t.Fatal("missing birth_date header must fail")
	}
}
