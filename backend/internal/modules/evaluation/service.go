package evaluation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type ContestAccess interface {
	ContestViewable(ctx context.Context, userID, contestID string, isMega bool) (bool, error)
	ContestEditable(ctx context.Context, userID, contestID string, isMega bool) (bool, error)
}

type PasswordVerifier interface {
	VerifyUserPassword(ctx context.Context, userID, password string) error
}

type Service struct {
	repo      *Repo
	access    ContestAccess
	audit     Auditor
	passwords PasswordVerifier
	enabled   bool
	hub       *Hub
	presign   func(context.Context, string) (string, error)
}

func NewService(repo *Repo, access ContestAccess, audit Auditor, enabled bool) *Service {
	return &Service{repo: repo, access: access, audit: audit, enabled: enabled, hub: NewHub()}
}

func (s *Service) SetPresigner(fn func(context.Context, string) (string, error)) {
	s.presign = fn
}

func (s *Service) SetPasswordVerifier(v PasswordVerifier) {
	s.passwords = v
}

func (s *Service) decorateAvatars(ctx context.Context, list []LiveContestant) {
	for i := range list {
		list[i].AvatarURL = s.avatarURL(ctx, list[i].AvatarKey)
	}
}

func (s *Service) avatarURL(ctx context.Context, key *string) *string {
	if key == nil || *key == "" || s.presign == nil {
		return nil
	}
	u, err := s.presign(ctx, *key)
	if err != nil || u == "" {
		return nil
	}
	return &u
}

func (s *Service) ensureEnabled() error {
	if !s.enabled {
		return ErrDisabled
	}
	return nil
}

func (s *Service) ensureView(ctx context.Context, a Actor, contestID string) error {
	ok, err := s.access.ContestViewable(ctx, a.UserID, contestID, a.IsMega)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ensureEdit(ctx context.Context, a Actor, contestID string) error {
	ok, err := s.access.ContestEditable(ctx, a.UserID, contestID, a.IsMega)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) challengeForView(ctx context.Context, a Actor, challengeID string) (*challengeRef, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	ch, err := s.repo.ChallengeByID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureView(ctx, a, ch.ContestID); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *Service) challengeForEdit(ctx context.Context, a Actor, challengeID string) (*challengeRef, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	ch, err := s.repo.ChallengeByID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureEdit(ctx, a, ch.ContestID); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *Service) Get(ctx context.Context, a Actor, challengeID string) (*Scheme, error) {
	if _, err := s.challengeForView(ctx, a, challengeID); err != nil {
		return nil, err
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.hydrateOperator(ctx, scheme)
	return scheme, nil
}

func (s *Service) Put(ctx context.Context, a Actor, challengeID string, in SchemeInput) (*Scheme, error) {
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSchemeWritable(ctx, challengeID); err != nil {
		return nil, err
	}
	in, err = normalizeScheme(in, ch.Title)
	if err != nil {
		return nil, err
	}
	if in.Type == TypeEliminationLives && in.OperatorUserID != nil {
		ok, err := s.repo.UserIsJury(ctx, *in.OperatorUserID, ch.ContestID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrValidation
		}
	}
	scheme, err := s.repo.UpsertScheme(ctx, challengeID, a.UserID, in)
	if err != nil {
		return nil, err
	}
	if err := s.syncOperator(ctx, ch, scheme); err != nil {
		return nil, err
	}
	s.hydrateOperator(ctx, scheme)
	s.snapshot(ctx, a.UserID, scheme)
	s.clearIncompatibleStageLink(ctx, challengeID, scheme.Type)
	s.audit.Log(ctx, a.UserID, "EVALUATION_SCHEME_UPSERT", "evaluation_scheme", scheme.ID, map[string]any{
		"challenge_id": challengeID, "type": scheme.Type,
	})
	return scheme, nil
}

func (s *Service) AddCriterion(ctx context.Context, a Actor, challengeID string, in CriterionInput) (*Criterion, error) {
	if _, err := s.challengeForEdit(ctx, a, challengeID); err != nil {
		return nil, err
	}
	if err := s.ensureSchemeWritable(ctx, challengeID); err != nil {
		return nil, err
	}
	scheme, err := s.requireScheme(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	in, err = normalizeCriterion(in)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.InsertCriterion(ctx, scheme.ID, in)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_CRITERION_CREATE", "evaluation_criterion", c.ID, map[string]any{
		"challenge_id": challengeID,
	})
	return c, nil
}

func (s *Service) UpdateCriterion(ctx context.Context, a Actor, challengeID, criterionID string, in CriterionInput) (*Criterion, error) {
	if _, err := s.challengeForEdit(ctx, a, challengeID); err != nil {
		return nil, err
	}
	if err := s.ensureSchemeWritable(ctx, challengeID); err != nil {
		return nil, err
	}
	if err := s.criterionBelongs(ctx, challengeID, criterionID); err != nil {
		return nil, err
	}
	in, err := normalizeCriterion(in)
	if err != nil {
		return nil, err
	}
	return s.repo.UpdateCriterion(ctx, criterionID, in)
}

func (s *Service) DeleteCriterion(ctx context.Context, a Actor, challengeID, criterionID string) error {
	if _, err := s.challengeForEdit(ctx, a, challengeID); err != nil {
		return err
	}
	if err := s.ensureSchemeWritable(ctx, challengeID); err != nil {
		return err
	}
	if err := s.criterionBelongs(ctx, challengeID, criterionID); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteCriterion(ctx, criterionID); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_CRITERION_DELETE", "evaluation_criterion", criterionID, map[string]any{
		"challenge_id": challengeID,
	})
	return nil
}

