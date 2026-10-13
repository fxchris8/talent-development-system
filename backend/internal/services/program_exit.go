package services

import (
	"sort"
	"strings"
	"time"

	"backend/internal/models/domain"
	"backend/internal/models/web"
)

// Why someone is no longer an active participant of an IDP program.
const (
	ExitReasonCompleted = "completed" // finished every training of the program
	ExitReasonPromoted  = "promoted"  // promoted before finishing
)

// promotionOutOfProgram finds, in a person's CITRIX mutation history, the move that took them
// out of an IDP program: the latest mutation from a rank the program covers (or a lower one) to
// a rank above it. For MDP that is e.g. "MUALIM II -> EXT. MUALIM I", even when a later mutation
// only changed "EXT. MUALIM I" into "MUALIM I".
func promotionOutOfProgram(mutations []domain.MutationCache, program string) (from, to string, at *time.Time) {
	level := programLevel(program)
	if level == 0 {
		return "", "", nil
	}

	sorted := make([]domain.MutationCache, len(mutations))
	copy(sorted, mutations)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].TransactionDate.After(sorted[j].TransactionDate)
	})

	for _, m := range sorted {
		fromLevel := positionLevel(m.FromRankName)
		if fromLevel == 0 || fromLevel > level || positionLevel(m.ToRankName) <= level {
			continue
		}
		var date *time.Time
		if !m.TransactionDate.IsZero() {
			d := m.TransactionDate
			date = &d
		}
		return strings.TrimSpace(m.FromRankName), strings.TrimSpace(m.ToRankName), date
	}
	return "", "", nil
}

// exitReason says why a report left the list of active participants: a graduate finished the
// program (whether or not a promotion followed), anyone else was promoted before finishing.
func exitReason(status string, completedBeforeExit bool) string {
	if status == domain.TrainingStatusCompleted || completedBeforeExit {
		return ExitReasonCompleted
	}
	return ExitReasonPromoted
}

// ProgramExits lists everyone who is no longer an active participant of an IDP program, newest
// first: graduates (status Completed) and people promoted out of it (status Exited). The
// promotion comes from the CITRIX mutation history.
func (service *ReportService) ProgramExits() ([]web.ProgramExit, error) {
	var reports []domain.Report
	if err := service.DB.
		Where("training_status IN ?", []string{domain.TrainingStatusExited, domain.TrainingStatusCompleted}).
		Find(&reports).Error; err != nil {
		return nil, err
	}

	exits := make([]web.ProgramExit, 0, len(reports))
	for _, report := range reports {
		promoted := report.TrainingStatus == domain.TrainingStatusExited
		program := report.ExitedProgram
		if !promoted {
			program = report.IDPProgram // a graduate who is still at the program's rank
		}

		exit := web.ProgramExit{
			ReportID:        report.ID,
			SeafarerCode:    report.SeafarerCode,
			Nama:            report.Nama,
			ExitedProgram:   program,
			CurrentPosition: report.Jabatan,
			VesselName:      report.VesselName,
			ExitedAt:        report.ProgramExitedAt,
			Reason:          exitReason(report.TrainingStatus, report.CompletedBeforeExit),
			Promoted:        promoted,
			CompletedAt:     report.ProgramCompletedAt,
		}

		if promoted {
			mutations, err := service.MutationCacheRepo.GetAllBySeamanCode(strings.TrimSpace(report.SeamanCode))
			if err != nil {
				service.Log.Warnf("Failed to load mutation history for %s: %v", report.SeamanCode, err)
			}
			exit.FromPosition, exit.ToPosition, exit.PromotedAt = promotionOutOfProgram(mutations, program)
		}

		exits = append(exits, exit)
	}

	// Newest event first: the promotion, else when the program was finished, else when the
	// system recorded the exit.
	when := func(e web.ProgramExit) time.Time {
		for _, t := range []*time.Time{e.PromotedAt, e.CompletedAt, e.ExitedAt} {
			if t != nil {
				return *t
			}
		}
		return time.Time{}
	}
	sort.SliceStable(exits, func(i, j int) bool { return when(exits[i]).After(when(exits[j])) })

	return exits, nil
}
