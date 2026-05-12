package utils

// CommonFlags holds flags shared across most phase subcommands.
// Populated by persistent flags on the parent and read by each subcommand.
type CommonFlags struct {
	// language   string
	Packs      []string
	NumThreads int
	CodeQLArgs string
}
