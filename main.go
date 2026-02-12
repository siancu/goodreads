package main

import (
	"fmt"
	"os"
)

func main() {
	// Go doesn't have a built-in argparse like Python.
	// We roll our own subcommand routing using os.Args.
	// This is idiomatic for small CLIs — the standard library's "flag"
	// package only handles flat flags, not nested subcommands.
	//
	// For now we just have login/logout. As we add commands
	// (shelf, book, author, etc.), we'll add cases here.

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "login":
		// Parse login-specific flags.
		email, password, debug := parseLoginFlags(os.Args[2:])
		cmdLogin(email, password, debug)

	case "logout":
		cmdLogout()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// parseLoginFlags extracts --email/-e, --password/-p, and --debug/-d
// from the argument list. This is a hand-rolled parser because the
// standard "flag" package doesn't support short flags (-e) out of the box.
func parseLoginFlags(args []string) (email, password string, debug bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--email", "-e":
			if i+1 < len(args) {
				i++
				email = args[i]
			}
		case "--password", "-p":
			if i+1 < len(args) {
				i++
				password = args[i]
			}
		case "--debug", "-d":
			debug = true
		}
	}
	return
}

func printUsage() {
	fmt.Println(`Goodreads CLI - Access your Goodreads account

Usage:
  goodreads-go <command> [arguments]

Commands:
  login     Log in to Goodreads
  logout    Log out (remove saved cookies)`)
}
