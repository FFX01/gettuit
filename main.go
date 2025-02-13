package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"

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
	todos []Todo
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
	text := " Todo List, 'Ctrl+c' to quit, press '?' for help "
	for idx, r := range text {
		v.SetContent(idx, 0, r, tcell.StyleDefault)
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
			v.Cursorx = len(text)
			v.ShowCursor()
		}
	}
}

func (m *Model) renderStatusLine(v *gotuit.View) {
	style := tcell.StyleDefault.
		Background(backgroundColor).
		Foreground(foregroundColor)

	width := v.InnerWidth()

	for xidx := 0; xidx < width; xidx++ {
		v.SetContent(xidx, 0, '#', style)
	}

	widthStr := strconv.Itoa(width)
	statusText := " Mode: " + modeMap[v.Mode] + ", Width: " + widthStr
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
}

// func (app *App) onJumpToTop(_ *tcell.EventKey) {
// 	app.cursorY = 0
// }

// func (app *App) onJumpToBottom(_ *tcell.EventKey) {
// 	app.cursorY = len(app.todos) - 1
// }

func (m *Model) onTodoListAddTodo(v *gotuit.View) {
	v.Mode = gotuit.InputMode
	t := Todo{temp: true}

	if len(m.todos) > 0 {
		m.todos = slices.Insert(m.todos, v.Cursory+1, t)
		v.Cursory++
	} else {
		m.todos = []Todo{t}
	}
}

// func (app *App) onInputEscape(_ *tcell.EventKey) {
// 	todo := app.todos[app.cursorY]
//
// 	if todo.temp && todo.text == "" {
// 		// handle cancel add
// 		app.todos = slices.Delete(app.todos, app.cursorY, app.cursorY+1)
// 	} else {
// 		// handle cancel edit
// 		todo.tempText = ""
// 		todo.temp = false
// 		app.todos[app.cursorY] = todo
// 	}
//
// 	app.mode = gotuit.NormalMode
// 	app.helpMode = gotuit.NormalMode
// 	app.cursorX = 0
// 	app.screen.HideCursor()
// }

// func (app *App) onEditTodo(_ *tcell.EventKey) {
// 	app.mode = gotuit.InputMode
// 	app.helpMode = gotuit.InputMode
// 	app.todos[app.cursorY].temp = true
// 	app.todos[app.cursorY].tempText = app.todos[app.cursorY].text
// 	app.cursorX = len(app.todos[app.cursorY].text)
// }

// func (app *App) onReplaceTodo(_ *tcell.EventKey) {
// 	app.mode = gotuit.InputMode
// 	app.helpMode = gotuit.InputMode
// 	app.todos[app.cursorY].temp = true
// 	app.todos[app.cursorY].tempText = ""
// }

// func (app *App) onExitHelpMode(_ *tcell.EventKey) {
// 	app.mode = gotuit.NormalMode
// 	app.helpMode = gotuit.NormalMode
// }

// func (app *App) onDeleteTodo(_ *tcell.EventKey) {
// 	newTodos := make([]Todo, 0)
// 	newTodos = append(newTodos, app.todos[:app.cursorY]...)
// 	newTodos = append(newTodos, app.todos[app.cursorY+1:]...)
// 	app.todos = newTodos
// 	if app.cursorY > 0 {
// 		app.cursorY--
// 	} else {
// 		app.cursorY = 0
// 	}
// 	app.SaveToDisk()
// }

