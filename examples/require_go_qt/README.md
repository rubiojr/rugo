# Qt GUI Example

A Qt6 GUI application written in Rugo using [miqt](https://github.com/mappu/miqt) — **no Go wrapper code needed**.

The GoOpaque bridge auto-discovers 3600+ functions, 600+ struct types with
hundreds of methods each, signal connections with Rugo lambdas, and the
full Qt class hierarchy via embedded struct navigation.

## Prerequisites

- Qt6 development libraries (`apt install qt6-base-dev` on Debian/Ubuntu)
- Go 1.22+
- miqt: `rugo run` will fetch it automatically, or pre-cache with `go install github.com/mappu/miqt/qt6@v0.13.0`

## Build and run

```bash
rugo build main.rugo -o hello_qt
./hello_qt
```

Paint-event variant:

```bash
rugo build paint_event.rugo -o paint_event
./paint_event
```

## How it works

No Go adapter needed. The Rugo script requires miqt/qt6 directly from the
Go module cache. A Rugo hash with lambdas provides a clean convenience layer
over the raw miqt function names:

```ruby
qt = {
  button: fn(text) qt6.new_qpush_button3(text) end,
  label:  fn(text) qt6.new_qlabel3(text) end
}
```

Everything works directly on opaque handles:

```ruby
win.set_window_title("Hello!")           # method via DotCall
btn.on_clicked(fn() ... end)             # signal + Rugo lambda
layout.add_widget(btn)                   # auto-upcast: QPushButton → QWidget
```

`paint_event.rugo` is an interactive QWidget virtual-event demo with nested callback params:

```ruby
win.on_paint_event(fn(super_paint, event)
  super_paint(event)
  puts "painted"
end)
```

It also binds regular widget interactions (`on_clicked`) and mouse events (`on_mouse_press_event`) so you can trigger repaints on demand and see paint-event logs update in the terminal.
