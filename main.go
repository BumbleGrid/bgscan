// Command bgscan scans a Kubernetes cluster and emits BGSpec Floor 0 JSON
// (nodes + edges) by invoking the Cobra CLI defined in the cmd package.
package main

import "github.com/BumbleGrid/bgscan/cmd"

func main() {
	cmd.Execute()
}
