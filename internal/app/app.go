package app

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
)

func Run() int {
	if len(os.Args) < 2 {
		fmt.Println("Usage: vd <command>")
		fmt.Println("Commands:")
		fmt.Println("  add      Add a new password [--name password_name --password password]")
		fmt.Println("  get      Copy a password to clipboard")
		fmt.Println("  delete   Delete a stored password")
		fmt.Println("  change   Change a stored password")
		fmt.Println("  register Register a new GPG key")
		fmt.Println("  ls       List stored passwords")
		fmt.Println("  gen      Generate a random password to clipboard")
		return 0
	}

	switch os.Args[1] {
	case "add":
		fs := flag.NewFlagSet("add", flag.ContinueOnError)
		name := fs.String("name", "", "password name")
		password := fs.String("password", "", "password value")

		if err := fs.Parse(os.Args[2:]); err != nil {
			return 1
		}

		if (*name == "") != (*password == "") || fs.NArg() > 0 {
			fmt.Println("Usage: vd add [--name password_name --password password]")
			return 1
		}

		obj := &Credentials{Name: *name, Password: *password}

		if *name == "" {
			obj = ReadCredential()
			if obj == nil {
				return 1
			}
		}

		if err := SavePassword(*obj); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		fmt.Printf("Password for %s added successfully\n", obj.Name)
	case "get":
		if len(os.Args) < 3 || len(os.Args) > 3 {
			fmt.Println("Usage: vd get password_name")
			return 1
		}
		if err := GetPassword(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("Password for `%s` copied to clipboard\n", os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: vd delete password_name")
			return 1
		}
		if !slices.Contains(ReadPasswordsLookup(), os.Args[2]) {
			fmt.Printf("Error: password for %s doesn't exist\n", os.Args[2])
			return 1
		}

		if !Confirm(fmt.Sprintf("Delete password for %s?", os.Args[2])) {
			return 1
		}

		success, err := DeletePassword(os.Args[2])
		if err != nil {
			fmt.Println("Error deleting password:", err)
			return 1
		}

		if !success {
			fmt.Printf("Error: password for %s doesn't exist\n", os.Args[2])
			return 1
		}

		fmt.Printf("Password for %s deleted\n", os.Args[2])
	case "register":
		register()
	case "ls":
		listCurrentPasswords()
	case "change":
		if len(os.Args) < 3 || len(os.Args) > 3 {
			fmt.Println("Usage: vd change password_name")
			return 1
		}
		if err := changePassword(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		fmt.Printf("Password for `%s` changed successfully\n", os.Args[2])
	case "gen":
		newPassword, err := GenerateRandomPassword()
		if err != nil {
			log.Println("Error: ", err)
			return 1
		}

		err = CopyToClipboard(newPassword)
		if err != nil {
			log.Println("Error: ", err)
			return 1
		}

		fmt.Println("New random password has been copied to clipboard")

	default:
		fmt.Println("Unknown command:", os.Args[1])
		return 1
	}

	return 0
}
