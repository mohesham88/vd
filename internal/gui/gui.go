package gui

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
	"github.com/sahilm/fuzzy"

	"github.com/ahmedhosssam/vd/internal/app"
)

const (
	GpgPassphraseView  = "gpgpassphraseview"
	PasswordsView      = "passwordsview"
	AddPasswordView    = "addpasswordview"
	AddNameView        = "addnameview"
	AddPassView        = "addpassview"
	AddConfirmView     = "addconfirmview"
	FeedbackView       = "feedbackview"
	DeleteConfirmView  = "deleteconfirmview"
	DeleteRowView      = "deleterowview"
	ChangePasswordView = "changepasswordview"
	BannerView         = "bannerview"
	PlaceholderView    = "placeholderview"
)

const maxVisibleRows = 12

var banner = []string{
	`__      _______  `,
	`\ \    / /  __ \ `,
	` \ \  / /| |  | |`,
	`  \ \/ / | |  | |`,
	`   \  /  | |__| |`,
	`    \/   |_____/ `,
}

var (
	addViews     = []string{AddNameView, AddPassView, AddConfirmView}
	addTitles    = []string{" Password Name ", " Password ", " Password Confirmation "}
	changeViews  = []string{AddNameView, AddPassView, AddConfirmView}
	changeTitles = []string{" Password Name ", " New Password ", " New Password Confirmation "}
	otpViews     = []string{AddNameView, AddPassView}
	otpTitles    = []string{" OTP Name ", " OTP Secret Key "}
	addFocus     int
)

var Commands = []string{"/add", "/addotp", "/delete", "/change", "/gen"}

var (
	gg            bool
	passphraseMsg string
	viewStack     []string
	passwords     []string
	otpNames      []string
	gui           *gocui.Gui
	selectedRow   int
	rowOffset     int
	rowCount      int
	query         string
	feedbackMsg   string
	feedbackID    int
	deleteMode    bool
	deleteTarget  string
	changeMode    bool
	changeTarget  string
	changeOldPass string
	otpMode       bool
)

func refreshPasswordsOnScreen() {
	passwords = app.ReadPasswordsLookup()
	otpNames = app.ReadOTPLookup()
}

func Run() {
	app.NoTerminalPrompt = true

	refreshPasswordsOnScreen()
	locked := passwords == nil

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	gui = g

	pushToViewStack(PasswordsView)

	if locked {
		pushToViewStack(GpgPassphraseView)
	}

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

	if err := g.SetKeybinding(PasswordsView, gocui.KeyEnter, gocui.ModNone, handleSearchBarEnter); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding(PasswordsView, gocui.KeyEsc, gocui.ModNone, cancelSelectionMode); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding(DeleteConfirmView, gocui.KeyEnter, gocui.ModNone, confirmDelete); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding(DeleteConfirmView, gocui.KeyEsc, gocui.ModNone, cancelDelete); err != nil {
		log.Panicln(err)
	}

	for _, name := range addViews {
		if err := g.SetKeybinding(name, gocui.KeyTab, gocui.ModNone, nextAddField); err != nil {
			log.Panicln(err)
		}

		if err := g.SetKeybinding(name, gocui.KeyArrowDown, gocui.ModNone, nextAddField); err != nil {
			log.Panicln(err)
		}

		if err := g.SetKeybinding(name, gocui.KeyBacktab, gocui.ModNone, prevAddField); err != nil {
			log.Panicln(err)
		}

		if err := g.SetKeybinding(name, gocui.KeyArrowUp, gocui.ModNone, prevAddField); err != nil {
			log.Panicln(err)
		}

		if err := g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, submitAddPassword); err != nil {
			log.Panicln(err)
		}

		if err := g.SetKeybinding(name, gocui.KeyEsc, gocui.ModNone, closeAddPassword); err != nil {
			log.Panicln(err)
		}
	}

	for _, name := range []string{AddNameView, AddPassView, AddConfirmView} {
		if err := g.SetKeybinding(name, gocui.KeyCtrlG, gocui.ModNone, generateAddPassword); err != nil {
			log.Panicln(err)
		}
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
	case AddPasswordView:
		return addPasswordLayout(g)
	case DeleteConfirmView:
		return deleteConfirmLayout(g)
	case ChangePasswordView:
		return addPasswordLayout(g)
	default:
		return nil
	}
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func clearScreen(g *gocui.Gui, keep ...string) error {
	var names []string
	for _, v := range g.Views() {
		names = append(names, v.Name())
	}

	for _, name := range names {
		if slices.Contains(keep, name) {
			continue
		}
		if err := g.DeleteView(name); err != nil {
			return err
		}
	}
	return nil
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

	return renderBanner(g, y0)
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
		clearFeedback()
		rowOffset = 0
		if isCommandsQuery() {
			selectedRow = 1<<31 - 1
		} else {
			selectedRow = 0
		}
	}
}

