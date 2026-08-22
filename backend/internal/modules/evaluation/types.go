package evaluation

import "errors"

var (
	ErrNotFound      = errors.New("evaluation not found")
	ErrForbidden     = errors.New("evaluation forbidden")
	ErrValidation    = errors.New("evaluation validation")
	ErrDisabled      = errors.New("evaluation disabled")
	ErrChallenge     = errors.New("challenge not found")
	ErrRevision      = errors.New("evaluation revision conflict")
	ErrSchemeLocked  = errors.New("evaluation scheme locked")
	ErrNotAssigned   = errors.New("evaluation not assigned")
	ErrScoringClosed = errors.New("evaluation scoring closed")
	ErrLifeGone      = errors.New("evaluation life eliminated")
	ErrLifeDuplicate = errors.New("evaluation life already marked")
	ErrWrongPassword = errors.New("evaluation wrong password")
)

const (
	TypeCriteriaScoring   = "CRITERIA_SCORING"
	TypeRemoteCriteria    = "REMOTE_CRITERIA"
	TypeNumericResult     = "NUMERIC_RESULT"
	TypeQuestionScoring   = "QUESTION_SCORING"
	TypeEliminationLives  = "ELIMINATION_LIVES"
	TypeHeadToHeadVoting  = "HEAD_TO_HEAD_VOTING"
	TypeCompositeScoring  = "COMPOSITE_SCORING"
	TypeFocusGroupScoring = "FOCUS_GROUP_SCORING"

	UnitPoints     = "POINTS"
	UnitLives      = "LIVES"
	UnitVotes      = "VOTES"
	UnitPercentage = "PERCENTAGE"
	UnitSeconds    = "SECONDS"

	DefaultStartingLives = 3
	MinStartingLives     = 1
	MaxStartingLives     = 20

	ReasonWrongAnswer = "WRONG_ANSWER"
	ReasonRestore     = "RESTORE_AFTER_JURY_CORRECTION"

	AnswerYes = "YES"
	AnswerNo  = "NO"

	MinLiveQuestions = 1
	MaxLiveQuestions = 50

	RoleTrialOperator = "TRIAL_OPERATOR"
	RoleJury          = "JURY"
	RoleRemoteJury    = "REMOTE_JURY"

	CombineRankSum  = "RANK_SUM"
	CombineScoreSum = "SCORE_SUM"

	ScoreKindCriterion = "CRITERION"
	ScoreKindNumeric   = "NUMERIC"

	MinOverrideReason = 5
	MaxOverrideReason = 1000

	CorridorNone   = "NONE"
	CorridorWarn   = "WARN"
	CorridorStrict = "STRICT"

	VisibilityAdminOnly = "ADMIN_ONLY"
	VisibilityJury      = "JURY"
	VisibilityPublic    = "PUBLIC"

	EditWhileActive = "WHILE_TRIAL_ACTIVE"
	EditUntilLock   = "UNTIL_LOCK"
	EditAlways      = "ALWAYS"

	StateNotStarted  = "NOT_STARTED"
	StatePreparing   = "PREPARING"
	StateLive        = "LIVE"
	StateQuestions   = "QUESTIONS"
	StateDiscussion  = "DISCUSSION"
	StateScoring     = "SCORING"
	StatePostScoring = "POST_SCORING"
	StatePaused      = "PAUSED"
	StateApplause    = "APPLAUSE"
	StateFinished    = "FINISHED"

	PerfPlanned   = "PLANNED"
	PerfReady     = "READY"
	PerfLive      = "LIVE"
	PerfQuestions = "QUESTIONS"
	PerfScoring   = "SCORING"
	PerfFinished  = "FINISHED"
	PerfCancelled = "CANCELLED"
)

var validTypes = map[string]bool{
	TypeCriteriaScoring: true, TypeRemoteCriteria: true, TypeNumericResult: true, TypeQuestionScoring: true,
	TypeEliminationLives: true, TypeHeadToHeadVoting: true, TypeCompositeScoring: true,
	TypeFocusGroupScoring: true,
}

var validUnits = map[string]bool{
	UnitPoints: true, UnitLives: true, UnitVotes: true, UnitPercentage: true, UnitSeconds: true,
}

var validCorridors = map[string]bool{CorridorNone: true, CorridorWarn: true, CorridorStrict: true}

var validVisibility = map[string]bool{
	VisibilityAdminOnly: true, VisibilityJury: true, VisibilityPublic: true,
}

var validEditPolicy = map[string]bool{
	EditWhileActive: true, EditUntilLock: true, EditAlways: true,
}

const (
	MinNumericScore = 0
	MaxNumericScore = 10000
)

func defaultUnit(schemeType string) string {
	switch schemeType {
	case TypeEliminationLives:
		return UnitLives
	case TypeHeadToHeadVoting, TypeFocusGroupScoring:
		return UnitVotes
	default:
		return UnitPoints
	}
}

// HasLiveSession — у типа есть live-пульт и жеребьёвка.
func HasLiveSession(schemeType string) bool {
	return schemeType != TypeNumericResult && schemeType != TypeRemoteCriteria
}

// UsesCriteria — жюри ставит баллы по критериям (live или заочно).
func UsesCriteria(schemeType string) bool {
	return schemeType == TypeCriteriaScoring || schemeType == TypeRemoteCriteria
}

// ExclusiveChallengeJury — в оценках участвуют только жюри, назначенные на испытание.
func ExclusiveChallengeJury(schemeType string) bool {
	return schemeType == TypeRemoteCriteria
}
