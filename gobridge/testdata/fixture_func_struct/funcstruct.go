package funcstruct

// --- Struct types used as callback parameters ---

// Item represents a simple struct passed by pointer in callbacks.
type Item struct {
	Name string
}

// NewItem creates a new Item.
func NewItem(name string) *Item {
	return &Item{Name: name}
}

// Date represents a value-type struct passed by value in callbacks.
type Date struct {
	Year int
}

// NewDate creates a new Date.
func NewDate(year int) Date {
	return Date{Year: year}
}

// Nested__Type uses double underscores, mimicking Qt nested types
// like QAbstractTextDocumentLayout__PaintContext.
type Nested__Type struct {
	Value string
}

// --- Widget with signal-like methods taking func callbacks ---

// Widget is the main struct with callback-taking methods.
type Widget struct {
	name string
}

// NewWidget creates a Widget.
func NewWidget() *Widget {
	return &Widget{}
}

// OnItemClicked takes a callback with a pointer-struct param.
// Mimics Qt signal: OnItemClicked(slot func(item *Item))
func (w *Widget) OnItemClicked(slot func(item *Item)) {
	slot(NewItem("test"))
}

// OnDateChanged takes a callback with a value-struct param.
// Mimics Qt signal: OnDateChanged(slot func(date Date))
func (w *Widget) OnDateChanged(slot func(date Date)) {
	slot(NewDate(2025))
}

// OnMixed takes a callback with struct + primitive params.
// Mimics: OnItemActivated(slot func(item *Item, column int))
func (w *Widget) OnMixed(slot func(item *Item, column int)) {
	slot(NewItem("mixed"), 42)
}

// OnMultiStruct takes a callback with multiple struct params.
func (w *Widget) OnMultiStruct(slot func(a *Item, b *Item)) {
	slot(NewItem("a"), NewItem("b"))
}

// OnValueAndPointer takes a callback mixing value and pointer structs.
func (w *Widget) OnValueAndPointer(slot func(d Date, item *Item)) {
	slot(NewDate(2026), NewItem("vp"))
}

// OnNestedType takes a callback with a nested-name struct (double underscore).
func (w *Widget) OnNestedType(slot func(ctx *Nested__Type)) {
	slot(&Nested__Type{Value: "nested"})
}

// OnPrimitiveOnly takes a callback with only primitives (baseline, should always work).
func (w *Widget) OnPrimitiveOnly(slot func(row int)) {
	slot(7)
}

// OnNoArgs takes a callback with no params (baseline).
func (w *Widget) OnNoArgs(slot func()) {
	slot()
}

// --- Functions with callback edge cases ---

// OnChanCallback takes a callback with a chan param — should remain blocked.
func (w *Widget) OnChanCallback(slot func(ch chan int)) {
	slot(make(chan int))
}

// OnFuncCallback takes a callback with a nested func param.
func (w *Widget) OnFuncCallback(slot func(fn func())) {
	slot(func() {})
}

// SetName is a regular method for comparison.
func (w *Widget) SetName(name string) {
	w.name = name
}

// GetName returns the name.
func (w *Widget) GetName() string {
	return w.name
}
