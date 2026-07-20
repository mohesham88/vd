package app

import (
	"crypto/rand"
	"math/big"
)

func generateRandomPassword() (string, error) {
	charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()"
	length := 20

	finalPassword := make([]byte, length)
	max := big.NewInt(int64(len(charset)))

	for i := range finalPassword {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		finalPassword[i] = charset[n.Int64()]
	}

	return string(finalPassword), nil
}
