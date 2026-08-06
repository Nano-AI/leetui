package syncer

// Phase names a stage of work, for display.
type Phase string

const (
	PhaseProblems  Phase = "problems"
	PhaseDetail    Phase = "detail"
	PhaseCompanies Phase = "companies"
)

// Progress is emitted as work proceeds.
//
// Done and Total are absolute, so a receiver can render a bar without accumulating.
type Progress struct {
	Phase Phase
	Done  int
	Total int

	// Note carries a human-readable status line, e.g. a rate-limit pause.
	Note string

	// Err is set on the final message when the job failed. A job that completes
	// successfully emits a final Progress with Done == Total and Err nil.
	Err error

	// Finished marks the last message on the channel.
	Finished bool
}

// Percent returns completion 0-100, or 0 when the total is unknown.
func (p Progress) Percent() float64 {
	if p.Total <= 0 {
		return 0
	}
	return float64(p.Done) / float64(p.Total) * 100
}
