package app

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
)

func printHelp() {
	fmt.Println("Usage: vd <command>")
	fmt.Println("Commands:")
	fmt.Println("  vd       Open the TUI")
	fmt.Println("  add      Add a new password [--name password_name --password password]")
	fmt.Println("  get      Copy a password to clipboard")
	fmt.Println("  delete   Delete a stored password [-f|--force]")
	fmt.Println("  change   Change a stored password")
	fmt.Println("  otp      Add an OTP or copy its code [add|get otp_name]")
	fmt.Println("  register Register a new GPG key")
	fmt.Println("  ls       List stored passwords")
	fmt.Println("  gen      Generate a random password to clipboard")
	fmt.Println("  export   Export passwords to a CSV file in the home directory [password_name]")
	fmt.Println("  import   Import OTP secrets from a Google Authenticator export QR image [otp google path/to/img.jpg]")
}

func Run() int {
	if len(os.Args) < 2 {
		printHelp()
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
		if err := CopyPasswordToClipboard(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("Password for `%s` copied to clipboard\n", os.Args[2])
	case "delete":
		var name string
		force := false

		for _, arg := range os.Args[2:] {
			switch arg {
			case "-f", "--force":
				force = true
			default:
				if name != "" {
					fmt.Println("Usage: vd delete [-f|--force] password_name")
					return 1
				}
				name = arg
			}
		}

		if name == "" {
			fmt.Println("Usage: vd delete [-f|--force] password_name")
			return 1
		}

		if !slices.Contains(ReadPasswordsLookup(), name) {
			fmt.Printf("Error: password for %s doesn't exist\n", name)
			return 1
		}

		if !force && !Confirm(fmt.Sprintf("Delete password for %s?", name)) {
			return 1
		}

		if err := DeletePassword(name); err != nil {
			fmt.Println("Error deleting password:", err)
			return 1
		}

		fmt.Printf("Password for %s deleted\n", name)
	case "otp":
		if len(os.Args) < 3 {
			fmt.Println("Usage: vd otp add|get")
			return 1
		}

		switch os.Args[2] {
		case "add":
			if len(os.Args) > 3 {
				fmt.Println("Usage: vd otp add")
				return 1
			}

			var name string
			fmt.Print("Enter OTP name: ")
			fmt.Scanln(&name)

			secret, err := ReadSecret("Enter OTP secret key: ")
			if err != nil {
				return 1
			}

			if name == "" || secret == "" {
				fmt.Println("OTP name and secret key can't be empty")
				return 1
			}

			if err := SavePassword(Credentials{Name: name, Password: secret, IsOTP: true}); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}

			fmt.Printf("OTP for %s added successfully\n", name)
		case "get":
			if len(os.Args) != 4 {
				fmt.Println("Usage: vd otp get otp_name")
				return 1
			}

			if !IsOTP(os.Args[3]) {
				fmt.Printf("Error: %s is not an OTP\n", os.Args[3])
				return 1
			}

			if err := CopyPasswordToClipboard(os.Args[3]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}

			fmt.Printf("OTP code for `%s` copied to clipboard\n", os.Args[3])
		default:
			fmt.Println("Usage: vd otp add|get")
			return 1
		}
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

		fmt.Println("New generated password has been copied to clipboard")

	case "export":
		if len(os.Args) > 3 {
			fmt.Println("Usage: vd export [password_name]")
			return 1
		}

		var targets []string
		if len(os.Args) == 3 {
			if !slices.Contains(ReadPasswordsLookup(), os.Args[2]) {
				fmt.Printf("Error: password for %s doesn't exist\n", os.Args[2])
				return 1
			}
			targets = append(targets, os.Args[2])
		}

		filename, err := ExportPasswords(targets...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		fmt.Printf("Passwords exported to %s\n", filename)
	case "import":
		if len(os.Args) != 5 || os.Args[2] != "otp" {
			fmt.Println("Usage: vd import otp google path/to/img.jpg")
			return 1
		}

		if os.Args[3] != "google" {
			fmt.Printf("%s is not supported yet\n", os.Args[3])
			return 1
		}

		added, err := ImportGoogleOTP(os.Args[4])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		if len(added) == 0 {
			fmt.Println("No new OTPs were imported")
			return 1
		}

		for _, name := range added {
			fmt.Printf("OTP for %s added successfully\n", name)
		}
	case "--help":
		printHelp()
		return 0
	default:
		fmt.Println("Unknown command:", os.Args[1])
		fmt.Println()
		printHelp()
		return 1
	}

	return 0
}
