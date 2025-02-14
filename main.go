package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/FFX01/gettuit/internal/gotuit"
	"github.com/gdamore/tcell/v2"
)

var modeMap = map[gotuit.Mode]string{
	gotuit.NormalMode: "Normal",
	gotuit.InputMode:  "Input",
}

var (
	backgroundColor tcell.Color = tcell.NewHexColor(0x284B63)
	foregroundColor tcell.Color = tcell.NewHexColor(0xF4F9E9)
)

type Model struct {
	todos             []Todo
	helpModalViewName string
}

func (m *Model) loadFromDisk() error {
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
	for _, t := range data.Todos {
		nt := Todo{
			text:     t.Text,
			complete: t.Complete,
		}
		newTodos = append(newTodos, nt)
	}
	m.todos = newTodos

	return nil
}

func (m *Model) SaveToDisk() error {
	data := DataSchema{}
	for _, t := range m.todos {
		todoData := TodoDataSchema{
			Text:     t.text,
			Complete: t.complete,
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

func (m *Model) Init() {
	m.todos = []Todo{}
	m.loadFromDisk()
}

func (m *Model) renderTitle(v *gotuit.View) {
	text := " Todo List, 'Ctrl+c' to quit, press 'F1' for help "
	for idx, r := range text {
		v.SetContent(idx, 0, r, tcell.StyleDefault)
	}
}

func (m *Model) renderHelpModal(v *gotuit.View) {
	height := v.InnerHeight()

	for xidx, r := range "`Esc` to exit help" {
		v.SetContent(xidx, height, r, tcell.StyleDefault.Background(backgroundColor))
	}

	viewForHelp, ok := v.App.GetView(m.helpModalViewName)
	if !ok {
		log.Fatal("No Focused View")
	}

	description := fmt.Sprintf("Help for %s, %s mode", viewForHelp.Name, modeMap[viewForHelp.Mode])
	for xidx, r := range description {
		v.SetContent(xidx, 0, r, tcell.StyleDefault.Background(backgroundColor))
	}

	yidx := 0
	for _, kb := range viewForHelp.Keybinds {
		if kb.Mode() != viewForHelp.Mode {
			continue
		}
		text := kb.String()
		for xidx, r := range text {
			v.SetContent(xidx, yidx+2, r, tcell.StyleDefault.Background(backgroundColor))
		}
		yidx++
	}
}

func (m *Model) renderTodos(v *gotuit.View) {
	for idx, todo := range m.todos {
		style := tcell.StyleDefault
		prefix := "[ ]"
		if todo.complete {
			prefix = "[x]"
		} else if todo.temp {
			prefix = "#>"
		}

		if idx == v.Cursory {
			style = style.Background(tcell.ColorGray)
		}

		var text string
		if todo.temp {
			text = fmt.Sprintf("%s %s", prefix, string(v.GetInputBuffer()))
		} else {
			text = fmt.Sprintf("%s %s", prefix, todo.text)
		}

		for tidx, r := range text {
			v.SetContent(tidx, idx, r, style)
		}

		if v.Mode == gotuit.InputMode && todo.temp {
			v.Cursorx = len(prefix) + v.InputCursor + 1
			v.ShowCursor()
		}
	}
}

func (m *Model) renderStatusLine(v *gotuit.View) {
	style := tcell.StyleDefault.
		Background(backgroundColor).
		Foreground(foregroundColor)

	focusedView, err := v.App.GetFocusedView()
	var mode string
	if err != nil {
		log.Println("No Focused View!")
		mode = "Error"
	} else {
		mode = modeMap[focusedView.Mode]
	}

	statusText := " Mode: " + mode

	logLine := v.App.PopLog()
	if logLine != "" {
		statusText += ", Log: " + logLine
	}
	for xidx, r := range statusText {
		v.SetContent(xidx, 0, r, style)
	}
}

type DataSchema struct {
	Todos []TodoDataSchema `json:"todos"`
}

func (m *Model) onTodoListToggleComplete(v *gotuit.View) {
	m.todos[v.Cursory].complete = !m.todos[v.Cursory].complete
	err := m.SaveToDisk()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Toggle Todo")
}

func (m *Model) onTodoListJumpToTop(v *gotuit.View) {
	v.Cursory = 0
}

func (m *Model) onTodoListJumpToBottom(v *gotuit.View) {
	v.Cursory = len(m.todos) - 1
}

func (m *Model) onTodoListAddTodo(v *gotuit.View) {
	log.Println("Adding todo...")
	v.Mode = gotuit.InputMode
	t := Todo{temp: true}

	if len(m.todos) > 0 {
		m.todos = slices.Insert(m.todos, v.Cursory+1, t)
		v.Cursory++
	} else {
		m.todos = []Todo{t}
	}
}

func (m *Model) onTodoListInputEscape(v *gotuit.View) {
	todo := m.todos[v.Cursory]
	if todo.temp && todo.text == "" {
		m.todos = slices.Delete(m.todos, v.Cursory, v.Cursory+1)
	} else {
		todo.temp = false
		m.todos[v.Cursory] = todo
	}

	v.Mode = gotuit.NormalMode
	v.ClearInputBuffer()
	v.HideCursor()
}

func (m *Model) onTodoListEditTodo(v *gotuit.View) {
	v.Mode = gotuit.InputMode
	m.todos[v.Cursory].temp = true
	textAsRunes := []rune(m.todos[v.Cursory].text)
	v.SetInputBuffer(textAsRunes)
	v.InputCursor = len(textAsRunes)
}

func (m *Model) onTodoListReplaceTodo(v *gotuit.View) {
	v.Mode = gotuit.InputMode
	m.todos[v.Cursory].temp = true
	v.ClearInputBuffer()
}

func (m *Model) onTodoListDeleteTodo(v *gotuit.View) {
	newTodos := make([]Todo, 0)
	newTodos = append(newTodos, m.todos[:v.Cursory]...)
	newTodos = append(newTodos, m.todos[v.Cursory+1:]...)
	m.todos = newTodos

	if v.Cursory > 0 {
		v.Cursory--
	} else {
		v.Cursory = 0
	}
	m.SaveToDisk()
}

func (m *Model) onTodoListConfirmTodo(v *gotuit.View) {
	m.todos[v.Cursory].text = string(v.GetInputBuffer())
	m.todos[v.Cursory].temp = false
	v.Mode = gotuit.NormalMode
	v.Cursorx = 0
	v.HideCursor()
	v.ClearInputBuffer()
	m.SaveToDisk()
}

func (m *Model) onTodoListMoveTodoDown(v *gotuit.View) {
	if v.Cursory < len(m.todos)-1 {
		m.todos[v.Cursory], m.todos[v.Cursory+1] = m.todos[v.Cursory+1], m.todos[v.Cursory]
		v.Cursory++
		m.SaveToDisk()
	}
}

func (m *Model) onTodoListMoveTodoUp(v *gotuit.View) {
	if v.Cursory > 0 {
		m.todos[v.Cursory], m.todos[v.Cursory-1] = m.todos[v.Cursory-1], m.todos[v.Cursory]
		v.Cursory--
		m.SaveToDisk()
	}
}

func (m *Model) onTodoListInputBackspace(v *gotuit.View) {
	text := v.GetInputBuffer()
	log.Println("Cursor x: ", v.InputCursor)

	if v.InputCursor > 0 {
		head := text[:v.InputCursor-1]
		var tail []rune
		if v.InputCursor < len(text) {
			tail = text[v.InputCursor:]
		}
		text = append(head, tail...)
		v.SetInputBuffer(text)
		v.InputCursor--
	}
}

func (m *Model) onTodoListInputLeft(v *gotuit.View) {
	if v.InputCursor > 0 {
		v.InputCursor--
	}
}

func (m *Model) onTodoListInputRight(v *gotuit.View) {
	lineLen := len(v.GetInputBuffer())
	if v.InputCursor < lineLen {
		v.InputCursor++
	}
}

// func (app *App) DrawHelpModal() {
// 	width, height := app.screen.Size()
// 	modalWidth := 70
// 	marginX := (width - modalWidth) / 2
// 	modalHeight := 50
// 	marginY := (height - modalHeight) / 2
//
// 	style := tcell.StyleDefault.Background(backgroundColor)
//
// 	drawBox(app.screen, marginX, marginY, modalWidth, modalHeight, style)
//
// 	yIndex := marginY + 1
// 	xIndex := marginX + 4
//
// 	// Draw Normal mode header
// 	headerText := "Normal Mode"
// 	headerStartX := marginX + ((modalWidth - len(headerText)) / 2)
// 	drawText(app.screen, headerStartX, yIndex, style, headerText)
// 	yIndex += 2
//
// 	// Draw normal mode help
// 	helpTexts, exists := app.keybindHelpText[gotuit.NormalMode]
// 	if !exists {
// 		return
// 	}
// 	for _, t := range helpTexts {
// 		drawText(app.screen, xIndex, yIndex, style, t)
// 		yIndex++
// 	}
// 	yIndex += 2
//
// 	// Draw Input Mode Header
// 	headerText = "Input Mode"
// 	headerStartX = marginX + ((modalWidth - len(headerText)) / 2)
// 	drawText(app.screen, headerStartX, yIndex, style, headerText)
// 	yIndex += 2
//
// 	helpTexts, exists = app.keybindHelpText[gotuit.InputMode]
// 	if !exists {
// 		return
// 	}
// 	for _, t := range helpTexts {
// 		drawText(app.screen, xIndex, yIndex, style, t)
// 		yIndex++
// 	}
//
// 	bottomTextStartY := marginY + modalHeight - 2
// 	bottomTextStartX := marginX + 4
// 	bottomText := "'?' to view this window, `Esc` to exit this window"
// 	drawText(app.screen, bottomTextStartX, bottomTextStartY, style, bottomText)
// }

type Todo struct {
	text     string
	complete bool
	temp     bool
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

func (m *Model) onTodoListCursorDown(v *gotuit.View) {
	if v.Cursory < len(m.todos)-1 {
		v.Cursory++
	}
}

func (m *Model) onTodoListCursorUp(v *gotuit.View) {
	if v.Cursory > 0 {
		v.Cursory--
	}
}

func onGlobalQuit(app *gotuit.App) {
	app.Quit()
}

func (m *Model) onGlobalShowHelp(app *gotuit.App) {
	focusedView, err := app.GetFocusedView()
	if err != nil {
		m.helpModalViewName = "Todo List"
	} else {
		m.helpModalViewName = focusedView.Name
	}
	app.ShowView("Help Modal")
	app.Focus("Help Modal")
}

func onHelpExit(v *gotuit.View) {
	v.App.HideView("Help Modal")
	v.App.Focus("Todo List")
}

func main() {
	model := Model{}
	model.Init()
	app := gotuit.NewApp()
	defer app.Cleanup()

	log.SetOutput(app)

	width, height := app.Size()

	list := gotuit.NewView("Todo List", 0, 1, width, height-4, model.renderTodos)
	list.SetPadding(1, 1, 2, 1)
	list.Bind(gotuit.NormalMode, 'k', "Up", "Move cursor up", model.onTodoListCursorUp)
	list.Bind(gotuit.NormalMode, 'j', "Down", "Move cursor down", model.onTodoListCursorDown)
	list.Bind(gotuit.NormalMode, tcell.KeyCtrlK, "Move Up", "Move item up", model.onTodoListMoveTodoUp)
	list.Bind(gotuit.NormalMode, tcell.KeyCtrlJ, "Move Down", "Move item down", model.onTodoListMoveTodoDown)
	list.Bind(gotuit.NormalMode, ' ', "Toggle Complete", "Toggle completion status", model.onTodoListToggleComplete)
	list.Bind(gotuit.NormalMode, 'a', "[A]dd Todo", "Add a new todo", model.onTodoListAddTodo)
	list.Bind(gotuit.NormalMode, 'D', "[D]elete Todo", "Delete todo on cursor", model.onTodoListDeleteTodo)
	list.Bind(gotuit.NormalMode, 'e', "[E]dit Todo", "Edit todo on cursor", model.onTodoListEditTodo)
	list.Bind(gotuit.NormalMode, 'r', "[R]eplace Todo", "Replace todo with a new one", model.onTodoListReplaceTodo)
	list.Bind(gotuit.NormalMode, tcell.KeyCtrlU, "Jump to top", "Jump to to top of list", model.onTodoListJumpToTop)
	list.Bind(gotuit.NormalMode, tcell.KeyCtrlD, "Jump to bottom", "Jump to to bottom of list", model.onTodoListJumpToBottom)
	list.Bind(gotuit.InputMode, tcell.KeyEnter, "Confirm", "Confirm changes", model.onTodoListConfirmTodo)
	list.Bind(gotuit.InputMode, tcell.KeyBackspace, "Backspace", "Backspace", model.onTodoListInputBackspace)
	list.Bind(gotuit.InputMode, tcell.KeyBackspace2, "Backspace", "Backspace", model.onTodoListInputBackspace)
	list.Bind(gotuit.InputMode, tcell.KeyLeft, "Left", "Move cursor left", model.onTodoListInputLeft)
	list.Bind(gotuit.InputMode, tcell.KeyRight, "Right", "Move cursor right", model.onTodoListInputRight)
	list.Bind(gotuit.InputMode, tcell.KeyEscape, "Exit", "Cancel Changes", model.onTodoListInputEscape)

	title := gotuit.NewView("Title", 0, 0, width, 1, model.renderTitle)

	statusLine := gotuit.NewView("Status Line", 0, height-3, width, 3, model.renderStatusLine)
	statusLine.SetFillColor(backgroundColor)

	helpModal := gotuit.NewView("Help Modal", width/4, height/4, width/2, height/2, model.renderHelpModal)
	helpModal.SetPadding(0, 1, 0, 1)
	helpModal.SetFillColor(backgroundColor)
	helpModal.Hide()
	helpModal.Bind(gotuit.NormalMode, tcell.KeyEscape, "Exit", "Exit Help", onHelpExit)

	app.AddView(title)
	app.AddView(list)
	app.AddView(statusLine)
	app.AddView(helpModal)

	app.Focus("Todo List")

	app.Bind(tcell.KeyCtrlC, "Quit", "Quit program", onGlobalQuit)
	app.Bind(tcell.KeyF1, "Help", "Show Help", model.onGlobalShowHelp)

	app.MainLoop()
}
