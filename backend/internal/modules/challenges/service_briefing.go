package challenges

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eazytech/student-leader-cabinet/internal/platform/filevalidation"
)

var briefingExts = map[string]bool{
	"pdf": true, "doc": true, "docx": true, "odt": true, "txt": true, "rtf": true,
	"xls": true, "xlsx": true, "ppt": true, "pptx": true, "zip": true,
	"jpg": true, "jpeg": true, "png": true, "webp": true, "gif": true,
}

func (s *Service) decorateBriefingFiles(ctx context.Context, files []BriefingFile) []BriefingFile {
	if s.presign == nil {
		return files
	}
	for i := range files {
		if files[i].ObjectKey == "" {
			continue
		}
		u, err := s.presign(ctx, files[i].ObjectKey)
		if err != nil || u == "" {
			continue
		}
		files[i].DownloadURL = &u
	}
	return files
}

func (s *Service) loadResolved(ctx context.Context, challengeID, contestantID string) (*Briefing, *BriefingOverride, ResolvedBriefing, error) {
	def, err := s.repo.GetBriefing(ctx, challengeID)
	if err != nil {
		return nil, nil, ResolvedBriefing{}, err
	}
	var ov *BriefingOverride
	if contestantID != "" {
		ov, err = s.repo.GetOverride(ctx, challengeID, contestantID)
		if err != nil {
			return nil, nil, ResolvedBriefing{}, err
		}
	}
	res := resolveBriefing(def, ov, time.Now().UTC())
	res.Files = s.decorateBriefingFiles(ctx, res.Files)
	return def, ov, res, nil
}

func (s *Service) AdminGetBriefing(ctx context.Context, a Actor, challengeID string) (*Briefing, []BriefingContestant, error) {
	c, err := s.AdminGet(ctx, a, challengeID)
	if err != nil {
		return nil, nil, err
	}
	def, err := s.repo.GetBriefing(ctx, challengeID)
	if err != nil {
		return nil, nil, err
	}
	def.Files = s.decorateBriefingFiles(ctx, def.Files)
	people, err := s.repo.ListBriefingContestants(ctx, c.ContestID, challengeID)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	for i := range people {
		if people[i].Override != nil {
			files, err := s.repo.listBriefingFiles(ctx, challengeID, &people[i].Override.ID)
			if err != nil {
				return nil, nil, err
			}
			people[i].Override.Files = s.decorateBriefingFiles(ctx, files)
		}
		res := resolveBriefing(def, people[i].Override, now)
		people[i].Visible = res.Visible
		people[i].PublishAt = res.PublishAt
		people[i].Personalized = res.Personalized || res.Hidden
	}
	return def, people, nil
}

func (s *Service) AdminSaveBriefing(ctx context.Context, a Actor, challengeID string, in BriefingInput) (*Briefing, []BriefingContestant, error) {
	if _, err := s.adminGetForEdit(ctx, a, challengeID); err != nil {
		return nil, nil, err
	}
	body := strings.TrimSpace(in.BodyText)
	if utf8.RuneCountInString(body) > MaxBriefingTextRunes {
		return nil, nil, ErrValidation
	}
	if err := s.repo.UpsertBriefing(ctx, challengeID, a.UserID, body, in.PublishAt); err != nil {
		return nil, nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_BRIEFING_SAVE", "challenge", challengeID, map[string]any{
		"publish_at": in.PublishAt,
	})
	return s.AdminGetBriefing(ctx, a, challengeID)
}

func (s *Service) AdminSaveOverride(ctx context.Context, a Actor, challengeID, contestantID string, in OverrideInput) (*Briefing, []BriefingContestant, error) {
	c, err := s.adminGetForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, nil, err
	}
	ok, err := s.repo.ContestantInContest(ctx, c.ContestID, contestantID)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, ErrValidation
	}
	in.BodyText = strings.TrimSpace(in.BodyText)
	if utf8.RuneCountInString(in.BodyText) > MaxBriefingTextRunes {
		return nil, nil, ErrValidation
	}
	if !in.CustomPublish {
		in.PublishAt = nil
	}
	if !in.CustomText {
		in.BodyText = ""
	}
	if _, err := s.repo.UpsertOverride(ctx, challengeID, contestantID, a.UserID, in); err != nil {
		return nil, nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_BRIEFING_OVERRIDE", "challenge", challengeID, map[string]any{
		"contestant_user_id": contestantID, "hidden": in.Hidden,
	})
	return s.AdminGetBriefing(ctx, a, challengeID)
}

func (s *Service) AdminClearOverride(ctx context.Context, a Actor, challengeID, contestantID string) (*Briefing, []BriefingContestant, error) {
	if _, err := s.adminGetForEdit(ctx, a, challengeID); err != nil {
		return nil, nil, err
	}
	if err := s.repo.DeleteOverride(ctx, challengeID, contestantID); err != nil {
		return nil, nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_BRIEFING_OVERRIDE_CLEAR", "challenge", challengeID, map[string]any{
		"contestant_user_id": contestantID,
	})
	return s.AdminGetBriefing(ctx, a, challengeID)
}

