package gui

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/awesome-gocui/gocui"

	"github.com/ahmedhosssam/vd/internal/app"
)

const searchBarView = "searchbar"

const searchBarContentRows = 1

const searchBarMiddleRow = searchBarContentRows / 2

const (
	numBars        = 5
	barContentRows = 1
	barWidth       = 60
	barTextLength  = 10
)

const barChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var currentBar = -1

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
	for i := 0; i < numBars; i++ {
		if err := g.SetKeybinding(barViewName(i), gocui.KeyEnter, gocui.ModNone, copyBarAndQuit); err != nil {
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
	for i := 0; i < numBars; i++ {
		name := fmt.Sprintf("bar%d", i)
		bx0 := x0
		by0 := y1 + 1 + i*(barH)
		bx1 := x1
		by1 := by0 + barH

		bv, err := g.SetView(name, bx0, by0, bx1, by1, 0)
		if err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}
			bv.Wrap = false
			bv.Frame = false
			bv.BgColor = gocui.ColorWhite
			bv.FgColor = gocui.ColorBlack
			fmt.Fprint(bv, randomText(barTextLength))
		}
	}

	return nil
}

func randomText(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = barChars[rand.Intn(len(barChars))]
	}
	return string(b)
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func barViewName(i int) string {
	return fmt.Sprintf("bar%d", i)
}

func setBarView(g *gocui.Gui, i int) error {
	if i < -1 || i >= numBars {
		return nil
	}
	if currentBar >= 0 {
		if old, err := g.View(barViewName(currentBar)); err == nil {
			old.Frame = false
		}
	}
	currentBar = i
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
	return setBarView(g, currentBar-1)
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	return setBarView(g, currentBar+1)
}

func submit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func copyBarAndQuit(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		_ = app.CopyToClipboard(strings.TrimSpace(v.ViewBuffer()))
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

	if cx, cy := v.Cursor(); cy != searchBarMiddleRow {
		_ = v.SetCursorUnrestricted(cx, searchBarMiddleRow)
	}
}
