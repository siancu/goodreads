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

	case "author":
		runAuthorCommand(os.Args[2:])

	case "user":
		runUserCommand(os.Args[2:])

	case "stats":
		yearFilter := ""
		if len(os.Args) >= 3 {
			yearFilter = os.Args[2]
		}
		cmdStats(yearFilter)

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
  book      Manage books
  author    Author information
  user      View user information
  stats     Show your reading statistics`)
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
			fatal("usage: goodreads book remove <book-id> [shelf]")
		}
		shelfName := ""
		if len(args) >= 3 {
			shelfName = args[2]
		}
		cmdBookRemove(args[1], shelfName)

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

	case "reviews":
		if len(args) < 2 {
			fatal("usage: goodreads book reviews <book-id> [--best N] [--worst N] [--limit N] [--full] [--review N]")
		}
		bookID := args[1]
		rest := args[2:]
		bestN := flagInt(rest, "--best", "-b", 0)
		worstN := flagInt(rest, "--worst", "-w", 0)
		limit := flagInt(rest, "--limit", "-n", 5)
		full := hasFlag(rest, "--full", "-f")
		reviewIndex := flagInt(rest, "--review", "-r", 0)
		cmdBookReviews(bookID, bestN, worstN, limit, full, reviewIndex)

	case "status":
		if len(args) < 3 {
			fatal("usage: goodreads book status <book-id> <reading|read|to-read>")
		}
		cmdBookStatus(args[1], args[2])

	case "progress":
		if len(args) < 2 {
			fatal("usage: goodreads book progress <book-id> --page N | --percent N [--comment TEXT]")
		}
		bookID := args[1]
		rest := args[2:]
		page := flagInt(rest, "--page", "-p", 0)
		pct := flagInt(rest, "--percent", "--percent", 0)
		comment := flagString(rest, "--comment", "-c", "")
		cmdBookProgress(bookID, page, pct, comment)

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
  remove <book-id> [shelf] Remove a book from a shelf (or all shelves)
  rate <book-id> <1-5>     Rate a book
  similar <book-id>        Find similar books [--limit N] [--show-lists] [--list N]
  reviews <book-id>        Show book reviews [--best N] [--worst N] [--limit N] [--full] [--review N]
  status <book-id> <status> Set reading status (reading, read, to-read)
  progress <book-id>        Update reading progress (--page N or --percent N) [--comment TEXT]`)
}

// runAuthorCommand dispatches author subcommands.
func runAuthorCommand(args []string) {
	if len(args) == 0 {
		printAuthorUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "search":
		if len(args) < 2 {
			fatal("usage: goodreads author search <query> [--limit N]")
		}
		query := args[1]
		limit := flagInt(args[2:], "--limit", "-n", 10)
		cmdAuthorSearch(query, limit)

	case "show":
		if len(args) < 2 {
			fatal("usage: goodreads author show <author-id>")
		}
		cmdAuthorShow(args[1])

	case "books":
		if len(args) < 2 {
			fatal("usage: goodreads author books <author-id> [--limit N]")
		}
		limit := flagInt(args[2:], "--limit", "-n", 20)
		cmdAuthorBooks(args[1], limit)

	default:
		fmt.Fprintf(os.Stderr, "Unknown author command: %s\n\n", args[0])
		printAuthorUsage()
		os.Exit(1)
	}
}

func printAuthorUsage() {
	fmt.Println(`Usage:
  goodreads author <command> [arguments]

Commands:
  search <query>         Search for authors [--limit N]
  show <author-id>       Show author bio
  books <author-id>      List books by author [--limit N]`)
}

// runUserCommand dispatches user subcommands.
func runUserCommand(args []string) {
	if len(args) == 0 {
		printUserUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		cmdUserList()

	case "show":
		if len(args) < 2 {
			fatal("usage: goodreads user show <user-id>")
		}
		cmdUserShow(args[1])

	case "shelves":
		if len(args) < 2 {
			fatal("usage: goodreads user shelves <user-id>")
		}
		cmdUserShelves(args[1])

	case "books":
		if len(args) < 2 {
			fatal("usage: goodreads user books <user-id> [--shelf NAME] [--limit N]")
		}
		uid := args[1]
		rest := args[2:]
		shelf := flagString(rest, "--shelf", "-s", "read")
		limit := flagInt(rest, "--limit", "-n", 0)
		cmdUserBooks(uid, shelf, limit)

	case "stats":
		if len(args) < 2 {
			fatal("usage: goodreads user stats <user-id>")
		}
		cmdUserStats(args[1])

	default:
		fmt.Fprintf(os.Stderr, "Unknown user command: %s\n\n", args[0])
		printUserUsage()
		os.Exit(1)
	}
}

// flagString extracts a string flag value from args, returning defaultVal if not found.
func flagString(args []string, long, short, defaultVal string) string {
	for i, a := range args {
		if (a == long || a == short) && i+1 < len(args) {
			return args[i+1]
		}
	}
	return defaultVal
}

func printUserUsage() {
	fmt.Println(`Usage:
  goodreads user <command> [arguments]

Commands:
  list                     List users you follow
  show <user-id>           Show user profile
  shelves <user-id>        List user's shelves
  books <user-id>          Show user's books [--shelf NAME] [--limit N]
  stats <user-id>          Show user's reading stats`)
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
