package gui

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/awesome-gocui/gocui"
)

const searchBarView = "searchbar"

const searchBarContentRows = 1

const searchBarMiddleRow = searchBarContentRows / 2

func Run() {
	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorCyan

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(searchBarView, gocui.KeyEnter, gocui.ModNone, submit); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Panicln(err)
	}

	if v, err := g.View(searchBarView); err == nil && v != nil {
		fmt.Printf("Search: %q\n", strings.TrimSpace(v.ViewBuffer()))
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	w := 60
	h := searchBarContentRows + 1
	x0 := (maxX - w) / 2
	y0 := (maxY - h) / 2
	x1 := x0 + w
	y1 := y0 + h

	v, err := g.SetView(searchBarView, x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Editable = true
		v.Wrap = false
		v.Editor = gocui.EditorFunc(oneLineEditor)
		if err := v.SetCursorUnrestricted(0, searchBarMiddleRow); err != nil {
			return err
		}
		if _, err := g.SetCurrentView(searchBarView); err != nil {
			return err
		}
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func submit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func oneLineEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch key {
	case gocui.KeyEnter, gocui.KeyCtrlJ: // ignore newline insertion
		return
	case gocui.KeyArrowUp, gocui.KeyArrowDown:
		return
	case gocui.KeyBackspace, gocui.KeyBackspace2:
		cx, _ := v.Cursor()
		if cx == 0 {
			return
		}
	}
	gocui.DefaultEditor.Edit(v, key, ch, mod)

	if cx, cy := v.Cursor(); cy != searchBarMiddleRow {
		_ = v.SetCursorUnrestricted(cx, searchBarMiddleRow)
	}
}
