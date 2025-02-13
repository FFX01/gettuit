package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"slices"

	"github.com/gdamore/tcell/v2"
)

type mode int

const (
	normalMode mode = iota
	inputMode
	helpMode
	searchMode
)

var modeMap = map[mode]string{
	normalMode: "Normal",
	inputMode:  "Input",
	helpMode:   "Help",
	searchMode: "Search",
}

var (
	backgroundColor tcell.Color = tcell.NewHexColor(0x284B63)
	foregroundColor tcell.Color = tcell.NewHexColor(0xF4F9E9)
)

type Keybind struct {
	name        string
	description string
	callback    func(*tcell.EventKey)
}

type State struct {
	todos []Todo
}

func NewState() *State {
	s := State{
		todos: []Todo{},
	}
	return &s
}

type App struct {
	screen          tcell.Screen
	state           *State
	todos           []Todo
	cursorY         int
	cursorX         int
	mode            mode
	helpMode        mode
	quit            bool
	keybinds        map[mode]map[tcell.Key]Keybind
	keybindHelpText map[mode][]string
	focusedView     string
	views           []*View
}

func (app *App) AddView(v *View) {
	v.app = app
	app.views = append(app.views, v)
}

func (app *App) Focus(viewName string) {
	app.focusedView = viewName
}

func (app *App) GetFocusedView() (*View, error) {
	for _, v := range app.views {
		if v.name == app.focusedView {
			return v, nil
		}
	}
	return nil, errors.New("View not found")
}

type Keybind2 struct {
	name        string
	description string
	key         tcell.Key
	mode        mode
	callback    func(*View, *State)
}

type cell struct {
	x, y  int
	char  rune
	style tcell.Style
}

type View struct {
	name             string
	app              *App
	mode             mode
	keybinds         []Keybind2
	x, y             int
	w, h             int
	cells            []cell
	renderFunc       func(*View, *State)
	cursorx, cursory int
	border           bool
	paddingt         int
	paddingr         int
	paddingb         int
	paddingl         int
}

func (v *View) SetPadding(t, r, b, l int) {
    v.paddingt = t
    v.paddingr = r
    v.paddingb = b
    v.paddingl = l
}

func (v *View) getKeybind(m mode, key tcell.Key) (Keybind2, error) {
	for _, kb := range v.keybinds {
		if kb.mode == m && kb.key == key {
			return kb, errors.New("Keybind does not exist")
		}
	}
	return Keybind2{}, nil
}

func (v *View) handleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		var key tcell.Key
		if ev.Key() == tcell.KeyRune {
			key = tcell.Key(ev.Rune())
		} else {
			key = ev.Key()
		}
		kb, err := v.getKeybind(v.mode, key)
		if err != nil {
			kb.callback(v, v.app.state)
		}
	}
}

func (v *View) Bind(mode mode, key tcell.Key, name, description string, cb func(*View, *State)) {
	kb := Keybind2{
		name:        name,
		description: description,
		key:         key,
		callback:    cb,
	}
	v.keybinds = append(v.keybinds, kb)
}

func NewView(name string, x, y, w, h int, renderFunc func(*View, *State)) *View {
	v := View{
		name:       name,
		mode:       normalMode,
		x:          x,
		y:          y,
		w:          w,
		h:          h,
		cells:      make([]cell, 0),
		renderFunc: renderFunc,
		border:     true,
	}

	return &v
}

func (v *View) getInnerBounds() (x1, y1, x2, y2 int) {
	x1 = v.x + 1 + v.paddingl
	y1 = v.y + 1 + v.paddingt
	x2 = v.x + v.w - 3 - v.paddingr
	y2 = v.y + v.h - 3 - v.paddingb
	return x1, y1, x2, y2
}

func (v *View) getOuterBounds() (x1, y1, x2, y2 int) {
	x1 = v.x
	y1 = v.y
	x2 = v.x + v.w - 2
	y2 = v.y + v.h - 2
	return x1, y1, x2, y2
}

func (v *View) SetContent(x, y int, r rune, style tcell.Style) {
	if x < 0 || x > v.w-1 {
		return
	}
	if y < 0 || y > v.h-1 {
		return
	}

	v.cells = append(v.cells, cell{x: x, y: y, char: r, style: style})
}

func (v *View) Clear() {
    v.cells = []cell{}
}

