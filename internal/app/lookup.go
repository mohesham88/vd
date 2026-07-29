package app

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"slices"
)

func ReadPasswordsLookup() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		PrintErr("Error getting home directory:", err)
		return nil
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")
	lookupPath := filepath.Join(vdDir, "passwords_lookup")

	data, err := os.ReadFile(lookupPath)
	if err != nil {
		if !os.IsNotExist(err) {
			PrintErr("Error reading passwords_lookup:", err)
			return nil
		}

		if err := os.MkdirAll(vdDir, 0o700); err != nil {
			PrintErr("Error creating vd directory:", err)
			return nil
		}

		lookup := map[string][]string{"current_passwords": {}}
		emptyData, err := json.Marshal(lookup)
		if err != nil {
			PrintErr("Error creating lookup JSON:", err)
			return nil
		}

		encrypted, err := Encrypt(emptyData)
		if err != nil {
			PrintErr("Error encrypting lookup:", err)
			return nil
		}

		if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
			PrintErr("Error writing passwords_lookup:", err)
			return nil
		}

		var emptyLookup map[string][]string
		if err := json.Unmarshal(emptyData, &emptyLookup); err != nil {
			PrintErr("Error parsing passwords_lookup:", err)
			return nil
		}
		PrintErr(emptyLookup["current_passwords"])
		return emptyLookup["current_passwords"]
	}

	decrypted, err := Decrypt(data)
	if err != nil {
		PrintErr("Error decrypting passwords_lookup:", err)
		return nil
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		PrintErr("Error parsing passwords_lookup:", err)
		return nil
	}

	return lookup["current_passwords"]
}

func ReadOTPLookup() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		PrintErr("Error getting home directory:", err)
		return nil
	}

	data, err := os.ReadFile(filepath.Join(home, ".local", "share", "vd", "passwords_lookup"))
	if err != nil {
		return nil
	}

	decrypted, err := Decrypt(data)
	if err != nil {
		PrintErr("Error decrypting passwords_lookup:", err)
		return nil
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		PrintErr("Error parsing passwords_lookup:", err)
		return nil
	}

	return lookup["otp_passwords"]
}

func IsOTP(name string) bool {
	return slices.Contains(ReadOTPLookup(), name)
}

func UpdateOTPLookup(name string, toBeDeleted ...bool) {
	toBeDeletedFlag := false
	if len(toBeDeleted) > 0 {
		toBeDeletedFlag = toBeDeleted[0]
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return
	}

	lookupPath := filepath.Join(home, ".local", "share", "vd", "passwords_lookup")

	data, err := os.ReadFile(lookupPath)
	if err != nil {
		log.Println("Error reading passwords_lookup:", err)
		return
	}

	decrypted, err := Decrypt(data)
	if err != nil {
		log.Println("Error decrypting passwords_lookup:", err)
		return
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		log.Println("Error parsing passwords_lookup:", err)
		return
	}

	if toBeDeletedFlag {
		lookup["otp_passwords"] = slices.DeleteFunc(lookup["otp_passwords"], func(n string) bool {
			return n == name
		})
	} else {
		if slices.Contains(lookup["otp_passwords"], name) {
			return
		}
		lookup["otp_passwords"] = append(lookup["otp_passwords"], name)
	}

	updated, err := json.Marshal(lookup)
	if err != nil {
		log.Println("Error encoding lookup JSON:", err)
		return
	}

	encrypted, err := Encrypt(updated)
	if err != nil {
		log.Println("Error encrypting lookup:", err)
		return
	}

	if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
		log.Println("Error writing passwords_lookup:", err)
	}
}

func UpdatePasswordsLookup(newPasswordName string, toBeDeleted ...bool) {
	toBeDeletedFlag := false
	if len(toBeDeleted) > 0 {
		toBeDeletedFlag = toBeDeleted[0]
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting home directory:", err)
		return
	}

	vdDir := filepath.Join(home, ".local", "share", "vd")
	lookupPath := filepath.Join(vdDir, "passwords_lookup")

	plaintext, err := os.ReadFile(lookupPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println("Error reading passwords_lookup:", err)
			return
		}

		if toBeDeletedFlag {
			return
		}

		lookup := map[string][]string{"current_passwords": {newPasswordName}}
		data, err := json.Marshal(lookup)
		if err != nil {
			log.Println("Error creating lookup JSON:", err)
			return
		}

		encrypted, err := Encrypt(data)
		if err != nil {
			log.Println("Error encrypting lookup:", err)
			return
		}

		if err := os.MkdirAll(vdDir, 0o700); err != nil {
			log.Println("Error creating vd directory:", err)
			return
		}

		if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
			log.Println("Error writing passwords_lookup:", err)
			return
		}

		return
	}

	decrypted, err := Decrypt(plaintext)
	if err != nil {
		log.Println("Error decrypting passwords_lookup:", err)
		return
	}

	var lookup map[string][]string
	if err := json.Unmarshal(decrypted, &lookup); err != nil {
		log.Println("Error parsing passwords_lookup:", err)
		return
	}

	if toBeDeletedFlag {
		if !slices.Contains(lookup["current_passwords"], newPasswordName) {
			log.Printf("Password %s not found in lookup", newPasswordName)
		}

		lookup["current_passwords"] = slices.DeleteFunc(lookup["current_passwords"], func(name string) bool {
			return name == newPasswordName
		})

	} else {
		if slices.Contains(lookup["current_passwords"], newPasswordName) {
			log.Printf("Password %s already exists", newPasswordName)
			return
		}

		lookup["current_passwords"] = append(lookup["current_passwords"], newPasswordName)
	}

	data, err := json.Marshal(lookup)
	if err != nil {
		log.Println("Error encoding lookup JSON:", err)
		return
	}

	encrypted, err := Encrypt(data)
	if err != nil {
		log.Println("Error encrypting lookup:", err)
		return
	}

	if err := os.WriteFile(lookupPath, encrypted, 0o600); err != nil {
		log.Println("Error writing passwords_lookup:", err)
		return
	}
}