func (s *Service) ReorderCriteria(ctx context.Context, a Actor, challengeID string, ids []string) error {
	if _, err := s.challengeForEdit(ctx, a, challengeID); err != nil {
		return err
	}
	if err := s.ensureSchemeWritable(ctx, challengeID); err != nil {
		return err
	}
	scheme, err := s.requireScheme(ctx, challengeID)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return ErrValidation
	}
	return s.repo.ReorderCriteria(ctx, scheme.ID, ids)
}

func (s *Service) JuryContests(ctx context.Context, userID string) ([]JuryContest, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	return s.repo.ListJuryContests(ctx, userID)
}

func (s *Service) juryChallenge(ctx context.Context, a Actor, challengeID string) (*challengeRef, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	ch, err := s.repo.ChallengeByID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if scheme != nil && ExclusiveChallengeJury(scheme.Type) {
		if a.IsMega {
			return ch, nil
		}
		assigned, err := s.repo.UserIsChallengeJury(ctx, a.UserID, ch.ID)
		if err != nil {
			return nil, err
		}
		if !assigned {
			return nil, ErrNotAssigned
		}
		return ch, nil
	}
	if !a.IsMega {
		remote, err := s.repo.UserIsRemoteJuryInContest(ctx, a.UserID, ch.ContestID)
		if err != nil {
			return nil, err
		}
		if remote {
			return nil, ErrNotAssigned
		}
		ok, err := s.repo.UserIsJury(ctx, a.UserID, ch.ContestID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotAssigned
		}
		restricted, err := s.repo.ChallengeJuryRestricted(ctx, ch.ID)
		if err != nil {
			return nil, err
		}
		if restricted {
			assigned, err := s.repo.UserIsChallengeJury(ctx, a.UserID, ch.ID)
			if err != nil {
				return nil, err
			}
			if !assigned {
				return nil, ErrNotAssigned
			}
		}
	}
	return ch, nil
}

func (s *Service) UserHasRemoteJury(ctx context.Context, userID string) (bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return false, nil
	}
	return s.repo.UserHasRemoteJury(ctx, userID)
}