func (v *View) Draw(screen tcell.Screen) {
	x1, y1, _, _ := v.getInnerBounds()

	if v.border && v.h > 2 {
		bx1, by1, bx2, by2 := v.getOuterBounds()

		screen.SetContent(bx1, by1, tcell.RuneULCorner, nil, tcell.StyleDefault)
		screen.SetContent(bx2, by1, tcell.RuneURCorner, nil, tcell.StyleDefault)
		screen.SetContent(bx1, by2, tcell.RuneLLCorner, nil, tcell.StyleDefault)
		screen.SetContent(bx2, by2, tcell.RuneLRCorner, nil, tcell.StyleDefault)

		for xidx := bx1 + 1; xidx < bx2; xidx++ {
			screen.SetContent(xidx, by1, tcell.RuneHLine, nil, tcell.StyleDefault)
			screen.SetContent(xidx, by2, tcell.RuneHLine, nil, tcell.StyleDefault)
		}
		for yidx := by1 + 1; yidx < by2; yidx++ {
			screen.SetContent(bx1, yidx, tcell.RuneVLine, nil, tcell.StyleDefault)
			screen.SetContent(bx2, yidx, tcell.RuneVLine, nil, tcell.StyleDefault)
		}
	}

	for _, cell := range v.cells {
		x := x1 + cell.x
		y := y1 + cell.y
		screen.SetContent(x, y, cell.char, nil, cell.style)
	}
}

func renderTitle(v *View, state *State) {
	text := " Todo List, 'Ctrl+c' to quit, press '?' for help "
	for idx, r := range text {
		v.SetContent(idx, 0, r, tcell.StyleDefault)
	}
}

func renderTodos(v *View, state *State) {
	for idx, todo := range state.todos {
		style := tcell.StyleDefault
		prefix := "[ ]"
		if todo.complete {
			prefix = "[x]"
		} else if todo.temp {
			prefix = "#>"
		}

		if idx == v.cursory {
			style = style.Background(tcell.ColorGray)
		}

		var text string
		if todo.temp {
			text = fmt.Sprintf("%s %s", prefix, todo.tempText)
		} else {
			text = fmt.Sprintf("%s %s", prefix, todo.text)
		}

		for tidx, r := range text {
			v.SetContent(tidx, idx, r, style)
		}

		// if app.mode == inputMode && todo.temp {
		// 	app.screen.ShowCursor(app.cursorX+len(prefix)+3, app.cursorY+2)
		// }
	}
}

func renderStatusLine(v *View, _ *State) {
	style := tcell.StyleDefault.
		Background(backgroundColor).
		Foreground(foregroundColor)

	for xidx := 0; xidx < v.w; xidx++ {
		v.SetContent(xidx, 0, ' ', style)
	}

	statusText := " Mode: " + modeMap[v.mode]
	for xidx, r := range statusText {
		v.SetContent(xidx, 0, r, style)
	}
}

type DataSchema struct {
	Todos []TodoDataSchema `json:"todos"`
}

func NewApp() *App {
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatal("Unable to draw screen", "error", err)
		os.Exit(1)
	}

	err = screen.Init()
	if err != nil {
		log.Fatal("Unable to initialize screen", "error", err)
		os.Exit(1)
	}

	state := NewState()

	app := App{
		screen: screen,
		state:  state,
		todos: []Todo{
			{text: "Example todo 1"},
			{text: "Example todo 2"},
			{text: "Example todo 3"},
		},
		mode:            normalMode,
		helpMode:        normalMode,
		keybinds:        make(map[mode]map[tcell.Key]Keybind),
		keybindHelpText: make(map[mode][]string),
	}

	err = app.loadFromDisk()
	if err != nil {
		slog.Warn("Unable to load from disk. File may not exist", "error", err)
	}

	return &app
}

