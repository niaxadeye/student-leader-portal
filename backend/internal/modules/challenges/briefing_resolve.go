package challenges

import (
	"strings"
	"time"
)

func resolveBriefing(def *Briefing, override *BriefingOverride, now time.Time) ResolvedBriefing {
	if def == nil {
		def = &Briefing{Files: []BriefingFile{}}
	}
	out := ResolvedBriefing{
		BodyText:  def.BodyText,
		Files:     def.Files,
		PublishAt: def.PublishAt,
	}
	if override != nil {
		out.Personalized = override.CustomText || override.CustomPublish || override.Hidden || override.ReplaceFiles
		if override.Hidden {
			out.Hidden = true
			out.Visible = false
			out.Scheduled = false
			out.PublishAt = nil
			out.BodyText = ""
			out.Files = []BriefingFile{}
			return out
		}
		if override.CustomText {
			out.BodyText = override.BodyText
		}
		if override.CustomPublish {
			out.PublishAt = override.PublishAt
		}
		if override.ReplaceFiles {
			out.Files = override.Files
		}
	}
	if out.Files == nil {
		out.Files = []BriefingFile{}
	}
	if out.PublishAt == nil {
		return out
	}
	if !out.PublishAt.After(now) {
		out.Visible = strings.TrimSpace(out.BodyText) != "" || len(out.Files) > 0
		return out
	}
	out.Scheduled = true
	return out
}

func contestantSeesBriefing(r ResolvedBriefing) bool {
	return r.Visible || r.Scheduled
}