func unlock(g *gocui.Gui, v *gocui.View) error {
	app.PassphraseCache = strings.TrimRight(v.Buffer(), "\r\n")
	refreshPasswordsOnScreen()

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

	g.Cursor = true

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

	if err := renderBanner(g, y0); err != nil {
		return err
	}

	if err := renderFeedback(g, x0, y0, x1); err != nil {
		return err
	}

	if err := renderRows(g); err != nil {
		return err
	}

	// Show the placeholder only in the search mode
	if !deleteMode && !changeMode {
		searchPlaceholder := "Search passwords, or type / for commands"
		if err := renderPlaceholder(g, searchPlaceholder, x0, y0, x1); err != nil {
			return err
		}
	}

	return nil
}

func renderPlaceholder(g *gocui.Gui, placeholderMsg string, x0, y0, x1 int) error {
	if query != "" {
		if _, err := g.View(PlaceholderView); err == nil {
			return g.DeleteView(PlaceholderView)
		}
		return nil
	}

	pv, err := g.SetView(PlaceholderView, x0, y0, x1, y0+2, 0)
	if err != nil && !errors.Is(err, gocui.ErrUnknownView) {
		return err
	}

	pv.Frame = false
	pv.FgColor = gocui.ColorDefault | gocui.AttrDim

	pv.Clear()

	fmt.Fprint(pv, placeholderMsg)

	return nil
}

