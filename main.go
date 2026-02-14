package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "login":
		email, password, debug := parseLoginFlags(os.Args[2:])
		cmdLogin(email, password, debug)

	case "logout":
		cmdLogout()

	case "shelf":
		runShelfCommand(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// runShelfCommand dispatches shelf subcommands: list, show, add, delete.
func runShelfCommand(args []string) {
	if len(args) == 0 {
		printShelfUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		cmdShelfList()

	case "show":
		if len(args) < 2 {
			fatal("usage: goodreads shelf show <shelf-name>")
		}
		cmdShelfShow(args[1])

	case "add":
		if len(args) < 2 {
			fatal("usage: goodreads shelf add <shelf-name> [--debug]")
		}
		name, debug := args[1], hasFlag(args[2:], "--debug", "-d")
		cmdShelfAdd(name, debug)

	case "delete":
		if len(args) < 2 {
			fatal("usage: goodreads shelf delete <shelf-name> [--force] [--debug]")
		}
		name := args[1]
		rest := args[2:]
		force := hasFlag(rest, "--force", "-f")
		debug := hasFlag(rest, "--debug", "-d")
		cmdShelfDelete(name, force, debug)

	default:
		fmt.Fprintf(os.Stderr, "Unknown shelf command: %s\n\n", args[0])
		printShelfUsage()
		os.Exit(1)
	}
}

// parseLoginFlags extracts --email/-e, --password/-p, and --debug/-d.
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

// hasFlag checks whether a flag (long or short form) appears in args.
func hasFlag(args []string, long, short string) bool {
	for _, a := range args {
		if a == long || a == short {
			return true
		}
	}
	return false
}

func printUsage() {
	fmt.Println(`Goodreads CLI - Access your Goodreads account

Usage:
  goodreads <command> [arguments]

Commands:
  login     Log in to Goodreads
  logout    Log out (remove saved cookies)
  shelf     Manage shelves`)
}

func printShelfUsage() {
	fmt.Println(`Usage:
  goodreads shelf <command> [arguments]

Commands:
  list                List all shelves
  show <shelf-name>   Show books on a shelf
  add <shelf-name>    Create a new shelf
  delete <shelf-name> Delete a shelf`)
}
