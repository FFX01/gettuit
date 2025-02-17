package gotuit

import (
	"errors"
	"github.com/gdamore/tcell/v2"
	"log"
	"log/slog"
	"os"
)

type App struct {
	screen      tcell.Screen
	quit        bool
	focusedView string
	views       []*View
	logs        []string
	keybinds    []GlobalKeybind
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

type GlobalKeybind struct {
	name        string
	description string
	key         tcell.Key
	callback    func(*App)
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

func (app *App) GetView(name string) (view *View, ok bool) {
	for _, v := range app.views {
		if v.Name == name {
			return v, true
		}
	}
	return nil, false
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
	switch ev := ev.(type) {
	case *tcell.EventKey:
		var key tcell.Key
		if ev.Key() == tcell.KeyRune {
			key = tcell.Key(ev.Rune())
		} else {
			key = ev.Key()
		}
		kb, err := app.getKeybind(key)
		if err == nil {
			kb.callback(app)
			return
		}
	}

	focusedView, err := app.GetFocusedView()
	if err == nil {
		focusedView.handleEvent(ev)
		return
	}
}

func (app *App) ShowView(name string) error {
	for _, v := range app.views {
		if v.Name == name {
			v.Show()
			return nil
		}
	}
	return errors.New("View not found")
}

func (app *App) HideView(name string) error {
	for _, v := range app.views {
		if v.Name == name {
			v.Hide()
			return nil
		}
	}
	return errors.New("View not found")
}

func (app *App) Draw() {
	app.screen.Clear()
	for _, v := range app.views {
		if !v.visible {
			continue
		}
		v.Clear()
		v.renderFunc(v)
		v.Draw(app.screen)
	}

	app.screen.Show()
}

func (app *App) Bind(key tcell.Key, name, description string, cb func(*App)) {
	kb := GlobalKeybind{
		name:        name,
		description: description,
		key:         key,
		callback:    cb,
	}
	app.keybinds = append(app.keybinds, kb)
}

func (app *App) getKeybind(key tcell.Key) (GlobalKeybind, error) {
	for _, kb := range app.keybinds {
		if kb.key == key {
			return kb, nil
		}
	}
	return GlobalKeybind{}, errors.New("Keybind not found")
}