func renderBanner(g *gocui.Gui, y0 int) error {
	maxX, _ := g.Size()
	w := len(banner[0])
	x0 := (maxX - w) / 2
	by1 := y0 - 7
	by0 := by1 - len(banner) - 1

	v, err := g.SetView(BannerView, x0, by0, x0+w+1, by1, 0)
	if err != nil && !errors.Is(err, gocui.ErrUnknownView) {
		return err
	}

	v.Frame = false
	v.FgColor = gocui.ColorMagenta
	v.Clear()
	fmt.Fprint(v, strings.Join(banner, "\n"))

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

func clearFeedback() {
	feedbackMsg = ""
	feedbackID++
}

func nextRow(g *gocui.Gui, v *gocui.View) error {
	if isCommandsQuery() {
		return moveUp()
	}
	return moveDown()
}

func prevRow(g *gocui.Gui, v *gocui.View) error {
	if isCommandsQuery() {
		return moveDown()
	}
	return moveUp()
}

func moveDown() error {
	if rowCount == 0 {
		return nil
	}
	if selectedRow < rowCount-1 {
		selectedRow++
	} else {
		selectedRow = 0
	}
	return nil
}

func moveUp() error {
	if rowCount == 0 {
		return nil
	}
	if selectedRow > 0 {
		selectedRow--
	} else {
		selectedRow = rowCount - 1
	}
	return nil
}

func isCommandsQuery() bool {
	return !deleteMode && !changeMode && strings.HasPrefix(query, "/")
}

func renderRows(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	w := 60
	h := 2
	x0 := (maxX - w) / 2
	y0 := (maxY - h) / 2
	x1 := x0 + w
	y1 := y0 + h + 1

	var entries []string

	isCommands := isCommandsQuery()

	if isCommands {
		entries = filterCommands()
	} else {
		entries = filterPasswords()
	}

	numRows := len(entries)
	rowH := 2

	rowCount = numRows
	if selectedRow >= numRows {
		selectedRow = numRows - 1
	}
	if selectedRow < 0 {
		selectedRow = 0
	}

	if selectedRow < rowOffset {
		rowOffset = selectedRow
	}

	if selectedRow >= (rowOffset + maxVisibleRows) {
		rowOffset = selectedRow - maxVisibleRows + 1
	}

	if rowOffset > (numRows - maxVisibleRows) {
		rowOffset = numRows - maxVisibleRows
	}

	if rowOffset < 0 {
		rowOffset = 0
	}

	visibleRows := min(numRows-rowOffset, maxVisibleRows)

	for i := range visibleRows {
		entry := entries[rowOffset+i]
		by0 := y1 + i*rowH

		if isCommands {
			by0 = -5 + y1 - i*(rowH-1)
		}

		by1 := by0 + rowH

		bv, err := g.SetView(fmt.Sprintf("row%d", i), x0, by0, x1, by1, 0)
		isNew := errors.Is(err, gocui.ErrUnknownView)
		if err != nil && !isNew {
			return err
		}
		bv.Wrap = false
		bv.Frame = false

		if rowOffset+i == selectedRow {
			bv.BgColor = gocui.ColorMagenta
			bv.FgColor = gocui.ColorBlack
		} else {
			if isCommands {
				bv.BgColor = gocui.ColorBlack
				bv.FgColor = gocui.ColorWhite
			} else {
				bv.BgColor = gocui.ColorWhite
				bv.FgColor = gocui.ColorBlack
			}
		}

		bv.Clear()
		if isCommands {
			fmt.Fprint(bv, entry)
		} else {
			fmt.Fprint(bv, entryLabel(entry))
		}
	}

	for i := visibleRows; ; i++ {
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

func entryLabel(name string) string {
	if slices.Contains(otpNames, name) {
		return name + " [OTP]"
	}
	return name
}

func filterCommands() []string {
	matches := fuzzy.Find(query, Commands)
	entries := make([]string, len(matches))

	for i, m := range matches {
		entries[i] = m.Str
	}

	return entries
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

func handleSearchBarEnter(g *gocui.Gui, v *gocui.View) error {
	var result error
	switch {
	case isCommandsQuery():
		result = handleCommand(g)
	case deleteMode:
		result = handleDeletePassword(g)
	case changeMode:
		result = handleChangePassword(g)
	default:
		result = handleGetPassword(g)
	}

	query = ""
	v.Clear()
	v.SetCursorUnrestricted(0, 0)

	return result
}

func handleGetPassword(g *gocui.Gui) error {
	entries := filterPasswords()

	if len(entries) == 0 {
		return nil
	}

	passwordName := entries[selectedRow]

	passwordsTmp := app.ReadPasswordsLookup()

	if passwordsTmp == nil {
		pushToViewStack(GpgPassphraseView)
	}

	if err := app.GetPassword(passwordName); err != nil {
		showFeedback(g, fmt.Sprintf("Error: %v", err))
		return nil
	}

	if slices.Contains(otpNames, passwordName) {
		showFeedback(g, fmt.Sprintf("OTP code for `%s` copied to clipboard!", passwordName))
		return nil
	}

	showFeedback(g, fmt.Sprintf("Password for `%s` copied to clipboard!", passwordName))
	return nil
}

func handleCommand(g *gocui.Gui) error {
	commands := filterCommands()

	commandName := commands[selectedRow]

	switch commandName {
	case "/gen":
		randomPassword, err := app.GenerateRandomPassword()
		if err != nil {
			showFeedback(g, "Error happened during generating random password")
			return err
		}

		if err := app.CopyToClipboard(randomPassword); err != nil {
			showFeedback(g, "Error happened during copying to clipboard")
			return nil
		}

		showFeedback(g, "Generated password copied to clipboard!")
	case "/add":
		pushToViewStack(AddPasswordView)
	case "/addotp":
		otpMode = true
		addFocus = 0
		pushToViewStack(AddPasswordView)
	case "/delete":
		deleteMode = true
		selectedRow = 0
		showFeedback(g, "Choose the password you want to delete")
	case "/change":
		changeMode = true
		selectedRow = 0
		showFeedback(g, "Choose the password you want to change")
	}

	return nil
}

func handleChangePassword(g *gocui.Gui) error {
	entries := filterPasswords()

	if len(entries) == 0 {
		return nil
	}

	changeTarget = entries[selectedRow]

	creds, err := app.LoadCredentials(changeTarget)
	if err != nil {
		resetChangeMode()
		showFeedback(g, err.Error())
		return nil
	}
	changeOldPass = creds.Password

	addFocus = 0
	pushToViewStack(ChangePasswordView)
	showFeedback(g, fmt.Sprintf("Changing the password for `%s`", changeTarget))

	return nil
}

func handleDeletePassword(g *gocui.Gui) error {
	entries := filterPasswords()

	if len(entries) == 0 {
		return nil
	}

	deleteTarget = entries[selectedRow]
	pushToViewStack(DeleteConfirmView)

	return nil
}

func deleteConfirmLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	w := 60
	h := 2
	x0 := (maxX - w) / 2
	y0 := (maxY - h) / 2
	x1 := x0 + w
	y1 := y0 + h

	g.Cursor = false

	if err := clearScreen(g, DeleteConfirmView, DeleteRowView); err != nil {
		return err
	}

	mv, err := g.SetView(DeleteConfirmView, x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}

		if _, err := g.SetCurrentView(DeleteConfirmView); err != nil {
			return err
		}
	}

	mv.Frame = false
	mv.FgColor = gocui.ColorMagenta
	mv.Clear()
	fmt.Fprintf(mv, "Are you sure you want to delete `%s`?", deleteTarget)

	rv, err := g.SetView(DeleteRowView, x0, y1, x1, y1+2, 0)
	if err != nil && !errors.Is(err, gocui.ErrUnknownView) {
		return err
	}

	rv.Frame = false
	rv.BgColor = gocui.ColorMagenta
	rv.FgColor = gocui.ColorBlack
	rv.Clear()
	fmt.Fprint(rv, deleteTarget)

	return nil
}

func confirmDelete(g *gocui.Gui, v *gocui.View) error {
	name := deleteTarget
	success, err := app.DeletePassword(name)

	resetDeleteMode()
	popViewStack()

	if err != nil || !success {
		showFeedback(g, fmt.Sprintf("Error happened while deleting `%s`", name))
		return nil
	}

	refreshPasswordsOnScreen()
	showFeedback(g, fmt.Sprintf("Password for `%s` deleted!", name))

	return nil
}

func cancelDelete(g *gocui.Gui, v *gocui.View) error {
	resetDeleteMode()
	popViewStack()
	return nil
}

func cancelSelectionMode(g *gocui.Gui, v *gocui.View) error {
	switch {
	case deleteMode:
		resetDeleteMode()
		showFeedback(g, "Deletion cancelled")
	case changeMode:
		resetChangeMode()
		showFeedback(g, "Change cancelled")
	}

	return nil
}

func resetDeleteMode() {
	deleteMode = false
	deleteTarget = ""
	selectedRow = 0
}

func pushToViewStack(v string) {
	clearScreen(gui)
	viewStack = append(viewStack, v)
	query = ""
}

func popViewStack() {
	clearScreen(gui)
	viewStack = viewStack[:len(viewStack)-1]
	clearScreen(gui)
}

func activeAddFields() ([]string, []string) {
	if changeMode {
		return changeViews, changeTitles
	}
	if otpMode {
		return otpViews, otpTitles
	}
	return addViews, addTitles
}

func defaultFieldValue(name string) string {
	if otpMode {
		return ""
	}

	switch name {
	case AddNameView:
		return changeTarget
	case AddPassView, AddConfirmView:
		return changeOldPass
	}

	return ""
}

func addPasswordLayout(g *gocui.Gui) error {
	views, titles := activeAddFields()

	maxX, maxY := g.Size()
	w := 60
	h := 2
	x0 := (maxX - w) / 2
	y0 := (maxY - len(views)*(h+1)) / 2
	x1 := x0 + w

	g.Cursor = true

	for i, name := range views {
		vy0 := y0 + i*(h+1)

		v, err := g.SetView(name, x0, vy0, x1, vy0+h, 0)
		if err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				return err
			}

			v.Editable = true
			v.Wrap = false
			v.Editor = gocui.EditorFunc(passphraseEditor)

			if name != AddNameView {
				v.Mask = '*'
			}

			if changeMode {
				if def := defaultFieldValue(name); def != "" {
					v.WriteString(def)
					if err := v.SetCursor(len(def), 0); err != nil {
						return err
					}
				}
			}
		}

		v.Title = titles[i]

		if name == AddPassView && !changeMode {
			if err := togglePlaceholder(g, v, x0, vy0, x1); err != nil {
				return err
			}
		}
	}

	if err := renderFeedback(g, x0, y0, x1); err != nil {
		return err
	}

	_, err := g.SetCurrentView(views[addFocus])
	return err
}

