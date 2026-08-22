package evaluation

import (
	"context"
	"errors"
	"strings"
)

func (s *Service) StageLinkView(ctx context.Context, a Actor, challengeID string) (*StageLinkView, error) {
	ch, err := s.challengeForView(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	view := &StageLinkView{RemoteOptions: []ChallengeOption{}}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if scheme != nil && ExclusiveChallengeJury(scheme.Type) {
		link, err := s.repo.StageLinkByRemote(ctx, ch.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		if link != nil {
			view.LinkedFrom = &ChallengeOption{ID: link.MainChallengeID, Title: link.MainTitle}
			view.Link = link
		}
		return view, nil
	}
	if scheme == nil {
		return view, nil
	}
	opts, err := s.repo.ListRemoteStageOptions(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	view.RemoteOptions = opts
	link, err := s.repo.StageLinkByMain(ctx, ch.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	view.Link = link
	return view, nil
}

func (s *Service) PutStageLink(ctx context.Context, a Actor, challengeID string, in StageLinkInput) (*StageLinkView, error) {
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	scheme, err := s.requireScheme(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if ExclusiveChallengeJury(scheme.Type) {
		return nil, ErrValidation
	}
	remoteID := ""
	if in.RemoteChallengeID != nil {
		remoteID = strings.TrimSpace(*in.RemoteChallengeID)
	}
	if remoteID == "" {
		if err := s.repo.DeleteStageLinkByMain(ctx, ch.ID); err != nil {
			return nil, err
		}
		s.audit.Log(ctx, a.UserID, "EVALUATION_STAGE_UNLINK", "challenge", challengeID, map[string]any{
			"contest_id": ch.ContestID,
		})
		return s.StageLinkView(ctx, a, challengeID)
	}
	mode := strings.ToUpper(strings.TrimSpace(in.CombineMode))
	if mode != CombineRankSum && mode != CombineScoreSum {
		return nil, ErrValidation
	}
	if in.MainWeight <= 0 || in.RemoteWeight <= 0 || in.MainWeight > 1000 || in.RemoteWeight > 1000 {
		return nil, ErrValidation
	}
	remote, err := s.repo.ChallengeByID(ctx, remoteID)
	if err != nil {
		return nil, ErrValidation
	}
	if remote.ContestID != ch.ContestID || remote.ID == ch.ID || remote.Status == "ARCHIVED" {
		return nil, ErrValidation
	}
	remoteScheme, err := s.repo.SchemeByChallenge(ctx, remote.ID)
	if err != nil {
		return nil, ErrValidation
	}
	if !ExclusiveChallengeJury(remoteScheme.Type) {
		return nil, ErrValidation
	}
	existing, err := s.repo.StageLinkByRemote(ctx, remote.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil && existing.MainChallengeID != ch.ID {
		return nil, ErrValidation
	}
	if _, err := s.repo.UpsertStageLink(ctx, ch.ContestID, ch.ID, remote.ID, in.MainWeight, in.RemoteWeight, mode); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_STAGE_LINK", "challenge", challengeID, map[string]any{
		"contest_id":          ch.ContestID,
		"remote_challenge_id": remote.ID,
		"main_weight":         in.MainWeight,
		"remote_weight":       in.RemoteWeight,
		"combine_mode":        mode,
	})
	return s.StageLinkView(ctx, a, challengeID)
}

func (s *Service) clearIncompatibleStageLink(ctx context.Context, challengeID, schemeType string) {
	if ExclusiveChallengeJury(schemeType) {
		_ = s.repo.DeleteStageLinkByMain(ctx, challengeID)
		return
	}
	_ = s.repo.DeleteStageLinkByRemote(ctx, challengeID)
}

type standing struct {
	UserID   string
	FullName string
	Score    *float64
	Rank     *int
}

func stageScoreOf(c ScoreboardContestant) *float64 {
	if c.NumericScore != nil {
		return c.NumericScore
	}
	if c.Lives != nil {
		v := float64(*c.Lives)
		return &v
	}
	return c.Sum
}

func standingsFromBoard(board *Scoreboard) map[string]standing {
	out := map[string]standing{}
	if board == nil {
		return out
	}
	for i := range board.Contestants {
		c := board.Contestants[i]
		out[c.UserID] = standing{
			UserID:   c.UserID,
			FullName: c.FullName,
			Score:    stageScoreOf(c),
			Rank:     c.Rank,
		}
	}
	return out
}

func (s *Service) attachCombined(ctx context.Context, ch *challengeRef, board *Scoreboard) error {
	link, err := s.repo.StageLinkByMain(ctx, ch.ID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	remoteCh, err := s.repo.ChallengeByID(ctx, link.RemoteChallengeID)
	if err != nil {
		return nil
	}
	remoteBoard, err := s.buildScoreboard(ctx, remoteCh)
	if err != nil {
		return err
	}
	remote := standingsFromBoard(remoteBoard)
	inputs := make([]combineInput, 0, len(board.Contestants))
	for i := range board.Contestants {
		c := board.Contestants[i]
		in := combineInput{
			UserID:    c.UserID,
			FullName:  c.FullName,
			MainScore: stageScoreOf(c),
			MainRank:  c.Rank,
		}
		if st, ok := remote[c.UserID]; ok {
			in.RemoteScore = st.Score
			in.RemoteRank = st.Rank
		}
		inputs = append(inputs, in)
	}
	board.Combined = &CombinedRanking{
		RemoteChallengeID:    link.RemoteChallengeID,
		RemoteChallengeTitle: link.RemoteTitle,
		MainWeight:           link.MainWeight,
		RemoteWeight:         link.RemoteWeight,
		CombineMode:          link.CombineMode,
		Rows:                 combineStandings(link.CombineMode, link.MainWeight, link.RemoteWeight, inputs),
	}
	return nil
}

type combineInput struct {
	UserID      string
	FullName    string
	MainScore   *float64
	MainRank    *int
	RemoteScore *float64
	RemoteRank  *int
}

func combineStandings(mode string, mainW, remoteW float64, inputs []combineInput) []CombinedRow {
	rows := make([]CombinedRow, len(inputs))
	vals := make([]*float64, len(inputs))
	for i, in := range inputs {
		row := CombinedRow{
			UserID:      in.UserID,
			FullName:    in.FullName,
			MainScore:   in.MainScore,
			MainRank:    in.MainRank,
			RemoteScore: in.RemoteScore,
			RemoteRank:  in.RemoteRank,
		}
		switch mode {
		case CombineRankSum:
			if in.MainRank != nil && in.RemoteRank != nil {
				v := float64(*in.MainRank)*mainW + float64(*in.RemoteRank)*remoteW
				row.Combined = &v
			}
		default:
			if in.MainScore != nil && in.RemoteScore != nil {
				v := *in.MainScore*mainW + *in.RemoteScore*remoteW
				row.Combined = &v
			}
		}
		rows[i] = row
		vals[i] = row.Combined
	}
	var ranks []*int
	if mode == CombineRankSum {
		ranks = competitionRanksLower(vals)
	} else {
		ranks = competitionRanks(vals)
	}
	for i := range rows {
		rows[i].Rank = ranks[i]
	}
	return rows
}

func competitionRanksLower(scores []*float64) []*int {
	inv := make([]*float64, len(scores))
	for i, s := range scores {
		if s == nil {
			continue
		}
		v := -*s
		inv[i] = &v
	}
	return competitionRanks(inv)
}