func (m *Model) onTodoListConfirmTodo(v *gotuit.View) {
    m.todos[v.Cursory].text = string(v.GetInputBuffer())
    m.todos[v.Cursory].temp = false
    v.Mode = gotuit.NormalMode
    v.Cursorx = 0
    v.HideCursor()
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

// func (app *App) onInputBackspace(_ *tcell.EventKey) {
// 	todo := app.todos[app.cursorY]
// 	if app.cursorX > 0 {
// 		head := todo.tempText[:app.cursorX-1]
// 		var tail string
// 		if app.cursorX < len(todo.tempText) {
// 			tail = todo.tempText[app.cursorX:]
// 		}
// 		todo.tempText = head + tail
// 		app.cursorX--
// 		app.todos[app.cursorY] = todo
// 	}
// }

// func (app *App) onInputCursorLeft(_ *tcell.EventKey) {
// 	if app.cursorX > 0 {
// 		app.cursorX--
// 	}
// }

// func (app *App) onInputCursorRight(_ *tcell.EventKey) {
// 	lineLen := len(app.todos[app.cursorY].tempText)
// 	if app.cursorX < lineLen {
// 		app.cursorX++
// 	}
// }

// func (app *App) onInputRune(ev *tcell.EventKey) {
// 	if ev.Rune() != 0 {
// 		if app.cursorX == len(app.todos[app.cursorY].tempText) {
// 			app.todos[app.cursorY].tempText += string(ev.Rune())
// 		} else {
// 			head := app.todos[app.cursorY].tempText[:app.cursorX]
// 			tail := app.todos[app.cursorY].tempText[app.cursorX:]
// 			newText := head + string(ev.Rune()) + tail
// 			app.todos[app.cursorY].tempText = newText
// 		}
// 		app.cursorX++
// 	}
// }

// func (app *App) DrawStatus() {
// 	width, height := app.screen.Size()
//
// 	style := tcell.StyleDefault.
// 		Background(backgroundColor).
// 		Foreground(foregroundColor)
//
// 	drawLine(app.screen, 2, height-2, width-3, style)
//
// 	statusText := " Mode: " + modeMap[app.mode]
// 	drawText(app.screen, 1, height-2, style, statusText)
// }

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

func onTodoListQuit(v *gotuit.View) {
	v.App.Quit()
}

func main() {
	model := Model{}
	model.Init()
	app := gotuit.NewApp()
	defer app.Cleanup()

	// Normal mode bindings
	// app.Bind(normalMode, 'e', "[E]dit Todo", "Edit a todo", true, app.onEditTodo)
	// app.Bind(normalMode, 'r', "[R]eplace Todo", "Replace a todo", true, app.onReplaceTodo)
	// app.Bind(normalMode, 'D', "[D]elete todo", "Delete a todo", true, app.onDeleteTodo)
	// app.Bind(normalMode, '?', "Help", "Show help modal", true, app.onActivateHelpMode)
	// app.Bind(normalMode, tcell.KeyCtrlU, "Jump to top", "Jump to the top of the list", true, app.onJumpToTop)
	// app.Bind(normalMode, tcell.KeyCtrlD, "Jump to bottom", "Jump to the bottom of the list", true, app.onJumpToBottom)

	// Input mode bindings
	// app.Bind(inputMode, tcell.KeyCtrlC, "Quit", "Quit program", true, app.Quit)
	// app.Bind(inputMode, tcell.KeyEnter, "Confirm", "Confirm changes", true, app.onConfirmTodo)
	// app.Bind(inputMode, tcell.KeyBackspace, "Backspace", "Remove character before cursor", true, app.onInputBackspace)
	// app.Bind(inputMode, tcell.KeyBackspace2, "Backspace", "Remove character before cursor", false, app.onInputBackspace)
	// app.Bind(inputMode, tcell.KeyEscape, "Exit", "Exit input mode", true, app.onInputEscape)
	// app.Bind(inputMode, tcell.KeyLeft, "Left", "Move cursor left", true, app.onInputCursorLeft)
	// app.Bind(inputMode, tcell.KeyRight, "Right", "Move cursor right", true, app.onInputCursorRight)

	// Help mode bindings
	// app.Bind(helpMode, tcell.KeyCtrlC, "Quit", "Quit program", false, app.Quit)
	// app.Bind(helpMode, tcell.KeyEscape, "Close", "Exit help mode", false, app.onExitHelpMode)

	width, height := app.Size()

	list := gotuit.NewView("Todo List", 0, 2, width, height-3, model.renderTodos)
	list.SetPadding(1, 1, 2, 1)
	list.Bind(gotuit.NormalMode, 'k', "Up", "Move cursor up", model.onTodoListCursorUp)
	list.Bind(gotuit.NormalMode, 'j', "Down", "Move cursor down", model.onTodoListCursorDown)
	list.Bind(gotuit.NormalMode, tcell.KeyCtrlC, "Quit", "Quit program", onTodoListQuit)
	list.Bind(gotuit.InputMode, tcell.KeyCtrlC, "Quit", "Quit program", onTodoListQuit)
	list.Bind(gotuit.NormalMode, tcell.KeyCtrlK, "Move Up", "Move item up", model.onTodoListMoveTodoUp)
	list.Bind(gotuit.NormalMode, tcell.KeyCtrlJ, "Move Down", "Move item down", model.onTodoListMoveTodoDown)
	list.Bind(gotuit.NormalMode, ' ', "Toggle Complete", "Toggle completion status", model.onTodoListToggleComplete)
	list.Bind(gotuit.NormalMode, 'a', "[A]dd Todo", "Add a new todo", model.onTodoListAddTodo)
    list.Bind(gotuit.InputMode, tcell.KeyEnter, "Confirm", "Confirm changes", model.onTodoListConfirmTodo)

	title := gotuit.NewView("Title", 0, 0, width, 1, model.renderTitle)

	statusLine := gotuit.NewView("Status Line", 0, height-2, width, 1, model.renderStatusLine)

	app.AddView(title)
	app.AddView(list)
	app.AddView(statusLine)

	app.Focus("Todo List")

	app.MainLoop()
}
