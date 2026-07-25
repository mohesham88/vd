package gui

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
	"github.com/sahilm/fuzzy"

	"github.com/ahmedhosssam/vd/internal/app"
)

const (
	GpgPassphraseView = "gpgpassphraseview"
	PasswordsView     = "passwordsview"
	FeedbackView      = "feedbackview"
)

var (
	gg            bool
	passphraseMsg string
	viewStack     []string
	passwords     []string
	gui           *gocui.Gui
	selectedRow   int
	rowCount      int
	query         string
	feedbackMsg   string
	feedbackID    int
)

func Run() {
	app.NoTerminalPrompt = true

	passwords = app.ReadPasswordsLookup()
	locked := passwords == nil

	viewStack = append(viewStack, PasswordsView)

	if locked {
		viewStack = append(viewStack, GpgPassphraseView)
	}

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	gui = g

	g.Cursor = true

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlG, gocui.ModNone, changeLayout); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding(GpgPassphraseView, gocui.KeyEnter, gocui.ModNone, unlock); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding(PasswordsView, gocui.KeyArrowDown, gocui.ModNone, nextRow); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding(PasswordsView, gocui.KeyArrowUp, gocui.ModNone, prevRow); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding(PasswordsView, gocui.KeyEnter, gocui.ModNone, handleGetPassword); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Panicln(err)
	}
}

func layout(g *gocui.Gui) error {
	topStack := viewStack[len(viewStack)-1]

	switch topStack {
	case PasswordsView:
		return passwordsLayout(g)
	case GpgPassphraseView:
		return gpgPassphraseLayout(g)
	default:
		return nil
	}
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func changeLayout(g *gocui.Gui, v *gocui.View) error {
	gg = !gg
	return nil
}

func gpgPassphraseLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	w := 60
	h := 2
	x0 := (maxX - w) / 2
	y0 := (maxY - h) / 2
	x1 := x0 + w
	y1 := y0 + h

	v, err := g.SetView(GpgPassphraseView, x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Editable = true
		v.Wrap = false
		v.Mask = '*'
		v.Editor = gocui.EditorFunc(passphraseEditor)

		if _, err := g.SetCurrentView(GpgPassphraseView); err != nil {
			return err
		}

	}

	v.Title = " Enter GPG Passphrase "
	if passphraseMsg != "" {
		v.Title = fmt.Sprintf(" %s ", passphraseMsg)
	}
	return nil
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

func passwordsSearchEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
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

	newQuery := strings.TrimRight(v.Buffer(), "\r\n")
	if newQuery != query {
		query = newQuery
		selectedRow = 0
	}
}

func unlock(g *gocui.Gui, v *gocui.View) error {
	app.PassphraseCache = strings.TrimRight(v.Buffer(), "\r\n")
	passwords = app.ReadPasswordsLookup()

	if passwords == nil {
		app.PassphraseCache = ""
		v.Clear()
		passphraseMsg = " Wrong passphrase, try again "
		if err := v.SetOrigin(0, 0); err != nil {
			return err
		}
		return v.SetCursorUnrestricted(0, 0)
	}

	viewStack = viewStack[:len(viewStack)-1]
	passphraseMsg = ""

	if err := g.DeleteView(GpgPassphraseView); err != nil {
		return err
	}

	return nil
}

func passwordsLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	w := 60
	h := 2
	x0 := (maxX - w) / 2
	y0 := (maxY - h) / 2
	x1 := x0 + w
	y1 := y0 + h

	v, err := g.SetView(PasswordsView, x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		v.Editable = true
		v.Wrap = false
		v.Editor = gocui.EditorFunc(passwordsSearchEditor)

		if _, err := g.SetCurrentView(PasswordsView); err != nil {
			return err
		}
	}

	if err := renderFeedback(g, x0, y0, x1); err != nil {
		return err
	}

	if err := renderRows(g, x0, y1+1, x1); err != nil {
		return err
	}

	return nil
}

func renderFeedback(g *gocui.Gui, x0, y0, x1 int) error {
	if feedbackMsg == "" {
		if _, err := g.View(FeedbackView); err == nil {
			return g.DeleteView(FeedbackView)
		}
		return nil
	}

	fv, err := g.SetView(FeedbackView, x0, y0-2, x1, y0, 0)
	if err != nil && !errors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	fv.Frame = false
	fv.FgColor = gocui.ColorMagenta

	fv.Clear()
	fmt.Fprint(fv, feedbackMsg)

	return nil
}

func showFeedback(g *gocui.Gui, msg string) {
	feedbackMsg = msg
	feedbackID++
	id := feedbackID

	go func() {
		time.Sleep(3 * time.Second)
		g.Update(func(g *gocui.Gui) error {
			if id == feedbackID {
				feedbackMsg = ""
			}
			return nil
		})
	}()
}

func nextRow(g *gocui.Gui, v *gocui.View) error {
	if selectedRow < rowCount-1 {
		selectedRow++
	}
	return nil
}

func prevRow(g *gocui.Gui, v *gocui.View) error {
	if selectedRow > 0 {
		selectedRow--
	}
	return nil
}

func renderRows(g *gocui.Gui, x0, y1, x1 int) error {
	entries := filterPasswords()

	numRows := len(entries)
	rowH := 2

	rowCount = numRows
	if selectedRow >= numRows {
		selectedRow = numRows - 1
	}
	if selectedRow < 0 {
		selectedRow = 0
	}

	for i := range numRows {
		by0 := y1 + i*(rowH)
		by1 := by0 + rowH

		bv, err := g.SetView(fmt.Sprintf("row%d", i), x0, by0, x1, by1, 0)
		isNew := errors.Is(err, gocui.ErrUnknownView)
		if err != nil && !isNew {
			return err
		}
		bv.Wrap = false
		bv.Frame = false

		if i == selectedRow {
			bv.BgColor = gocui.ColorMagenta
		} else {
			bv.BgColor = gocui.ColorWhite
		}
		bv.FgColor = gocui.ColorBlack

		bv.Clear()
		fmt.Fprint(bv, entries[i])
	}

	for i := numRows; ; i++ {
		name := fmt.Sprintf("row%d", i)
		if _, err := g.View(name); err != nil {
			break
		}
		if err := g.DeleteView(name); err != nil {
			return err
		}
	}

	return nil
}

func filterPasswords() []string {
	if query == "" {
		return passwords
	}

	matches := fuzzy.Find(query, passwords)
	entries := make([]string, len(matches))
	for i, m := range matches {
		entries[i] = m.Str
	}
	return entries
}

func handleGetPassword(g *gocui.Gui, v *gocui.View) error {
	entries := filterPasswords()

	if len(entries) == 0 {
		return nil
	}

	passwordName := entries[selectedRow]

	passwordsTmp := app.ReadPasswordsLookup()

	if passwordsTmp == nil {
		viewStack = append(viewStack, GpgPassphraseView)
	}

	app.GetPassword(passwordName)
	showFeedback(g, fmt.Sprintf("Password for `%s` copied to clipboard!", passwordName))
	return nil
}
