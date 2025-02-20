package gotuit

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gdamore/tcell/v2"
)

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
	InputCursor      int
	fillColor        tcell.Color
	visible          bool
	Parent           *View
	Children         []*View
	focusedview      string
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
		visible:     true,
		focusedview: name,
	}

	return &v
}

func (v *View) GetFocusedView() (*View, error) {
    if v.focusedview == v.Name {
        return v, nil
    }

	for _, c := range v.Children {
		if c.Name == v.focusedview {
			return c, nil
		}
	}
	return nil, errors.New("Child not found")
}

func (self *View) Focus(viewName string) {
    self.focusedview = viewName
}

func (self *View) FocusedView() string {
    return self.focusedview
}

func (parent *View) AddChild(child *View) {
	child.Parent = parent
	parent.Children = append(parent.Children, child)
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

func (kb *Keybind) String() string {
	keyString := tcell.KeyNames[kb.key]
	if keyString == "" {
		keyString = string(kb.key)
	}
	if keyString == " " {
		keyString = "<space>"
	}

	return fmt.Sprintf("%s - %s [%s]", keyString, kb.name, kb.description)
}

func (kb *Keybind) Mode() Mode {
	return kb.mode
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

	if v.Parent != nil {
		px1, py1, px2, py2 := v.Parent.getInnerBounds()
		x1 += px1
		y1 += py1
		x2 -= px2
		y2 -= py2
	}

	return x1, y1, x2, y2
}

func (v *View) getOuterBounds() (x1, y1, x2, y2 int) {
	x1 = v.x
	y1 = v.y
	x2 = v.x + v.w - 1
	y2 = v.y + v.h - 1

	if v.Parent != nil {
		px1, py1, _, _ := v.Parent.getInnerBounds()
		x1 += px1
		y1 += py1
		x2 += px1
		y2 += py1
	}

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

func (v *View) SetTextContent(x, y int, text string, style tcell.Style) {
	width := v.InnerWidth()
	for xidx, t := range text {
		if xidx < width {
			v.SetContent(x+xidx, y, t, style)
		}
	}
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

	if len(v.Children) > 0 {
		for _, child := range v.Children {
			if !child.visible {
				continue
			}
            child.Clear()
			child.renderFunc(child)
			child.Draw(screen)
		}
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

func (v *View) handleInputRune(r rune) {
	if v.InputCursor == len(v.inputBuffer) {
		v.inputBuffer = append(v.inputBuffer, r)
	} else {
		head := v.inputBuffer[:v.InputCursor]
		tail := v.inputBuffer[v.InputCursor:]
		v.inputBuffer = append(head, append([]rune{r}, tail...)...)
	}
	v.InputCursor++
}

func (self *View) handleEvent(ev tcell.Event) {
    if len(self.Children) > 0 {
        if self.focusedview != self.Name {
            focusedView, err := self.GetFocusedView()
            if err != nil {
                slog.Error("No focused child view", "error", err)
            } else {
                focusedView.handleEvent(ev)
                return
            }
        }
    }

	switch ev := ev.(type) {
	case *tcell.EventKey:
		var key tcell.Key
		if ev.Key() == tcell.KeyRune {
			if self.Mode == InputMode {
				self.handleInputRune(ev.Rune())
				return
			}
			key = tcell.Key(ev.Rune())
		} else {
			key = ev.Key()
		}
		kb, err := self.getKeybind(self.Mode, key)
		if err == nil {
			kb.callback(self)
		}
	}
}

func (v *View) getKeybind(m Mode, key tcell.Key) (Keybind, error) {
	for _, kb := range v.Keybinds {
		if kb.mode == m && kb.key == key {
			return kb, nil
		}
	}
	return Keybind{}, errors.New("Keybind does not exist")
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

func (v *View) Height() int {
	return v.h
}

func (v *View) InnerWidth() int {
	return v.w - v.paddingl - v.paddingr - 2
}

func (v *View) InnerHeight() int {
	return v.h - v.paddingt - v.paddingb
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
	v.InputCursor = 0
}

func (v *View) SetFillColor(color tcell.Color) {
	v.fillColor = color
}

func (v *View) Show() {
	v.visible = true
}

func (v *View) Hide() {
	v.visible = false
}
