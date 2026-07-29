package app

import "io"

type Credentials struct {
	Name     string `json:"Name"`
	Password string `json:"Password"`
	IsOTP    bool   `json:"isOTP"`
}

type prefixWriter struct {
	w      io.Writer
	prefix []byte
	atBOL  bool
}