func togglePlaceholder(g *gocui.Gui, v *gocui.View, x0, y0, x1 int) error {
	filled := strings.TrimRight(v.Buffer(), "\r\n") != ""

	if cv, err := g.View(AddConfirmView); err == nil && strings.TrimRight(cv.Buffer(), "\r\n") != "" {
		filled = true
	}

	if !filled {
		return renderPlaceholder(g, "ctrl+g to generate password", x0, y0, x1)
	}

	if _, err := g.View(PlaceholderView); err == nil {
		return g.DeleteView(PlaceholderView)
	}

	return nil
}

func generateAddPassword(g *gocui.Gui, v *gocui.View) error {
	if otpMode {
		return nil
	}

	password, err := app.GenerateRandomPassword()
	if err != nil {
		showFeedback(g, "Error happened during generating random password")
		return nil
	}

	for _, name := range []string{AddPassView, AddConfirmView} {
		fv, err := g.View(name)
		if err != nil {
			continue
		}

		fv.Mask = 0
		fv.Clear()
		fv.WriteString(password)

		if err := fv.SetOrigin(0, 0); err != nil {
			return err
		}

		if err := fv.SetCursor(len(password), 0); err != nil {
			return err
		}
	}

	return nil
}

func nextAddField(g *gocui.Gui, v *gocui.View) error {
	views, _ := activeAddFields()
	addFocus = (addFocus + 1) % len(views)
	return nil
}

