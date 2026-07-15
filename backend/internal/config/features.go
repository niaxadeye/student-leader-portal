package config

// Features — feature flags (SITE.md §28). Backend — источник истины,
// frontend получает их через /api/v1/config.
type Features struct {
	ReferenceCMS       bool `json:"reference_cms"`
	EmailNotifications bool `json:"email_notifications"`
	ParticipantCabinet bool `json:"participant_cabinet"`
	Attendance         bool `json:"attendance"`
	Points             bool `json:"points"`
	Merch              bool `json:"merch"`
	Predictions        bool `json:"predictions"`
	Jury               bool `json:"jury"`
}

func loadFeatures() Features {
	return Features{
		ReferenceCMS:       envBool("FEATURE_REFERENCE_CMS", true),
		EmailNotifications: envBool("FEATURE_EMAIL_NOTIFICATIONS", false),
		ParticipantCabinet: envBool("FEATURE_PARTICIPANT_CABINET", false),
		Attendance:         envBool("FEATURE_ATTENDANCE", false),
		Points:             envBool("FEATURE_POINTS", false),
		Merch:              envBool("FEATURE_MERCH", false),
		Predictions:        envBool("FEATURE_PREDICTIONS", false),
		Jury:               envBool("FEATURE_JURY", false),
	}
}
