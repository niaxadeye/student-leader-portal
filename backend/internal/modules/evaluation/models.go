package evaluation

import "time"

type Actor struct {
	UserID  string
	IsSuper bool
	IsMega  bool
}

type Scheme struct {
	ID               string
	ChallengeID      string
	ContestID        string
	Name             string
	Type             string
	ScoringUnit      string
	MinScore         *float64
	MaxScore         *float64
	CorridorMode     string
	ResultVisibility string
	EditPolicy       string
	SettingsJSON     []byte
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Criteria         []Criterion
	OperatorUserID   *string
}

type Criterion struct {
	ID          string
	SchemeID    string
	GroupID     *string
	Title       string
	Description *string
	MinScore    float64
	MaxScore    float64
	Weight      float64
	IsRequired  bool
	SortOrder   int
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Bands       []ScaleBand
}

type ScaleBand struct {
	ID          string
	CriterionID string
	MinScore    float64
	MaxScore    float64
	Description string
	SortOrder   int
}

type SchemeInput struct {
	Name             string
	Type             string
	ScoringUnit      string
	MinScore         *float64
	MaxScore         *float64
	CorridorMode     string
	ResultVisibility string
	EditPolicy       string
	StartingLives    *int
	OperatorUserID   *string
	SettingsJSON     []byte
}

type CriterionInput struct {
	Title       string
	Description *string
	MinScore    float64
	MaxScore    float64
	Weight      float64
	IsRequired  *bool
	Bands       []ScaleBandInput
}

type ScaleBandInput struct {
	MinScore    float64
	MaxScore    float64
	Description string
}

type JuryContest struct {
	ID         string
	Name       string
	Slug       string
	Challenges []JuryChallenge
}

type JuryChallenge struct {
	ID         string
	Title      string
	Slug       string
	Status     string
	HasScheme  bool
	SchemeType string
}