func prevAddField(g *gocui.Gui, v *gocui.View) error {
	views, _ := activeAddFields()
	addFocus = (addFocus - 1 + len(views)) % len(views)
	return nil
}

func addFieldValue(g *gocui.Gui, name string) string {
	v, err := g.View(name)
	if err != nil {
		return ""
	}
	return strings.TrimRight(v.Buffer(), "\r\n")
}

func submitAddPassword(g *gocui.Gui, v *gocui.View) error {
	if changeMode {
		return submitChangePassword(g, v)
	}

	if otpMode {
		return submitAddOTP(g, v)
	}

	name := addFieldValue(g, AddNameView)
	password := addFieldValue(g, AddPassView)
	confirmation := addFieldValue(g, AddConfirmView)

	if name == "" || password == "" {
		showFeedback(g, "Password name and password can't be empty")
		return nil
	}

	if password != confirmation {
		showFeedback(g, "Passwords don't match")
		return nil
	}

	if err := app.SavePassword(app.Credentials{Name: name, Password: password}); err != nil {
		showFeedback(g, err.Error())
		return nil
	}

	refreshPasswordsOnScreen()
	showFeedback(g, fmt.Sprintf("Password for `%s` saved!", name))

	return closeAddPassword(g, v)
}

func submitAddOTP(g *gocui.Gui, v *gocui.View) error {
	name := addFieldValue(g, AddNameView)
	secret := addFieldValue(g, AddPassView)

	if name == "" || secret == "" {
		showFeedback(g, "OTP name and secret key can't be empty")
		return nil
	}

	if err := app.SavePassword(app.Credentials{Name: name, Password: secret, IsOTP: true}); err != nil {
		showFeedback(g, err.Error())
		return nil
	}

	refreshPasswordsOnScreen()
	showFeedback(g, fmt.Sprintf("OTP for `%s` saved!", name))

	return closeAddPassword(g, v)
}

func submitChangePassword(g *gocui.Gui, v *gocui.View) error {
	name := addFieldValue(g, AddNameView)
	password := addFieldValue(g, AddPassView)
	confirmation := addFieldValue(g, AddConfirmView)

	if name == "" || password == "" {
		showFeedback(g, "Password name and password can't be empty")
		return nil
	}

	if password != confirmation {
		showFeedback(g, "Passwords don't match")
		return nil
	}

	success, err := app.ChangePassword(changeTarget, password)
	if err != nil || !success {
		showFeedback(g, fmt.Sprintf("Error happened while changing `%s`", changeTarget))
		return closeAddPassword(g, v)
	}

	if name != changeTarget {
		if success, err := app.RenamePassword(changeTarget, name); err != nil || !success {
			refreshPasswordsOnScreen()
			showFeedback(g, fmt.Sprintf("Password changed, but renaming to `%s` failed", name))
			return closeAddPassword(g, v)
		}
	}

	refreshPasswordsOnScreen()
	showFeedback(g, fmt.Sprintf("Password for `%s` changed!", name))

	return closeAddPassword(g, v)
}

func closeAddPassword(g *gocui.Gui, v *gocui.View) error {
	addFocus = 0
	otpMode = false
	resetChangeMode()
	popViewStack()
	return nil
}

func resetChangeMode() {
	changeMode = false
	changeTarget = ""
	changeOldPass = ""
	selectedRow = 0
}