func (app *App) Bind(m mode, key tcell.Key, name, description string, showInHelp bool, callback func(*tcell.EventKey)) {
	kb := Keybind{
		name:        name,
		description: description,
		callback:    callback,
	}

	_, exists := app.keybinds[m]
	if !exists {
		app.keybinds[m] = make(map[tcell.Key]Keybind)
	}
	app.keybinds[m][key] = kb

	if !showInHelp {
		return
	}

	ekey := tcell.NewEventKey(key, rune(key), tcell.ModNone)
	var keyText string
	if ekey.Name()[:3] == "Key" {
		keyText = string(key)
	} else {
		keyText = ekey.Name()
	}

	if keyText == " " {
		keyText = "<space>"
	}

	_, exists = app.keybindHelpText[m]
	if !exists {
		app.keybindHelpText[m] = []string{}
	}
	text := fmt.Sprintf("%s - %s [%s]", keyText, name, description)
	app.keybindHelpText[m] = append(app.keybindHelpText[m], text)
}

func (app *App) Quit(_ *tcell.EventKey) {
	app.quit = true
}

func (app *App) loadFromDisk() error {
	file, err := os.Open("todoData.json")
	if err != nil {
		return err
	}

	data := DataSchema{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}

	newTodos := []Todo{}
	for _, td := range data.Todos {
		t := Todo{
			text:     td.Text,
			complete: td.Complete,
			temp:     td.Temp,
		}
		newTodos = append(newTodos, t)
	}
	app.todos = newTodos
	app.state.todos = newTodos

	return nil
}

