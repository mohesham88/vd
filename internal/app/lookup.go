package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

func readPasswordsLookup() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return nil
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")
	lookupPath := filepath.Join(vdDir, "passwords_lookup")

	data, err := os.ReadFile(lookupPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("Error reading passwords_lookup:", err)
			return nil
		}

		if err := os.MkdirAll(vdDir, 0o700); err != nil {
			fmt.Println("Error creating vd directory:", err)
			return nil
		}

		lookup := map[string][]string{"current_passwords": {}}
		emptyData, err := json.Marshal(lookup)
		if err != nil {
			fmt.Println("Error creating lookup JSON:", err)
			return nil
		}

		encrypted, err := encrypt(emptyData)
		if err != nil {
			fmt.Println("Error encrypting lookup:", err)
			return nil
		}

		if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
			fmt.Println("Error writing passwords_lookup:", err)
			return nil
		}

		var emptyLookup map[string][]string
		if err := json.Unmarshal(emptyData, &emptyLookup); err != nil {
			fmt.Println("Error parsing passwords_lookup:", err)
			return nil
		}
		fmt.Println(emptyLookup["current_passwords"])
		return emptyLookup["current_passwords"]
	}

	decrypted, err := decrypt(data)
	if err != nil {
		fmt.Println("Error decrypting passwords_lookup:", err)
		return nil
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		fmt.Println("Error parsing passwords_lookup:", err)
		return nil
	}

	return lookup["current_passwords"]
}

func updatePasswordsLookup(newPasswordName string, toBeDeleted ...bool) {
	toBeDeletedFlag := false
	if len(toBeDeleted) > 0 {
		toBeDeletedFlag = toBeDeleted[0]
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")
	lookupPath := filepath.Join(vdDir, "passwords_lookup")

	plaintext, err := os.ReadFile(lookupPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("Error reading passwords_lookup:", err)
			return
		}

		if toBeDeletedFlag {
			return
		}

		lookup := map[string][]string{"current_passwords": {newPasswordName}}
		data, err := json.Marshal(lookup)
		if err != nil {
			fmt.Println("Error creating lookup JSON:", err)
			return
		}

		encrypted, err := encrypt(data)
		if err != nil {
			fmt.Println("Error encrypting lookup:", err)
			return
		}

		if err := os.MkdirAll(vdDir, 0o700); err != nil {
			fmt.Println("Error creating vd directory:", err)
			return
		}

		if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
			fmt.Println("Error writing passwords_lookup:", err)
			return
		}

		return
	}

	decrypted, err := decrypt(plaintext)
	if err != nil {
		fmt.Println("Error decrypting passwords_lookup:", err)
		return
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		fmt.Println("Error parsing passwords_lookup:", err)
		return
	}

	if toBeDeletedFlag {
		if !slices.Contains(lookup["current_passwords"], newPasswordName) {
			fmt.Println("Password", newPasswordName, "not found in lookup")
		}

		lookup["current_passwords"] = slices.DeleteFunc(lookup["current_passwords"], func(name string) bool {
			return name == newPasswordName
		})

	} else {
		if slices.Contains(lookup["current_passwords"], newPasswordName) {
			fmt.Println("Password", newPasswordName, "already exists")
			return
		}

		lookup["current_passwords"] = append(lookup["current_passwords"], newPasswordName)
	}

	data, err := json.Marshal(lookup)
	if err != nil {
		fmt.Println("Error encoding lookup JSON:", err)
		return
	}

	encrypted, err := encrypt(data)
	if err != nil {
		fmt.Println("Error encrypting lookup:", err)
		return
	}

	if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
		fmt.Println("Error writing passwords_lookup:", err)
		return
	}
}
