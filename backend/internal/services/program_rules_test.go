package services

import (
	"testing"
	"time"

	"backend/internal/models/domain"
)

func TestProgramForPosition(t *testing.T) {
	cases := map[string]string{
		"MUALIM I": "SDP", "MASINIS II": "SDP",
		"MUALIM II": "MDP", "MASINIS III": "MDP",
		"MUALIM III": "FDP", "MASINIS IV": "FDP",
		"  mualim   iii ": "FDP",
		"NAKHODA":         "", "KKM": "", "ELECTRICIAN": "", "": "",
	}
	for position, want := range cases {
		if got := ProgramForPosition(position); got != want {
			t.Errorf("ProgramForPosition(%q) = %q, want %q", position, got, want)
		}
	}
}

func TestApplyPositionEnrolsWhenThereIsNoProgram(t *testing.T) {
	for _, blank := range []string{"", "-", "UNKNOWN"} {
		next, exited := ProgramState{Program: blank}.ApplyPosition("MUALIM II")
		if exited || next.Program != "MDP" || next.Status != domain.TrainingStatusOngoing {
			t.Errorf("program %q: got %+v exited=%v", blank, next, exited)
		}
	}
	next, _ := ProgramState{}.ApplyPosition("ELECTRICIAN")
	if next.Program != "" || next.Status != "" {
		t.Errorf("position without a program must stay empty: %+v", next)
	}
}

func TestPromotionOutOfProgram(t *testing.T) {
	day := func(s string) time.Time {
		d, _ := time.Parse("2006-01-02", s)
		return d
	}
	history := []domain.MutationCache{
		{FromRankName: "MUALIM III", ToRankName: "MUALIM II", TransactionDate: day("2024-03-01")},
		{FromRankName: "MUALIM II", ToRankName: "MUALIM II", TransactionDate: day("2025-01-10")}, // vessel change only
		{FromRankName: "MUALIM II", ToRankName: "EXT. MUALIM I", TransactionDate: day("2026-05-02")},
		{FromRankName: "EXT. MUALIM I", ToRankName: "MUALIM I", TransactionDate: day("2026-08-12")},
	}

	// MDP covers MUALIM II: the exit is the move to (extra) MUALIM I, not the later rename
	from, to, at := promotionOutOfProgram(history, "MDP")
	if from != "MUALIM II" || to != "EXT. MUALIM I" || at == nil || !at.Equal(day("2026-05-02")) {
		t.Errorf("MDP: got %q -> %q at %v", from, to, at)
	}
	// FDP covers MUALIM III
	if from, to, _ := promotionOutOfProgram(history, "FDP"); from != "MUALIM III" || to != "MUALIM II" {
		t.Errorf("FDP: got %q -> %q", from, to)
	}
	// nothing in the history explains an SDP exit
	if from, to, at := promotionOutOfProgram(history, "SDP"); from != "" || to != "" || at != nil {
		t.Errorf("SDP: expected no promotion, got %q -> %q at %v", from, to, at)
	}
	if from, _, _ := promotionOutOfProgram(history, ""); from != "" {
		t.Errorf("no program: got %q", from)
	}
}

func TestExtraOfficersFollowTheirRank(t *testing.T) {
	for position, program := range map[string]string{
		"EXT. MUALIM I":    "SDP",
		"EXT. MASINIS II":  "SDP",
		"ext.  mualim iii": "FDP",
		"EXTRA NAKHODA":    "",
		"MUALIM IV":        "", // below the ladder: no program
	} {
		if got := ProgramForPosition(position); got != program {
			t.Errorf("ProgramForPosition(%q) = %q, want %q", position, got, program)
		}
	}

	// SDP participant promoted to extra master leaves the program
	if _, exited := (ProgramState{Program: "SDP", Status: domain.TrainingStatusOngoing}).ApplyPosition("EXTRA NAKHODA"); !exited {
		t.Error("SDP -> EXTRA NAKHODA should exit")
	}
	// same rank with the extra marker stays
	if next, exited := (ProgramState{Program: "SDP", Status: domain.TrainingStatusOngoing}).ApplyPosition("EXT. MUALIM I"); exited || next.Program != "SDP" {
		t.Errorf("SDP -> EXT. MUALIM I should stay, got %+v exited=%v", next, exited)
	}
}

func TestReadinessMonthsAfterIsIdempotent(t *testing.T) {
	// first run of the month at >= 85%: one month comes off
	if got := readinessMonthsAfter(12, 0, -1); got != 11 {
		t.Errorf("first reduction: got %d", got)
	}
	// the daily job runs again in the same month: nothing more comes off
	if got := readinessMonthsAfter(11, -1, -1); got != 11 {
		t.Errorf("repeat run: got %d", got)
	}
	// the rate later drops below 85% in the same month: the month is given back
	if got := readinessMonthsAfter(11, -1, 0); got != 12 {
		t.Errorf("reverted: got %d", got)
	}
	if got := readinessMonthsAfter(0, 0, -1); got != 0 {
		t.Errorf("never below zero: got %d", got)
	}
}

func TestApplyPositionExitsOnPromotion(t *testing.T) {
	// Andi, MUALIM III in FDP, is promoted to MUALIM II before finishing
	next, exited := ProgramState{Program: "FDP", Status: domain.TrainingStatusOngoing}.ApplyPosition("MUALIM II")
	if !exited || next.Program != "" || next.Status != domain.TrainingStatusExited || next.ExitedProgram != "FDP" {
		t.Fatalf("got %+v exited=%v", next, exited)
	}

	// promoted across the deck/engine equivalent and beyond the ladder
	for _, c := range []struct{ program, position string }{
		{"FDP", "MASINIS III"}, {"MDP", "MUALIM I"}, {"SDP", "NAKHODA"}, {"SDP", "KKM"}, {"FDP", "MASINIS I"},
	} {
		if _, exited := (ProgramState{Program: c.program, Status: domain.TrainingStatusOngoing}).ApplyPosition(c.position); !exited {
			t.Errorf("%s -> %s should exit", c.program, c.position)
		}
	}

	// a completed person who is then promoted also leaves the program
	if next, exited := (ProgramState{Program: "FDP", Status: domain.TrainingStatusCompleted}).ApplyPosition("MUALIM II"); !exited || next.ExitedProgram != "FDP" {
		t.Errorf("completed+promoted: %+v exited=%v", next, exited)
	}
}

