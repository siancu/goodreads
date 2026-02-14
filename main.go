package main

import (
	"fmt"
	"os"
	"strconv"
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

	case "book":
		runBookCommand(os.Args[2:])

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
  shelf     Manage shelves
  book      Manage books`)
}

// runBookCommand dispatches book subcommands.
func runBookCommand(args []string) {
	if len(args) == 0 {
		printBookUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "search":
		if len(args) < 2 {
			fatal("usage: goodreads book search <query> [--limit N]")
		}
		query := args[1]
		limit := flagInt(args[2:], "--limit", "-n", 10)
		cmdBookSearch(query, limit)

	case "show":
		if len(args) < 2 {
			fatal("usage: goodreads book show <book-id>")
		}
		cmdBookShow(args[1])

	case "add":
		if len(args) < 3 {
			fatal("usage: goodreads book add <book-id> <shelf>")
		}
		cmdBookAdd(args[1], args[2])

	case "remove":
		if len(args) < 2 {
			fatal("usage: goodreads book remove <book-id>")
		}
		cmdBookRemove(args[1])

	case "rate":
		if len(args) < 3 {
			fatal("usage: goodreads book rate <book-id> <1-5>")
		}
		rating, err := strconv.Atoi(args[2])
		if err != nil {
			fatal("rating must be a number between 1 and 5")
		}
		cmdBookRate(args[1], rating)

	case "similar":
		if len(args) < 2 {
			fatal("usage: goodreads book similar <book-id> [--limit N] [--show-lists] [--list N]")
		}
		bookID := args[1]
		rest := args[2:]
		limit := flagInt(rest, "--limit", "-n", 10)
		showLists := hasFlag(rest, "--show-lists", "--show-lists")
		listIndex := flagInt(rest, "--list", "-l", 0)
		cmdBookSimilar(bookID, limit, showLists, listIndex)

	default:
		fmt.Fprintf(os.Stderr, "Unknown book command: %s\n\n", args[0])
		printBookUsage()
		os.Exit(1)
	}
}

// flagInt extracts an integer flag value from args, returning defaultVal if not found.
func flagInt(args []string, long, short string, defaultVal int) int {
	for i, a := range args {
		if (a == long || a == short) && i+1 < len(args) {
			v, err := strconv.Atoi(args[i+1])
			if err == nil {
				return v
			}
		}
	}
	return defaultVal
}

func printBookUsage() {
	fmt.Println(`Usage:
  goodreads book <command> [arguments]

Commands:
  search <query>           Search for books [--limit N]
  show <book-id>           Show book details
  add <book-id> <shelf>    Add a book to a shelf
  remove <book-id>         Remove a book from all shelves
  rate <book-id> <1-5>     Rate a book
  similar <book-id>        Find similar books [--limit N] [--show-lists] [--list N]`)
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
