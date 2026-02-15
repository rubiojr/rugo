package stringview

// AnyStringView mimics Qt6's QAnyStringView — a value-type struct that wraps
// a string for API purposes. The bridge should treat params of this type as
// GoString and auto-convert using the FromString constructor.
type AnyStringView struct {
	s string
}

// NewAnyStringView creates an AnyStringView from a Go string.
func NewAnyStringView(s string) AnyStringView {
	return AnyStringView{s: s}
}

// String returns the wrapped string.
func (v AnyStringView) String() string {
	return v.s
}

// Widget is a simple struct with methods that take AnyStringView params.
type Widget struct {
	name string
}

// NewWidget creates a Widget.
func NewWidget() *Widget {
	return &Widget{}
}

// SetObjectName takes an AnyStringView param (string view type).
func (w *Widget) SetObjectName(name AnyStringView) {
	w.name = name.s
}

// ObjectName returns the name.
func (w *Widget) ObjectName() string {
	return w.name
}

// SetTitle takes a plain string param (for comparison).
func (w *Widget) SetTitle(title string) {
	w.name = title
}

// PtrStringView is like AnyStringView but its constructor returns a pointer.
type PtrStringView struct {
	s string
}

// NewPtrStringView creates a PtrStringView returning a pointer.
func NewPtrStringView(s string) *PtrStringView {
	return &PtrStringView{s: s}
}

// SetTooltip takes a PtrStringView (pointer-returning constructor).
func (w *Widget) SetTooltip(tip PtrStringView) {
	w.name = tip.s
}
