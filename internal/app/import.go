package app

import (
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

type otpAccount struct {
	Issuer  string
	Account string
	Secret  string
}

func readVarint(buf []byte, i int) (uint64, int, error) {
	var result uint64
	var shift uint
	for {
		if i >= len(buf) {
			return 0, 0, fmt.Errorf("error: truncated varint")
		}
		b := buf[i]
		i++
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return result, i, nil
		}
		shift += 7
	}
}

type protoField struct {
	number int
	bytes  []byte
}

func readFields(buf []byte) ([]protoField, error) {
	var fields []protoField
	i := 0
	for i < len(buf) {
		key, next, err := readVarint(buf, i)
		if err != nil {
			return nil, err
		}
		i = next

		number, wire := int(key>>3), key&7
		switch wire {
		case 0:
			_, i, err = readVarint(buf, i)
			if err != nil {
				return nil, err
			}
			fields = append(fields, protoField{number: number})
		case 2:
			length, next, err := readVarint(buf, i)
			if err != nil {
				return nil, err
			}
			i = next
			if i+int(length) > len(buf) {
				return nil, fmt.Errorf("error: truncated field")
			}
			fields = append(fields, protoField{number: number, bytes: buf[i : i+int(length)]})
			i += int(length)
		case 5:
			if i+4 > len(buf) {
				return nil, fmt.Errorf("error: truncated field")
			}
			fields = append(fields, protoField{number: number, bytes: buf[i : i+4]})
			i += 4
		case 1:
			if i+8 > len(buf) {
				return nil, fmt.Errorf("error: truncated field")
			}
			fields = append(fields, protoField{number: number, bytes: buf[i : i+8]})
			i += 8
		default:
			return nil, fmt.Errorf("error: unsupported wire type %d", wire)
		}
	}
	return fields, nil
}

func decodeMigrationPayload(migrationURL string) ([]otpAccount, error) {
	parsed, err := url.Parse(migrationURL)
	if err != nil {
		return nil, fmt.Errorf("error: invalid migration URL: %w", err)
	}

	data := parsed.Query().Get("data")
	if data == "" {
		return nil, fmt.Errorf("error: not an otpauth-migration URL (no data parameter)")
	}

	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("error: decoding migration data: %w", err)
	}

	fields, err := readFields(raw)
	if err != nil {
		return nil, err
	}

	var accounts []otpAccount
	for _, field := range fields {
		if field.number != 1 {
			continue
		}

		params, err := readFields(field.bytes)
		if err != nil {
			return nil, err
		}

		var acct otpAccount
		for _, param := range params {
			switch param.number {
			case 1:
				acct.Secret = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(param.bytes)
			case 2:
				acct.Account = string(param.bytes)
			case 3:
				acct.Issuer = string(param.bytes)
			}
		}

		if acct.Issuer != "" {
			acct.Account = strings.TrimPrefix(acct.Account, acct.Issuer+":")
		}

		accounts = append(accounts, acct)
	}

	return accounts, nil
}

func readQR(imagePath string) (string, error) {
	out, err := exec.Command("zbarimg", "--raw", "-q", imagePath).Output()
	if err != nil {
		if _, ok := err.(*exec.Error); ok {
			return "", fmt.Errorf("error: zbarimg not found, install it first please")
		}
		return "", fmt.Errorf("error: no QR code found in %s", imagePath)
	}
	return strings.TrimSpace(string(out)), nil
}

func ImportOTPQR(imagePath string) (string, error) {
	raw, err := readQR(imagePath)
	if err != nil {
		return "", err
	}

	raw = strings.Split(raw, "\n")[0]

	if strings.HasPrefix(raw, "otpauth-migration://") {
		return "", fmt.Errorf("error: this is an export QR, use /importotp instead")
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "otpauth" {
		return "", fmt.Errorf("error: no OTP QR code found in %s", imagePath)
	}

	secret := parsed.Query().Get("secret")
	if secret == "" {
		return "", fmt.Errorf("error: the QR code has no secret key")
	}

	label := strings.TrimPrefix(parsed.Path, "/")
	issuer := parsed.Query().Get("issuer")

	if issuer != "" {
		label = strings.TrimPrefix(label, issuer+":")
	}

	name := label
	if issuer != "" {
		name = issuer + "-" + label
	}

	if err := SavePassword(Credentials{Name: name, Password: secret, IsOTP: true}); err != nil {
		return "", err
	}

	return name, nil
}

func ImportGoogleOTP(imagePath string) ([]string, error) {
	migrationURL, err := readQR(imagePath)
	if err != nil {
		return nil, err
	}

	accounts, err := decodeMigrationPayload(migrationURL)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("error: no OTP accounts found in %s", imagePath)
	}

	var added []string
	seen := map[string]bool{}
	for _, acct := range accounts {
		name := acct.Account
		if acct.Issuer != "" {
			name = acct.Issuer + "-" + acct.Account
		}

		if seen[name] {
			continue
		}
		seen[name] = true

		if err := SavePassword(Credentials{Name: name, Password: acct.Secret, IsOTP: true}); err != nil {
			continue
		}

		added = append(added, name)
	}

	return added, nil
}
