package gotuit

import (
	"errors"
	"log"
	"log/slog"
	"os"

	"github.com/gdamore/tcell/v2"
)

type Mode int

const (
	NormalMode Mode = iota
	InputMode
)

type App struct {
	screen      tcell.Screen
	quit        bool
	focusedView string
	views       []*View
	logs        []string
}

func (app *App) Write(p []byte) (n int, err error) {
    n = len(p)
    app.logs = append(app.logs, string(p))
    return n, nil
}

func (app *App) PopLog() string {
    if len(app.logs) < 1 {
        return ""
    }

    v := app.logs[0]
    app.logs = app.logs[1:]
    return v
}

type View struct {
	Name             string
	App              *App
	Mode             Mode
	Keybinds         []Keybind
	x, y             int
	w, h             int
	cells            []cell
	renderFunc       func(*View)
	Cursorx, Cursory int
	border           bool
	paddingt         int
	paddingr         int
	paddingb         int
	paddingl         int
	inputBuffer      []rune
	fillColor        tcell.Color
}

type Keybind struct {
	name        string
	description string
	key         tcell.Key
	mode        Mode
	callback    func(*View)
}

type cell struct {
	x, y  int
	char  rune
	style tcell.Style
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

	app := App{
		screen: screen,
	}

	if err != nil {
		slog.Warn("Unable to load from disk. File may not exist", "error", err)
	}

	return &app
}

func (app *App) Cleanup() {
	app.screen.Fini()
}

func (app *App) Size() (w int, h int) {
	return app.screen.Size()
}

func (app *App) MainLoop() {
	for !app.quit {
		app.Draw()
		app.handleEvent(app.screen.PollEvent())
	}
}

func (app *App) AddView(v *View) {
	v.App = app
	app.views = append(app.views, v)
}

func (app *App) Focus(viewName string) {
	app.focusedView = viewName
}

func (app *App) GetFocusedView() (*View, error) {
	for _, v := range app.views {
		if v.Name == app.focusedView {
			return v, nil
		}
	}
	return nil, errors.New("View not found")
}

func (app *App) Quit() {
	app.quit = true
}

func (app *App) handleEvent(ev tcell.Event) {
	focusedView, err := app.GetFocusedView()
	if err == nil {
		focusedView.handleEvent(ev)
		return
	}

	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyCtrlC {
			app.Quit()
		}
	}
}

func (app *App) Draw() {
	app.screen.Clear()
	for _, v := range app.views {
		v.Clear()
		v.renderFunc(v)
		v.Draw(app.screen)
	}

	app.screen.Show()
}

func NewView(name string, x, y, w, h int, renderFunc func(*View)) *View {
	v := View{
		Name:        name,
		Mode:        NormalMode,
		x:           x,
		y:           y,
		w:           w,
		h:           h,
		cells:       make([]cell, 0),
		renderFunc:  renderFunc,
		border:      true,
		inputBuffer: []rune{},
		fillColor:   tcell.ColorDefault,
	}

	return &v
}

func (v *View) getInnerBounds() (x1, y1, x2, y2 int) {
	x1 = v.x + 1 + v.paddingl
	y1 = v.y + 1 + v.paddingt
	x2 = v.x + v.w - 2 - v.paddingr
	y2 = v.y + v.h - 2 - v.paddingb

    if v.h == 1 {
        y1 = v.y
        y2 = v.y
    }
    if v.w == 1 {
        x1 = v.x
        x2 = v.x
    }

	return x1, y1, x2, y2
}

func (v *View) getOuterBounds() (x1, y1, x2, y2 int) {
	x1 = v.x
	y1 = v.y
	x2 = v.x + v.w - 1
	y2 = v.y + v.h - 1
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
	bx1, by1, bx2, by2 := v.getOuterBounds()

	// Draw fillColor
	fillStyle := tcell.StyleDefault.Background(v.fillColor)
	for yidx := by1; yidx <= by2; yidx++ {
		for xidx := bx1; xidx <= bx2; xidx++ {
			screen.SetContent(xidx, yidx, ' ', nil, fillStyle)
		}
	}

	if v.border && v.h > 2 {
		screen.SetContent(bx1, by1, tcell.RuneULCorner, nil, fillStyle)
		screen.SetContent(bx2, by1, tcell.RuneURCorner, nil, fillStyle)
		screen.SetContent(bx1, by2, tcell.RuneLLCorner, nil, fillStyle)
		screen.SetContent(bx2, by2, tcell.RuneLRCorner, nil, fillStyle)

		for xidx := bx1 + 1; xidx < bx2; xidx++ {
			screen.SetContent(xidx, by1, tcell.RuneHLine, nil, fillStyle)
			screen.SetContent(xidx, by2, tcell.RuneHLine, nil, fillStyle)
		}
		for yidx := by1 + 1; yidx < by2; yidx++ {
			screen.SetContent(bx1, yidx, tcell.RuneVLine, nil, fillStyle)
			screen.SetContent(bx2, yidx, tcell.RuneVLine, nil, fillStyle)
		}
	}

	for _, cell := range v.cells {
		x := x1 + cell.x
		y := y1 + cell.y
		screen.SetContent(x, y, cell.char, nil, cell.style)
	}
}

func (v *View) Bind(mode Mode, key tcell.Key, name, description string, cb func(*View)) {
	kb := Keybind{
		name:        name,
		description: description,
		key:         key,
		callback:    cb,
		mode:        mode,
	}
	v.Keybinds = append(v.Keybinds, kb)
}

func (v *View) handleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		var key tcell.Key
		if ev.Key() == tcell.KeyRune {
			if v.Mode == InputMode {
				v.inputBuffer = append(v.inputBuffer, ev.Rune())
				return
			}
			key = tcell.Key(ev.Rune())
		} else {
			key = ev.Key()
		}
		kb, err := v.getKeybind(v.Mode, key)
		if err != nil {
			kb.callback(v)
		}
	}
}

func (v *View) getKeybind(m Mode, key tcell.Key) (Keybind, error) {
	for _, kb := range v.Keybinds {
		if kb.mode == m && kb.key == key {
			return kb, errors.New("Keybind does not exist")
		}
	}
	return Keybind{}, nil
}

func (v *View) SetPadding(t, r, b, l int) {
	v.paddingt = t
	v.paddingr = r
	v.paddingb = b
	v.paddingl = l
}

func (v *View) Width() int {
	return v.w
}

func (v *View) InnerWidth() int {
	return v.w - v.paddingl - v.paddingr - 2
}

func (v *View) ShowCursor() {
	x1, y1, _, _ := v.getInnerBounds()
	v.App.screen.ShowCursor(x1+v.Cursorx, y1+v.Cursory)
}

func (v *View) HideCursor() {
	v.App.screen.HideCursor()
}

func (v *View) GetInputBuffer() []rune {
	return v.inputBuffer
}

func (v *View) SetInputBuffer(runes []rune) {
	v.inputBuffer = runes
}

func (v *View) ClearInputBuffer() {
	v.inputBuffer = []rune{}
}

func (v *View) SetFillColor(color tcell.Color) {
	v.fillColor = color
}