func TestApplyPositionKeepsProgramOtherwise(t *testing.T) {
	keep := []struct{ program, status, position string }{
		{"FDP", domain.TrainingStatusOngoing, "MUALIM III"},   // same rank
		{"FDP", domain.TrainingStatusOngoing, "MASINIS IV"},   // equivalent rank
		{"MDP", domain.TrainingStatusCompleted, "MUALIM II"},  // completed stays completed
		{"MDP", domain.TrainingStatusOngoing, "MUALIM III"},   // lower rank: not a promotion
		{"SDP", domain.TrainingStatusOngoing, "PENDING GAJI"}, // unknown position
		{"SDP", domain.TrainingStatusOngoing, ""},             // CITRIX has no position
	}
	for _, c := range keep {
		next, exited := ProgramState{Program: c.program, Status: c.status}.ApplyPosition(c.position)
		if exited || next.Program != c.program || next.Status != c.status {
			t.Errorf("%+v: got %+v exited=%v", c, next, exited)
		}
	}

	// legacy rows have no status yet
	if next, _ := (ProgramState{Program: "SDP"}).ApplyPosition("MUALIM I"); next.Status != domain.TrainingStatusOngoing {
		t.Errorf("missing status should default to Ongoing: %+v", next)
	}
}

func TestExitedPersonIsNeverEnrolledAutomatically(t *testing.T) {
	state := ProgramState{Status: domain.TrainingStatusExited, ExitedProgram: "FDP"}
	for _, position := range []string{"MUALIM II", "MUALIM I", "MUALIM III", "NAKHODA", ""} {
		next, exited := state.ApplyPosition(position)
		if exited || next != state {
			t.Errorf("position %q changed an exited person: %+v", position, next)
		}
	}
}

func TestTrainingStatusFromProgress(t *testing.T) {
	gaps := []string{"TW", "LD"}
	cases := []struct {
		name string
		done []string
		want string
	}{
		{"nothing done", nil, domain.TrainingStatusOngoing},
		{"one material", []string{"TW-1"}, domain.TrainingStatusOngoing},
		{"one competency", []string{"TW-1; TW-2"}, domain.TrainingStatusOngoing},
		{"all, across months", []string{"TW-1; TW-2", "LD-1", " ld-2 "}, domain.TrainingStatusCompleted},
		{"extra unrelated", []string{"TW-1; TW-2; LD-1; LD-2; XX-1"}, domain.TrainingStatusCompleted},
	}
	for _, c := range cases {
		if got := TrainingStatusFromProgress(gaps, c.done); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	// no gap competencies means there is nothing to complete yet
	if got := TrainingStatusFromProgress(nil, []string{"TW-1"}); got != domain.TrainingStatusOngoing {
		t.Errorf("no gaps: got %q", got)
	}
	if got := TrainingStatusFromProgress([]string{"-", ""}, nil); got != domain.TrainingStatusOngoing {
		t.Errorf("placeholder gaps: got %q", got)
	}
}

func TestIsSchedulePostponed(t *testing.T) {
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	cases := []struct {
		scheduled, today time.Time
		want             bool
	}{
		{day(2026, 10, 8), day(2026, 10, 12), false}, // week II still running
		{day(2026, 10, 8), day(2026, 10, 15), true},  // week II over
		{day(2026, 10, 1), day(2026, 10, 8), true},
		{day(2026, 10, 22), day(2026, 10, 31), false},
		{day(2026, 12, 25), day(2027, 1, 1), true}, // week IV of December ends on 1 January
		{day(2025, 10, 1), day(2026, 10, 12), true},
		{day(2026, 11, 1), day(2026, 10, 12), false},
	}
	for _, c := range cases {
		if got := isSchedulePostponed(c.scheduled, c.today); got != c.want {
			t.Errorf("scheduled %s, today %s: got %v", c.scheduled.Format("2006-01-02"), c.today.Format("2006-01-02"), got)
		}
	}
}

func TestExitRemembersWhetherTheProgramWasFinished(t *testing.T) {
	// promoted while still training
	next, exited := ProgramState{Program: "MDP", Status: domain.TrainingStatusOngoing}.ApplyPosition("MUALIM I")
	if !exited || next.CompletedBeforeExit {
		t.Errorf("ongoing -> promoted: %+v exited=%v", next, exited)
	}
	if got := exitReason(next.Status, next.CompletedBeforeExit); got != ExitReasonPromoted {
		t.Errorf("reason = %q", got)
	}

	// finished the program first, promoted afterwards: still counts as a graduate
	next, exited = ProgramState{Program: "MDP", Status: domain.TrainingStatusCompleted}.ApplyPosition("MUALIM I")
	if !exited || !next.CompletedBeforeExit {
		t.Errorf("completed -> promoted: %+v exited=%v", next, exited)
	}
	if got := exitReason(next.Status, next.CompletedBeforeExit); got != ExitReasonCompleted {
		t.Errorf("reason = %q", got)
	}

	// finished and still at the same rank
	if got := exitReason(domain.TrainingStatusCompleted, false); got != ExitReasonCompleted {
		t.Errorf("graduate reason = %q", got)
	}
}