func (app *App) SaveToDisk() error {
	data := DataSchema{}

	for _, t := range app.todos {
		todoData := TodoDataSchema{
			Text:     t.text,
			Complete: t.complete,
			Temp:     t.temp,
		}
		data.Todos = append(data.Todos, todoData)
	}
	marshalledData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = os.WriteFile("todoData.json", marshalledData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (app *App) markTodoComplete(_ *tcell.EventKey) {
	app.todos[app.cursorY].complete = !app.todos[app.cursorY].complete
	err := app.SaveToDisk()
	if err != nil {
		panic("Could not write file")
	}
}

func (app *App) onJumpToTop(_ *tcell.EventKey) {
	app.cursorY = 0
}

func (app *App) onJumpToBottom(_ *tcell.EventKey) {
	app.cursorY = len(app.todos) - 1
}

func (app *App) onAddTodo(_ *tcell.EventKey) {
	app.mode = inputMode
	app.helpMode = inputMode
	t := Todo{temp: true}
	if len(app.todos) > 0 {
		app.todos = slices.Insert(app.todos, app.cursorY+1, t)
		app.cursorY++
	} else {
		app.todos = []Todo{t}
	}
}

func (app *App) onInputEscape(_ *tcell.EventKey) {
	todo := app.todos[app.cursorY]

	if todo.temp && todo.text == "" {
		// handle cancel add
		app.todos = slices.Delete(app.todos, app.cursorY, app.cursorY+1)
	} else {
		// handle cancel edit
		todo.tempText = ""
		todo.temp = false
		app.todos[app.cursorY] = todo
	}

	app.mode = normalMode
	app.helpMode = normalMode
	app.cursorX = 0
	app.screen.HideCursor()
}

func (app *App) onEditTodo(_ *tcell.EventKey) {
	app.mode = inputMode
	app.helpMode = inputMode
	app.todos[app.cursorY].temp = true
	app.todos[app.cursorY].tempText = app.todos[app.cursorY].text
	app.cursorX = len(app.todos[app.cursorY].text)
}

func (app *App) onReplaceTodo(_ *tcell.EventKey) {
	app.mode = inputMode
	app.helpMode = inputMode
	app.todos[app.cursorY].temp = true
	app.todos[app.cursorY].tempText = ""
}

func (app *App) onActivateHelpMode(_ *tcell.EventKey) {
	app.mode = helpMode
}

func (app *App) onExitHelpMode(_ *tcell.EventKey) {
	app.mode = normalMode
	app.helpMode = normalMode
}

func (app *App) onDeleteTodo(_ *tcell.EventKey) {
	newTodos := make([]Todo, 0)
	newTodos = append(newTodos, app.todos[:app.cursorY]...)
	newTodos = append(newTodos, app.todos[app.cursorY+1:]...)
	app.todos = newTodos
	if app.cursorY > 0 {
		app.cursorY--
	} else {
		app.cursorY = 0
	}
	app.SaveToDisk()
}

func (app *App) onConfirmTodo(_ *tcell.EventKey) {
	app.todos[app.cursorY].text = app.todos[app.cursorY].tempText
	app.todos[app.cursorY].tempText = ""
	app.todos[app.cursorY].temp = false
	app.mode = normalMode
	app.helpMode = normalMode
	app.cursorX = 0
	app.screen.HideCursor()
	app.SaveToDisk()
}

func (app *App) onMoveTodoUp(_ *tcell.EventKey) {
	if app.cursorY > 0 {
		app.todos[app.cursorY], app.todos[app.cursorY-1] = app.todos[app.cursorY-1], app.todos[app.cursorY]
		app.cursorY--
		app.SaveToDisk()
	}
}

func (app *App) onMoveTodoDown(_ *tcell.EventKey) {
	if app.cursorY < len(app.todos)-1 {
		app.todos[app.cursorY], app.todos[app.cursorY+1] = app.todos[app.cursorY+1], app.todos[app.cursorY]
		app.cursorY++
		app.SaveToDisk()
	}
}

func onTodoListMoveTodoDown(v *View, s *State) {
    if v.cursory < len(s.todos) - 1 {
        s.todos[v.cursory], s.todos[v.cursory+1] = s.todos[v.cursory+1], s.todos[v.cursory]
        v.cursory++
    }
}

func (app *App) onMoveCursorUp(_ *tcell.EventKey) {
	if app.cursorY > 0 {
		app.cursorY--
	}
}

func (app *App) onMoveCursorDown(_ *tcell.EventKey) {
	if app.cursorY < len(app.todos)-1 {
		app.cursorY++
	}
}

func (app *App) onInputBackspace(_ *tcell.EventKey) {
	todo := app.todos[app.cursorY]
	if app.cursorX > 0 {
		head := todo.tempText[:app.cursorX-1]
		var tail string
		if app.cursorX < len(todo.tempText) {
			tail = todo.tempText[app.cursorX:]
		}
		todo.tempText = head + tail
		app.cursorX--
		app.todos[app.cursorY] = todo
	}
}

func (app *App) onInputCursorLeft(_ *tcell.EventKey) {
	if app.cursorX > 0 {
		app.cursorX--
	}
}

func (app *App) onInputCursorRight(_ *tcell.EventKey) {
	lineLen := len(app.todos[app.cursorY].tempText)
	if app.cursorX < lineLen {
		app.cursorX++
	}
}

func (app *App) onInputRune(ev *tcell.EventKey) {
	if ev.Rune() != 0 {
		if app.cursorX == len(app.todos[app.cursorY].tempText) {
			app.todos[app.cursorY].tempText += string(ev.Rune())
		} else {
			head := app.todos[app.cursorY].tempText[:app.cursorX]
			tail := app.todos[app.cursorY].tempText[app.cursorX:]
			newText := head + string(ev.Rune()) + tail
			app.todos[app.cursorY].tempText = newText
		}
		app.cursorX++
	}
}

func (app *App) handleEvent(ev tcell.Event) {
	focusedView, err := app.GetFocusedView()
	if err == nil {
		focusedView.handleEvent(ev)
		return
	}

	switch ev := ev.(type) {
	case *tcell.EventKey:
		modeMap, exists := app.keybinds[app.mode]
		if !exists {
			return
		}
		var (
			kb Keybind
		)

		if ev.Key() == tcell.KeyRune && app.mode != inputMode {
			kb, exists = modeMap[tcell.Key(ev.Rune())]
		} else {
			kb, exists = modeMap[ev.Key()]
		}

		if !exists {
			return
		}

		kb.callback(ev)
	}
}

func (app *App) DrawStatus() {
	width, height := app.screen.Size()

	style := tcell.StyleDefault.
		Background(backgroundColor).
		Foreground(foregroundColor)

	drawLine(app.screen, 2, height-2, width-3, style)

	statusText := " Mode: " + modeMap[app.mode]
	drawText(app.screen, 1, height-2, style, statusText)
}

func (app *App) DrawHelpModal() {
	width, height := app.screen.Size()
	modalWidth := 70
	marginX := (width - modalWidth) / 2
	modalHeight := 50
	marginY := (height - modalHeight) / 2

	style := tcell.StyleDefault.Background(backgroundColor)

	drawBox(app.screen, marginX, marginY, modalWidth, modalHeight, style)

	yIndex := marginY + 1
	xIndex := marginX + 4

	// Draw Normal mode header
	headerText := "Normal Mode"
	headerStartX := marginX + ((modalWidth - len(headerText)) / 2)
	drawText(app.screen, headerStartX, yIndex, style, headerText)
	yIndex += 2

	// Draw normal mode help
	helpTexts, exists := app.keybindHelpText[normalMode]
	if !exists {
		return
	}
	for _, t := range helpTexts {
		drawText(app.screen, xIndex, yIndex, style, t)
		yIndex++
	}
	yIndex += 2

	// Draw Input Mode Header
	headerText = "Input Mode"
	headerStartX = marginX + ((modalWidth - len(headerText)) / 2)
	drawText(app.screen, headerStartX, yIndex, style, headerText)
	yIndex += 2

	helpTexts, exists = app.keybindHelpText[inputMode]
	if !exists {
		return
	}
	for _, t := range helpTexts {
		drawText(app.screen, xIndex, yIndex, style, t)
		yIndex++
	}

	bottomTextStartY := marginY + modalHeight - 2
	bottomTextStartX := marginX + 4
	bottomText := "'?' to view this window, `Esc` to exit this window"
	drawText(app.screen, bottomTextStartX, bottomTextStartY, style, bottomText)
}

func (app *App) Draw() {
	app.screen.Clear()
	width, height := app.screen.Size()
	appStyle := tcell.StyleDefault
	drawBox(app.screen, 0, 0, width-1, height-1, appStyle)

	drawText(app.screen, 1, 0, tcell.StyleDefault, " Todo List, 'Ctrl+c' to quit, press '?' for help ")

	v := NewView("Todo List", 1, 1, width, height-2, renderTodos)
	app.AddView(v)
	app.Focus("Todo List")

	for idx, todo := range app.todos {
		style := tcell.StyleDefault
		prefix := "[ ]"
		if todo.complete {
			prefix = "[x]"
		} else if todo.temp {
			prefix = "#>"
		}

		if idx == app.cursorY {
			style = style.Background(tcell.ColorGray)
		}

		var text string
		if todo.temp {
			text = fmt.Sprintf("%s %s", prefix, todo.tempText)
		} else {
			text = fmt.Sprintf("%s %s", prefix, todo.text)
		}
		// drawText(app.screen, 2, idx+2, style, text)
		for tidx, r := range text {
			v.SetContent(tidx, idx, r, style)
		}

		if app.mode == inputMode && todo.temp {
			app.screen.ShowCursor(app.cursorX+len(prefix)+3, app.cursorY+2)
		}
	}

	v.Draw(app.screen)

	app.DrawStatus()

	if app.mode == helpMode {
		app.DrawHelpModal()
	}

	app.screen.Show()
}

func (app *App) Draw2() {
	app.screen.Clear()
	for _, v := range app.views {
        v.Clear()
		v.renderFunc(v, app.state)
		v.Draw(app.screen)
	}

	app.screen.Show()
}

type Todo struct {
	text     string
	complete bool
	temp     bool
	tempText string
}

type TodoDataSchema struct {
	Text     string `json:"text"`
	Complete bool   `json:"complete"`
	Temp     bool   `json:"temp"`
}

func drawText(screen tcell.Screen, x, y int, style tcell.Style, text string) {
	for idx, r := range text {
		screen.SetContent(x+idx, y, r, nil, style)
	}
}

func drawBox(screen tcell.Screen, x, y, width, height int, style tcell.Style) {
	x2 := x + width
	y2 := y + height
	for yidx := y; yidx < y2; yidx++ {
		for xidx := x; xidx < x2; xidx++ {
			screen.SetContent(xidx, yidx, ' ', nil, style)
		}
	}

	for col := x; col < x2; col++ {
		screen.SetContent(col, y, tcell.RuneHLine, nil, style)
		screen.SetContent(col, y+height, tcell.RuneHLine, nil, style)
	}
	for row := y; row < y+height; row++ {
		screen.SetContent(x, row, tcell.RuneVLine, nil, style)
		screen.SetContent(x+width, row, tcell.RuneVLine, nil, style)
	}

	screen.SetContent(x, y, tcell.RuneULCorner, nil, style)
	screen.SetContent(x+width, y, tcell.RuneURCorner, nil, style)
	screen.SetContent(x, y+height, tcell.RuneLLCorner, nil, style)
	screen.SetContent(x+width, y+height, tcell.RuneLRCorner, nil, style)
}

func drawLine(screen tcell.Screen, x, y, width int, style tcell.Style) {
	x2 := x + width

	for xidx := x; xidx < x2; xidx++ {
		screen.SetContent(xidx, y, ' ', nil, style)
	}
}

func onTodoListCursorDown(v *View, s *State) {
	if v.cursory < len(s.todos)-1 {
		v.cursory++
	}
}

func onTodoListQuit(v *View, s *State) {
	v.app.quit = true
}

func main() {
	app := NewApp()
	defer app.screen.Fini()

	// Normal mode bindings
	app.Bind(normalMode, tcell.KeyCtrlC, "Quit", "Quit program", true, app.Quit)
	app.Bind(normalMode, 'k', "Up", "Move cursor up", true, app.onMoveCursorUp)
	app.Bind(normalMode, 'j', "Down", "Move cursor down", true, app.onMoveCursorDown)
	app.Bind(normalMode, tcell.KeyCtrlK, "Todo Up", "Move a todo Up", true, app.onMoveTodoUp)
	app.Bind(normalMode, tcell.KeyCtrlJ, "Todo Down", "Move a todo down", true, app.onMoveTodoDown)
	app.Bind(normalMode, 'a', "[A]dd Todo", "Add a new todo", true, app.onAddTodo)
	app.Bind(normalMode, 'e', "[E]dit Todo", "Edit a todo", true, app.onEditTodo)
	app.Bind(normalMode, 'r', "[R]eplace Todo", "Replace a todo", true, app.onReplaceTodo)
	app.Bind(normalMode, 'D', "[D]elete todo", "Delete a todo", true, app.onDeleteTodo)
	app.Bind(normalMode, '?', "Help", "Show help modal", true, app.onActivateHelpMode)
	app.Bind(normalMode, ' ', "Toggle Complete", "Toggle completion status", true, app.markTodoComplete)
	app.Bind(normalMode, tcell.KeyCtrlU, "Jump to top", "Jump to the top of the list", true, app.onJumpToTop)
	app.Bind(normalMode, tcell.KeyCtrlD, "Jump to bottom", "Jump to the bottom of the list", true, app.onJumpToBottom)

	// Input mode bindings
	app.Bind(inputMode, tcell.KeyCtrlC, "Quit", "Quit program", true, app.Quit)
	app.Bind(inputMode, tcell.KeyEnter, "Confirm", "Confirm changes", true, app.onConfirmTodo)
	app.Bind(inputMode, tcell.KeyBackspace, "Backspace", "Remove character before cursor", true, app.onInputBackspace)
	app.Bind(inputMode, tcell.KeyBackspace2, "Backspace", "Remove character before cursor", false, app.onInputBackspace)
	app.Bind(inputMode, tcell.KeyEscape, "Exit", "Exit input mode", true, app.onInputEscape)
	app.Bind(inputMode, tcell.KeyLeft, "Left", "Move cursor left", true, app.onInputCursorLeft)
	app.Bind(inputMode, tcell.KeyRight, "Right", "Move cursor right", true, app.onInputCursorRight)
	app.Bind(inputMode, tcell.KeyRune, "Text", "Enter text", false, app.onInputRune)

	// Help mode bindings
	app.Bind(helpMode, tcell.KeyCtrlC, "Quit", "Quit program", false, app.Quit)
	app.Bind(helpMode, tcell.KeyEscape, "Close", "Exit help mode", false, app.onExitHelpMode)

	width, height := app.screen.Size()

	list := NewView("Todo List", 1, 2, width, height-2, renderTodos)
    list.SetPadding(1, 1, 2, 1)
	list.Bind(normalMode, 'j', "Down", "Move cursor down", onTodoListCursorDown)
	list.Bind(normalMode, tcell.KeyCtrlC, "Quit", "Quit program", onTodoListQuit)
	list.Bind(normalMode, tcell.KeyCtrlJ, "Move Down", "Move item down", onTodoListMoveTodoDown)

	title := NewView("Title", 0, 0, width, 1, renderTitle)

	statusLine := NewView("Status Line", 0, height-2, width, 1, renderStatusLine)

	app.AddView(title)
	app.AddView(list)
	app.AddView(statusLine)

	app.Focus("Todo List")

	for !app.quit {
		// app.Draw()
		app.Draw2()
		app.handleEvent(app.screen.PollEvent())
	}
}
