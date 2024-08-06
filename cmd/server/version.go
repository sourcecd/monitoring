package main

import "fmt"

// build vars
var buildVersion, buildDate, buildCommit string

// check build args function
func checkBuildFlags(s string) string {
	if s != "" {
		return s
	}
	return "N/A"
}

// print build args
func printBuildFlags() {
	fmt.Printf("Build version: %s\n", checkBuildFlags(buildVersion))
	fmt.Printf("Build date: %s\n", checkBuildFlags(buildDate))
	fmt.Printf("Build commit: %s\n", checkBuildFlags(buildCommit))
}
