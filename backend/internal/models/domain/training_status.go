package domain

// Values of reports.training_status: where a person stands in their IDP program.
const (
	TrainingStatusOngoing   = "Ongoing"   // still in the program
	TrainingStatusCompleted = "Completed" // finished every training of the program
	TrainingStatusExited    = "Exited"    // promoted before finishing; left the program
)

// EffectiveTrainingStatus is the status to show for a report. Rows created before the
// column existed have no stored status: they are Ongoing when they are in a program and
// have no status at all when they are not.
func EffectiveTrainingStatus(program, stored string) string {
	if stored != "" {
		return stored
	}
	switch program {
	case "FDP", "MDP", "SDP":
		return TrainingStatusOngoing
	}
	return ""
}
