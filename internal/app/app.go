package app

import (
	"log"
	"os"
)

func Run() {
	if len(os.Args) < 2 {
		log.Println("Usage: vd <command>")
		log.Println("Commands:")
		log.Println("  add      Add a new password")
		log.Println("  get      Copy a password to clipboard")
		log.Println("  delete   Delete a stored password")
		log.Println("  change   Change a stored password")
		log.Println("  register Register a new GPG key")
		log.Println("  ls       List stored passwords")
		log.Println("  gen      Generate a random password to clipboard")
		return
	}

	switch os.Args[1] {
	case "add":
		obj := ReadCredential()
		if obj == nil {
			return
		}

		success, err := SavePassword(*obj)
		if err != nil {
			return
		}

		if success {
			log.Printf("Password for %s added successfully", obj.Name)
		}
	case "get":
		if len(os.Args) < 3 || len(os.Args) > 3 {
			log.Println("Usage: vd get password_name")
			return
		}
		GetPassword(os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			log.Println("Usage: vd delete password_name")
			return
		}
		deletePassword(os.Args[2])
	case "register":
		register()
	case "ls":
		listCurrentPasswords()
	case "change":
		if len(os.Args) < 3 || len(os.Args) > 3 {
			log.Println("Usage: vd change password_name")
			return
		}
		changePassword(os.Args[2])
	case "gen":
		newPassword, err := GenerateRandomPassword()
		if err != nil {
			log.Println("Error: ", err)
			return
		}

		err = CopyToClipboard(newPassword)
		if err != nil {
			log.Println("Error: ", err)
			return
		}

		log.Println("New random password has been copied to clipboard")

	default:
		log.Println("Unknown command:", os.Args[1])
	}
}
