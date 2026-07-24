package gui

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/awesome-gocui/gocui"

	"github.com/ahmedhosssam/vd/internal/app"
	"github.com/ahmedhosssam/vd/internal/fuzzy"
)

const searchBarView = "searchbar"

const passphraseView = "passphrase"

const searchBarContentRows = 1

const searchBarMiddleRow = searchBarContentRows / 2

const (
	barContentRows = 1
	barWidth       = 60
)

const maxRows = 20

var numRows = 0

var currentRow = -1

var gui *gocui.Gui

var passwords []string

var (
	locked        bool
	passphraseMsg string
)

func Run() {
	app.NoTerminalPrompt = true

	passwords = app.ReadPasswordsLookup()
	locked = passwords == nil

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorCyan

	gui = g

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(passphraseView, gocui.KeyEnter, gocui.ModNone, unlock); err != nil {
		log.Panicln(err)
	}

	for i := range maxRows {
		if err := g.SetKeybinding(barViewName(i), gocui.KeyEnter, gocui.ModNone, copyRowAndQuit); err != nil {
			log.Panicln(err)
		}
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
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
	if locked {
		return passphraseLayout(g)
	}

	maxX, maxY := g.Size()
	w := barWidth
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

	barH := barContentRows + 1
	if err := renderRows(g, x0, y1, x1, barH); err != nil {
		return err
	}

	return nil
}

func passphraseLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	w := barWidth
	h := searchBarContentRows + 1
	x0 := (maxX - w) / 2
	y0 := (maxY - h) / 2

	v, err := g.SetView(passphraseView, x0, y0, x0+w, y0+h, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Editable = true
		v.Wrap = false
		v.Mask = '*'
		v.Editor = gocui.EditorFunc(passphraseEditor)
		if _, err := g.SetCurrentView(passphraseView); err != nil {
			return err
		}
	}

	v.Title = " GPG passphrase "
	if passphraseMsg != "" {
		v.Title = " " + passphraseMsg + " "
	}

	return nil
}

func unlock(g *gocui.Gui, v *gocui.View) error {
	app.PassphraseCache = strings.TrimRight(v.Buffer(), "\r\n")

	passwords = app.ReadPasswordsLookup()
	if passwords == nil {
		app.PassphraseCache = ""
		passphraseMsg = "wrong passphrase, try again"
		v.Clear()
		if err := v.SetOrigin(0, 0); err != nil {
			return err
		}
		return v.SetCursorUnrestricted(0, 0)
	}

	locked = false
	return g.DeleteView(passphraseView)
}

func passphraseEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch key {
	case gocui.KeyEnter, gocui.KeyCtrlJ:
		return
	case gocui.KeyArrowUp, gocui.KeyArrowDown:
		return
	case gocui.KeyBackspace, gocui.KeyBackspace2:
		if cx, _ := v.Cursor(); cx == 0 {
			return
		}
	}
	gocui.DefaultEditor.Edit(v, key, ch, mod)
}

func renderRows(g *gocui.Gui, x0, y1, x1, barH int) error {
	sv, err := g.View(searchBarView)
	if err != nil {
		return nil
	}
	pattern := strings.TrimSpace(sv.ViewBuffer())
	matches := fuzzy.Search(pattern, passwords)
	numRows = len(matches)

	for i := 0; i < numRows; i++ {
		by0 := y1 + 1 + i*barH
		by1 := by0 + barH

		bv, err := g.SetView(barViewName(i), x0, by0, x1, by1, 0)
		isNew := errors.Is(err, gocui.ErrUnknownView)
		if err != nil && !isNew {
			return err
		}
		bv.Wrap = false
		if isNew {
			bv.Frame = false
			bv.BgColor = gocui.ColorWhite
			bv.FgColor = gocui.ColorBlack
		} else if i == currentRow {
			bv.Frame = true
		} else {
			bv.Frame = false
		}
		bv.Clear()
		fmt.Fprint(bv, matches[i].Str)
	}

	// Delete stale bar views beyond the current match count.
	for i := numRows; i < maxRows; i++ {
		if _, err := g.View(barViewName(i)); err == nil {
			if err := g.DeleteView(barViewName(i)); err != nil {
				return err
			}
		}
	}

	if currentRow >= numRows {
		_ = setRowView(g, -1)
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func barViewName(i int) string {
	return fmt.Sprintf("bar%d", i)
}

func setRowView(g *gocui.Gui, i int) error {
	if locked || i < -1 || i >= numRows {
		return nil
	}
	if currentRow >= 0 {
		if old, err := g.View(barViewName(currentRow)); err == nil {
			old.Frame = false
		}
	}
	currentRow = i
	name := searchBarView
	if i >= 0 {
		name = barViewName(i)
		if nv, err := g.View(name); err == nil {
			nv.Frame = true
		}
	}
	_, err := g.SetCurrentView(name)
	return err
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	return setRowView(g, currentRow-1)
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	return setRowView(g, currentRow+1)
}

func copyRowAndQuit(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		app.GetPassword(strings.TrimSpace(v.ViewBuffer()))
	}
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

	cx, cy := v.Cursor()

	gui.Update(func(g *gocui.Gui) error {
		if err := layout(g); err != nil {
			return err
		}
		if _, err := g.SetCurrentView(searchBarView); err != nil {
			return err
		}
		if sv, err := g.View(searchBarView); err == nil {
			_ = sv.SetCursorUnrestricted(cx, searchBarMiddleRow)
		}
		return nil
	})

	if cy != searchBarMiddleRow {
		_ = v.SetCursorUnrestricted(cx, searchBarMiddleRow)
	}
}
