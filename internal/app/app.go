package app

import (
	"fmt"
	"os"
)

func Run() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: vd <command>")
		fmt.Println("Commands:")
		fmt.Println("  add      Add a new password")
		fmt.Println("  get      Copy a password to clipboard")
		fmt.Println("  delete   Delete a stored password")
		fmt.Println("  register Register a new GPG key")
		fmt.Println("  ls       List stored passwords")
		fmt.Println("  gen      Generate a random password to clipboard")
		return
	}

	switch os.Args[1] {
	case "add":
		obj := readCredential()
		if obj == nil {
			return
		}

		success, err := savePassword(obj)
		if err != nil {
			return
		}

		if success {
			fmt.Println("Password for", obj["Name"], "added successfully")
		}
	case "get":
		if len(os.Args) < 3 || len(os.Args) > 3 {
			fmt.Println("Usage: vd get password_name")
			return
		}
		getPassword(os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: vd delete password_name")
			return
		}
		deletePassword(os.Args[2])
	case "register":
		register()
	case "ls":
		listCurrentPasswords()
	case "gen":
		newPassword, err := generateRandomPassword()
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		err = copyToClipboard(newPassword)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		fmt.Println("New random password has been copied to clipboard")

	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}