func (s *Service) UploadBriefingFile(ctx context.Context, a Actor, challengeID, contestantID string, up BriefingUpload) (*Briefing, []BriefingContestant, error) {
	c, err := s.adminGetForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, nil, err
	}
	if s.store == nil {
		return nil, nil, ErrNoStorage
	}
	if up.Size <= 0 || up.Size > s.maxBytes {
		return nil, nil, ErrBriefingFile
	}
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(up.OriginalName)), ".")
	if !briefingExts[ext] {
		return nil, nil, ErrBriefingFile
	}
	if err := inspectBriefingImage(&up, ext); err != nil {
		return nil, nil, err
	}
	var overrideID *string
	if contestantID != "" {
		ok, err := s.repo.ContestantInContest(ctx, c.ContestID, contestantID)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, ErrValidation
		}
		ov, err := s.repo.GetOverride(ctx, challengeID, contestantID)
		if err != nil {
			return nil, nil, err
		}
		if ov == nil {
			ov, err = s.repo.UpsertOverride(ctx, challengeID, contestantID, a.UserID, OverrideInput{ReplaceFiles: true})
			if err != nil {
				return nil, nil, err
			}
		} else if !ov.ReplaceFiles {
			in := OverrideInput{
				CustomText: ov.CustomText, BodyText: ov.BodyText,
				CustomPublish: ov.CustomPublish, PublishAt: ov.PublishAt,
				Hidden: ov.Hidden, ReplaceFiles: true,
			}
			ov, err = s.repo.UpsertOverride(ctx, challengeID, contestantID, a.UserID, in)
			if err != nil {
				return nil, nil, err
			}
		}
		overrideID = &ov.ID
	}
	n, err := s.repo.CountBriefingFiles(ctx, challengeID, overrideID)
	if err != nil {
		return nil, nil, err
	}
	if n >= MaxBriefingFiles {
		return nil, nil, ErrBriefingFile
	}
	safe := briefingSafeName(up.OriginalName)
	objectKey := fmt.Sprintf("briefings/%s/%s/%s-%s", c.ContestID, challengeID, up.KeySuffix, safe)
	if err := s.store.Put(ctx, objectKey, up.Reader, up.Size, up.ContentType); err != nil {
		return nil, nil, err
	}
	if _, err := s.repo.InsertBriefingFile(ctx, challengeID, overrideID, a.UserID, c.ContestID, objectKey, up.OriginalName, safe, ext, up.ContentType, up.Size); err != nil {
		_ = s.store.Remove(ctx, objectKey)
		return nil, nil, err
	}
	s.audit.Log(ctx, a.UserID, "CHALLENGE_BRIEFING_FILE", "challenge", challengeID, map[string]any{
		"contestant_user_id": contestantID,
	})
	return s.AdminGetBriefing(ctx, a, challengeID)
}

func (s *Service) DeleteBriefingFile(ctx context.Context, a Actor, challengeID, fileID string) (*Briefing, []BriefingContestant, error) {
	if _, err := s.adminGetForEdit(ctx, a, challengeID); err != nil {
		return nil, nil, err
	}
	key, _, err := s.repo.BriefingFileMeta(ctx, challengeID, fileID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.DeleteBriefingFile(ctx, challengeID, fileID); err != nil {
		return nil, nil, err
	}
	if s.store != nil && key != "" {
		_ = s.store.Remove(ctx, key)
	}
	return s.AdminGetBriefing(ctx, a, challengeID)
}

func (s *Service) ensureJuryReview(ctx context.Context, a Actor, challengeID string) error {
	if s.juryReview == nil {
		return ErrForbidden
	}
	ok, err := s.juryReview(ctx, a.UserID, challengeID, a.IsMega)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) JuryBriefing(ctx context.Context, a Actor, challengeID string) (*ResolvedBriefing, error) {
	if err := s.ensureJuryReview(ctx, a, challengeID); err != nil {
		return nil, err
	}
	if _, err := s.repo.ByID(ctx, challengeID); err != nil {
		return nil, err
	}
	def, err := s.repo.GetBriefing(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	files := s.decorateBriefingFiles(ctx, def.Files)
	has := strings.TrimSpace(def.BodyText) != "" || len(files) > 0
	return &ResolvedBriefing{
		Visible:   has,
		BodyText:  def.BodyText,
		Files:     files,
		PublishAt: def.PublishAt,
	}, nil
}

func (s *Service) ContestantBriefing(ctx context.Context, a Actor, challengeID string) (ResolvedBriefing, error) {
	c, err := s.repo.ByID(ctx, challengeID)
	if err != nil {
		return ResolvedBriefing{}, err
	}
	if err := s.ensureContestantChallenge(ctx, a, c); err != nil {
		return ResolvedBriefing{}, err
	}
	_, _, res, err := s.loadResolved(ctx, challengeID, a.UserID)
	if err != nil {
		return ResolvedBriefing{}, err
	}
	if !contestantSeesBriefing(res) {
		return ResolvedBriefing{Files: []BriefingFile{}}, nil
	}
	if !res.Visible {
		res.BodyText = ""
		res.Files = []BriefingFile{}
	}
	return res, nil
}

func inspectBriefingImage(up *BriefingUpload, ext string) error {
	switch ext {
	case "jpg", "jpeg", "png", "webp", "gif":
		reader, mime, err := filevalidation.InspectImage(up.Reader, up.ContentType, up.OriginalName)
		if err != nil {
			if errors.Is(err, filevalidation.ErrInvalidImage) {
				return ErrBriefingFile
			}
			return err
		}
		up.Reader = reader
		up.ContentType = mime
	}
	return nil
}

func briefingSafeName(name string) string {
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
		return "file"
	}
	return name
}