func (s *Service) SearchRemoteJury(ctx context.Context, a Actor, challengeID, _ string) ([]JuryPerson, error) {
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	scheme, err := s.requireScheme(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if !ExclusiveChallengeJury(scheme.Type) {
		return nil, ErrValidation
	}
	return s.repo.ListContestRemoteJury(ctx, ch.ContestID)
}

func (s *Service) ContestRemoteJury(ctx context.Context, a Actor, challengeID string) ([]JuryPerson, error) {
	ch, err := s.challengeForView(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListContestRemoteJury(ctx, ch.ContestID)
}

// AssertRemoteJury — заочное жюри этого испытания (для чтения работ и материалов).
func (s *Service) AssertRemoteJury(ctx context.Context, a Actor, challengeID string) error {
	_, err := s.juryChallenge(ctx, a, challengeID)
	if err != nil {
		return err
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotAssigned
	}
	if err != nil {
		return err
	}
	if !ExclusiveChallengeJury(scheme.Type) {
		return ErrNotAssigned
	}
	return nil
}

func (s *Service) ensureSchemeWritable(ctx context.Context, challengeID string) error {
	state, err := s.repo.SessionState(ctx, challengeID)
	if err != nil {
		return err
	}
	if sessionLocked(state) {
		return ErrSchemeLocked
	}
	return nil
}

func (s *Service) requireScheme(ctx context.Context, challengeID string) (*Scheme, error) {
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrValidation
	}
	return scheme, err
}

func (s *Service) criterionBelongs(ctx context.Context, challengeID, criterionID string) error {
	scheme, err := s.requireScheme(ctx, challengeID)
	if err != nil {
		return err
	}
	sid, err := s.repo.CriterionSchemeID(ctx, criterionID)
	if err != nil {
		return err
	}
	if sid != scheme.ID {
		return ErrNotFound
	}
	return nil
}

func (s *Service) snapshot(ctx context.Context, actorID string, scheme *Scheme) {
	if scheme == nil {
		return
	}
	body, err := json.Marshal(schemeJSON(scheme))
	if err != nil {
		return
	}
	_ = s.repo.SnapshotScheme(ctx, scheme.ID, actorID, body)
}

func normalizeScheme(in SchemeInput, challengeTitle string) (SchemeInput, error) {
	in.Type = strings.ToUpper(strings.TrimSpace(in.Type))
	if !validTypes[in.Type] {
		return in, ErrValidation
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = challengeTitle
	}
	if in.Name == "" {
		in.Name = "Схема оценивания"
	}
	in.ScoringUnit = strings.ToUpper(strings.TrimSpace(in.ScoringUnit))
	if in.ScoringUnit == "" {
		in.ScoringUnit = defaultUnit(in.Type)
	}
	if !validUnits[in.ScoringUnit] {
		return in, ErrValidation
	}
	in.CorridorMode = strings.ToUpper(strings.TrimSpace(in.CorridorMode))
	if in.CorridorMode == "" {
		in.CorridorMode = CorridorNone
	}
	if !validCorridors[in.CorridorMode] {
		return in, ErrValidation
	}
	in.ResultVisibility = strings.ToUpper(strings.TrimSpace(in.ResultVisibility))
	if in.ResultVisibility == "" {
		in.ResultVisibility = VisibilityAdminOnly
	}
	if !validVisibility[in.ResultVisibility] {
		return in, ErrValidation
	}
	in.EditPolicy = strings.ToUpper(strings.TrimSpace(in.EditPolicy))
	if in.EditPolicy == "" {
		in.EditPolicy = EditWhileActive
	}
	if !validEditPolicy[in.EditPolicy] {
		return in, ErrValidation
	}
	if in.Type == TypeEliminationLives {
		in.ScoringUnit = UnitLives
		in.CorridorMode = CorridorNone
		lives := DefaultStartingLives
		if in.StartingLives != nil {
			lives = *in.StartingLives
		}
		if lives < MinStartingLives || lives > MaxStartingLives {
			return in, ErrValidation
		}
		in.StartingLives = &lives
		if in.OperatorUserID != nil {
			op := strings.TrimSpace(*in.OperatorUserID)
			if op == "" {
				in.OperatorUserID = nil
			} else {
				in.OperatorUserID = &op
			}
		}
	} else {
		in.StartingLives = nil
		in.OperatorUserID = nil
	}
	if in.Type == TypeNumericResult {
		in.ScoringUnit = UnitPoints
		in.CorridorMode = CorridorNone
		min := float64(MinNumericScore)
		in.MinScore = &min
		if in.MaxScore == nil || *in.MaxScore <= 0 || *in.MaxScore > MaxNumericScore {
			return in, ErrValidation
		}
		if *in.MinScore > *in.MaxScore {
			return in, ErrValidation
		}
	}
	if in.Type == TypeRemoteCriteria {
		in.ScoringUnit = UnitPoints
	}
	if in.MinScore != nil && in.MaxScore != nil && *in.MinScore > *in.MaxScore {
		return in, ErrValidation
	}
	in.SettingsJSON = encodeSchemeSettings(in)
	return in, nil
}

func operatorIDOf(s *Scheme) string {
	if s == nil {
		return ""
	}
	if s.OperatorUserID != nil && strings.TrimSpace(*s.OperatorUserID) != "" {
		return strings.TrimSpace(*s.OperatorUserID)
	}
	if len(s.SettingsJSON) == 0 {
		return ""
	}
	var parsed struct {
		OperatorUserID *string `json:"operator_user_id"`
	}
	if json.Unmarshal(s.SettingsJSON, &parsed) != nil || parsed.OperatorUserID == nil {
		return ""
	}
	return strings.TrimSpace(*parsed.OperatorUserID)
}

func (s *Service) ContestJury(ctx context.Context, a Actor, challengeID string) ([]JuryPerson, error) {
	ch, err := s.challengeForView(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListContestJury(ctx, ch.ContestID, ch.ID)
}

func (s *Service) ContestRoleJury(ctx context.Context, a Actor, challengeID string) ([]JuryPerson, error) {
	ch, err := s.challengeForView(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListContestRoleJury(ctx, ch.ContestID)
}

func (s *Service) JuryScope(ctx context.Context, a Actor, challengeID string) (string, error) {
	ch, err := s.challengeForView(ctx, a, challengeID)
	if err != nil {
		return "", err
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, ch.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", err
	}
	if scheme != nil && ExclusiveChallengeJury(scheme.Type) {
		return "CHALLENGE", nil
	}
	restricted, err := s.repo.ChallengeJuryRestricted(ctx, ch.ID)
	if err != nil {
		return "", err
	}
	if restricted {
		return "CHALLENGE", nil
	}
	return "CONTEST", nil
}

func (s *Service) hydrateOperator(ctx context.Context, scheme *Scheme) {
	if scheme == nil || scheme.Type != TypeEliminationLives {
		return
	}
	op, err := s.repo.ChallengeOperator(ctx, scheme.ChallengeID)
	if err == nil && op != nil {
		scheme.OperatorUserID = op
		return
	}
	if id := operatorIDOf(scheme); id != "" {
		scheme.OperatorUserID = &id
	}
}

func (s *Service) syncOperator(ctx context.Context, ch *challengeRef, scheme *Scheme) error {
	if scheme == nil || scheme.Type != TypeEliminationLives {
		return s.repo.SetChallengeOperator(ctx, ch.ContestID, ch.ID, "")
	}
	op := operatorIDOf(scheme)
	if scheme.OperatorUserID != nil {
		op = strings.TrimSpace(*scheme.OperatorUserID)
	}
	if op == "" {
		return s.repo.SetChallengeOperator(ctx, ch.ContestID, ch.ID, "")
	}
	ok, err := s.repo.UserIsJury(ctx, op, ch.ContestID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrValidation
	}
	return s.repo.SetChallengeOperator(ctx, ch.ContestID, ch.ID, op)
}

func encodeSchemeSettings(in SchemeInput) []byte {
	if in.Type != TypeEliminationLives {
		return []byte("{}")
	}
	raw := map[string]any{}
	if in.StartingLives != nil {
		raw["starting_lives"] = *in.StartingLives
	}
	if in.OperatorUserID != nil && strings.TrimSpace(*in.OperatorUserID) != "" {
		raw["operator_user_id"] = strings.TrimSpace(*in.OperatorUserID)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return []byte("{}")
	}
	return body
}

func normalizeCriterion(in CriterionInput) (CriterionInput, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return in, ErrValidation
	}
	if in.Weight <= 0 {
		in.Weight = 1
	}
	if in.MaxScore == 0 && in.MinScore == 0 {
		in.MinScore, in.MaxScore = 1, 10
	}
	if in.MinScore > in.MaxScore {
		return in, ErrValidation
	}
	for i := range in.Bands {
		in.Bands[i].Description = strings.TrimSpace(in.Bands[i].Description)
		if in.Bands[i].Description == "" || in.Bands[i].MinScore > in.Bands[i].MaxScore {
			return in, ErrValidation
		}
	}
	return in, nil
}