type Session struct {
	ID                      string
	ChallengeID             string
	CurrentPerformanceID    *string
	CurrentContestantUserID *string
	CurrentMatchID          *string
	State                   string
	CurrentPhaseID          *string
	StartedAt               *time.Time
	StateChangedAt          *time.Time
	FinishedAt              *time.Time
	ControlledBy            *string
	Revision                int
	PhaseStartedAt          *time.Time
	PhaseDurationSeconds    *int
	PausedAt                *time.Time
	AccumulatedPauseSeconds float64
	CurrentQuestionNumber   int
	QuestionCount           int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type Performance struct {
	ID               string
	ChallengeID      string
	ContestantUserID string
	SequenceNumber   *int
	Status           string
	StartedAt        *time.Time
	FinishedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type PhaseTemplate struct {
	ID              string
	SchemeID        string
	Title           string
	DurationSeconds *int
	ScoringAllowed  bool
	MapsToState     string
	SortOrder       int
}

type LiveContestant struct {
	UserID                string
	Login                 string
	FullName              string
	Organization          *string
	PerformanceID         *string
	PerformanceStatus     *string
	DrawNumber            *int
	SpeechDurationSeconds *float64
	AvatarKey             *string
	AvatarURL             *string
}

type LiveSnapshot struct {
	ChallengeID           string
	ContestID             string
	ChallengeTitle        string
	Session               Session
	Performance           *Performance
	Current               *LiveContestant
	Phases                []PhaseTemplate
	Contestants           []LiveContestant
	TimerRemaining        *float64
	JuryOnline            int
	ServerTime            time.Time
	Drawn                 bool
	SchemeType            string
	StartingLives         *int
	CurrentQuestionNumber int
	OperatorUserID        *string
	Lives                 *LivesBoard
	CorrectAnswer         *string
	ShowAnswerKey         bool
	QuestionCount         int
	QuestionKeys          []QuestionKey
}

type ContestantDrawEntry struct {
	DrawNumber   int
	FullName     string
	Organization *string
	IsMe         bool
}

type ContestantDraw struct {
	Drawn        bool
	MyDrawNumber *int
	Total        int
	Order        []ContestantDrawEntry
}

type MyDrawSummary struct {
	ChallengeID  string
	MyDrawNumber *int
	Total        int
}

type ScoreValue struct {
	ID             string
	ScoreSheetID   string
	CriterionID    string
	Score          float64
	Comment        *string
	Revision       int
	LastMutationID *string
}

type ScoreMutation struct {
	PerformanceID string
	CriterionID   string
	Score         float64
	MutationID    string
	BaseRevision  *int
}

// ScoreRevisionConflict carries the current server value so a local-first
// client can rebase an unacknowledged mutation without silently dropping it.
type ScoreRevisionConflict struct {
	Score    *float64
	Revision int
}

func (e *ScoreRevisionConflict) Error() string { return ErrRevision.Error() }
func (e *ScoreRevisionConflict) Unwrap() error { return ErrRevision }

type ScorecardCriterion struct {
	Criterion
	Score    *float64
	Revision int
}

type Scorecard struct {
	Configured    bool
	SchemeType    string
	ScoringUI     string
	Editable      bool
	PerformanceID *string
	Contestant    *LiveContestant
	Criteria      []ScorecardCriterion
	Filled        int
	Total         *float64
}

type ScoreWriteResult struct {
	CriterionID string
	Score       float64
	Revision    int
	Total       float64
}

type JuryPerson struct {
	UserID   string
	Login    string
	FullName string
}

type ScoreboardValue struct {
	CriterionID string
	Score       *float64
}

type ScoreboardSheet struct {
	JuryUserID string
	Filled     int
	Total      *float64
	Values     []ScoreboardValue
}

type ScoreboardContestant struct {
	LiveContestant
	Sheets             []ScoreboardSheet
	Average            *float64
	Sum                *float64
	Rank               *int
	Lives              *int
	Eliminated         bool
	EliminatedQuestion *int
	NumericScore       *float64
}

type Scoreboard struct {
	Configured              bool
	SchemeType              string
	ScoringUI               string
	CurrentContestantUserID *string
	Criteria                []Criterion
	Jury                    []JuryPerson
	Contestants             []ScoreboardContestant
	StartingLives           *int
	OperatorUserID          *string
	CurrentQuestionNumber   int
	QuestionCount           int
	LifeLogs                []JuryLifeLog
	MinScore                *float64
	MaxScore                *float64
	Combined                *CombinedRanking
	CanOverride             bool
	Corrections             []ScoreCorrection
}

type ScoreCorrection struct {
	ID               string
	Kind             string
	ActorUserID      string
	ActorName        string
	ContestantUserID string
	ContestantName   string
	JuryUserID       *string
	JuryName         *string
	CriterionID      *string
	CriterionTitle   string
	OldScore         *float64
	NewScore         *float64
	Reason           string
	CreatedAt        time.Time
}

type ScoreOverrideInput struct {
	Kind             string
	ContestantUserID string
	JuryUserID       string
	CriterionID      string
	Score            *float64
	Reason           string
}

type StageLink struct {
	ID                string
	ContestID         string
	MainChallengeID   string
	MainTitle         string
	RemoteChallengeID string
	RemoteTitle       string
	MainWeight        float64
	RemoteWeight      float64
	CombineMode       string
}

type ChallengeOption struct {
	ID    string
	Title string
}

type StageLinkView struct {
	Link          *StageLink
	LinkedFrom    *ChallengeOption
	RemoteOptions []ChallengeOption
}

type CombinedRow struct {
	UserID      string
	FullName    string
	MainScore   *float64
	MainRank    *int
	RemoteScore *float64
	RemoteRank  *int
	Combined    *float64
	Rank        *int
}

type CombinedRanking struct {
	RemoteChallengeID    string
	RemoteChallengeTitle string
	MainWeight           float64
	RemoteWeight         float64
	CombineMode          string
	Rows                 []CombinedRow
}

type StageLinkInput struct {
	RemoteChallengeID *string
	MainWeight        float64
	RemoteWeight      float64
	CombineMode       string
}

type LifeEvent struct {
	ID                  string
	ChallengeID         string
	ContestantUserID    string
	QuestionNumber      int
	Delta               int
	Reason              string
	CreatedByUserID     string
	ReversesLifeEventID *string
	CreatedAt           time.Time
}

type QuestionLoss struct {
	ContestantUserID string
	FullName         string
	Organization     *string
	AvatarURL        *string
}

type QuestionAnswerMark struct {
	ContestantUserID string
	FullName         string
	Answer           string
	Mismatch         bool
}

type QuestionLogEntry struct {
	QuestionNumber int
	Current        bool
	CorrectAnswer  *string
	Losses         []QuestionLoss
	Answers        []QuestionAnswerMark
}

type LivesRow struct {
	UserID             string
	Lives              int
	Eliminated         bool
	EliminatedQuestion *int
	Rank               *int
	RestoreQuestions   []int
	Answer             *string
	Mismatch           bool
	CanReveal          bool
}

type LivesBoard struct {
	StartingLives   int
	CurrentQuestion int
	OperatorUserID  *string
	ViewerUserID    *string
	Official        bool
	CorrectAnswer   *string
	Questions       []QuestionLogEntry
	Rows            []LivesRow
}

type AnswerMark struct {
	ContestantUserID string
	JuryUserID       string
	QuestionNumber   int
	Answer           string
}

type QuestionKey struct {
	QuestionNumber int
	CorrectAnswer  string
}

type JuryLifeLog struct {
	JuryUserID string
	Official   bool
	Questions  []QuestionLogEntry
	Rows       []LivesRow
}

type contestantLifeState struct {
	ID                 string
	Lives              int
	EliminatedQuestion *int
	Draw               int
}

type challengeScoreRow struct {
	ContestantUserID string
	JuryUserID       string
	CriterionID      string
	Score            float64
}
