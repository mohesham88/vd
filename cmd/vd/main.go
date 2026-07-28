package main

import (
	"os"

	"github.com/ahmedhosssam/vd/internal/app"
	"github.com/ahmedhosssam/vd/internal/gui"
)

func main() {
	if len(os.Args) >= 2 {
		os.Exit(app.Run())
	} else {
		gui.Run()
	}
}

