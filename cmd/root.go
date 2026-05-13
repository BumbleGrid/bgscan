// Package cmd defines the bgscan CLI surface.
//
// The root command wires together configuration loading, the Kubernetes
// reader, the BGSpec mapper, and the output writer. Subcommands (when
// added) live in this same package.
package cmd

// Execute runs the root command. It is the single entry point invoked
// from main and is responsible for parsing flags, building the pipeline,
// and surfacing exit codes.
func Execute() {
	// TODO: build the cobra.Command tree (root + subcommands), parse
	// flags, and dispatch to the scan pipeline.
}
