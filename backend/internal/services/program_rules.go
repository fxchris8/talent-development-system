package services

import (
	"strings"

	"backend/internal/models/domain"
)

// IDP program rules.
//
//	MUALIM III / MASINIS IV  -> FDP
//	MUALIM II  / MASINIS III -> MDP
//	MUALIM I   / MASINIS II  -> SDP
//
// A program prepares a crew member for the next rank, so once CITRIX shows a rank above the
// program's rank the person has been promoted and leaves the program ("Exited"), whether or
// not the trainings were finished. Leaving is never followed by an automatic enrolment into
// the next program; that stays a manual decision.

// normalizePosition upper-cases, collapses spaces and drops the "extra" marker CITRIX puts in
// front of a supernumerary officer ("EXT. MUALIM I", "EXTRA NAKHODA"): the rank is the same.
func normalizePosition(position string) string {
	normalized := strings.Join(strings.Fields(strings.ToUpper(position)), " ")
	for _, prefix := range []string{"EXTRA ", "EXT. ", "EXT.", "EXT "} {
		if strings.HasPrefix(normalized, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(normalized, prefix))
		}
	}
	return normalized
}

// positionLevel ranks the officer positions; 0 means "not part of the ladder / unknown".
func positionLevel(position string) int {
	switch normalizePosition(position) {
	case "MUALIM III", "MASINIS IV":
		return 1
	case "MUALIM II", "MASINIS III":
		return 2
	case "MUALIM I", "MASINIS II":
		return 3
	case "NAKHODA", "KKM", "MASINIS I":
		return 4
	}
	return 0
}

func programLevel(program string) int {
	switch strings.ToUpper(strings.TrimSpace(program)) {
	case "FDP":
		return 1
	case "MDP":
		return 2
	case "SDP":
		return 3
	}
	return 0
}

// ProgramForPosition returns the IDP program of a position, or "" when the position has none.
func ProgramForPosition(position string) string {
	switch positionLevel(position) {
	case 1:
		return "FDP"
	case 2:
		return "MDP"
	case 3:
		return "SDP"
	}
	return ""
}

// ProgramState is the part of a report the program rules read and write.
type ProgramState struct {
	Program       string // idp_program
	Status        string // training_status
	ExitedProgram string // program the person left, when Status is Exited
	// CompletedBeforeExit is set on an exit: had the person finished the program already?
	CompletedBeforeExit bool
}

// ApplyPosition returns the program state after the person's position became `position`,
// and whether the person has just left the program because of a promotion.
func (state ProgramState) ApplyPosition(position string) (next ProgramState, exited bool) {
	// Someone who left a program is only enrolled again by hand.
	if state.Status == domain.TrainingStatusExited {
		return state, false
	}

	current := programLevel(state.Program)
	if current == 0 {
		// No (valid) program yet: derive it from the position.
		program := ProgramForPosition(position)
		if program == "" {
			return ProgramState{}, false
		}
		return ProgramState{Program: program, Status: domain.TrainingStatusOngoing}, false
	}

	if positionLevel(position) > current {
		return ProgramState{
			Status:              domain.TrainingStatusExited,
			ExitedProgram:       strings.ToUpper(strings.TrimSpace(state.Program)),
			CompletedBeforeExit: state.Status == domain.TrainingStatusCompleted,
		}, true
	}

	// Same rank, an unknown position, or a lower rank: stay in the program.
	if state.Status == "" {
		state.Status = domain.TrainingStatusOngoing
	}
	return state, false
}

// TrainingStatusFromProgress decides Ongoing/Completed from the trainings a person has done.
// Every gap competency has two materials; `done` holds entries such as "TW-1; TW-2" collected
// from the monthly IDP tracking. The program is completed once every material is done.
func TrainingStatusFromProgress(gapCodes []string, done []string) string {
	finished := make(map[string]bool)
	for _, chunk := range done {
		for _, item := range strings.Split(chunk, ";") {
			if item = strings.ToUpper(strings.TrimSpace(item)); item != "" {
				finished[item] = true
			}
		}
	}

	required := 0
	for _, code := range gapCodes {
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" || code == "-" {
			continue
		}
		required++
		if !finished[code+"-1"] || !finished[code+"-2"] {
			return domain.TrainingStatusOngoing
		}
	}
	if required == 0 {
		return domain.TrainingStatusOngoing
	}
	return domain.TrainingStatusCompleted
}
