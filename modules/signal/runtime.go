//go:build ignore

package signalmod

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

// --- signal module ---
//
// Race-free by construction: registered handlers execute on the goroutine
// that calls Wait(), never on Go's internal signal-delivery goroutine. While
// a handler runs, the Wait() goroutine is busy and the program's main flow is
// parked inside Wait(), so a handler never runs concurrently with the code
// that installed it. The handlers map itself is guarded by an RWMutex so that
// registration and dispatch are safe even across goroutines.

type Signal struct {
	mu       sync.RWMutex
	handlers map[os.Signal]func(...interface{}) interface{}
	ch       chan os.Signal
	once     sync.Once
}

// ensure lazily initializes the handler map and delivery channel.
func (s *Signal) ensure() {
	s.once.Do(func() {
		s.handlers = make(map[os.Signal]func(...interface{}) interface{})
		s.ch = make(chan os.Signal, 1)
	})
}

// nameToSignal maps a friendly signal name to an os.Signal. Both "INT" and
// "SIGINT" forms are accepted, case-insensitively.
func nameToSignal(name string) os.Signal {
	n := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(name)), "SIG")
	switch n {
	case "INT":
		return syscall.SIGINT
	case "TERM":
		return syscall.SIGTERM
	case "HUP":
		return syscall.SIGHUP
	case "QUIT":
		return syscall.SIGQUIT
	case "USR1":
		return syscall.SIGUSR1
	case "USR2":
		return syscall.SIGUSR2
	case "WINCH":
		return syscall.SIGWINCH
	case "PIPE":
		return syscall.SIGPIPE
	case "ALRM":
		return syscall.SIGALRM
	default:
		panic(fmt.Sprintf("unknown signal name %q (try INT, TERM, HUP, QUIT, USR1, USR2, WINCH, PIPE, ALRM)", name))
	}
}

// On registers a handler for the named signal. The handler runs on the
// goroutine that calls Wait(), which is what makes signal handling race-free.
// Registering the same signal again replaces the previous handler.
func (s *Signal) On(name string, handler interface{}) interface{} {
	fn, ok := handler.(func(...interface{}) interface{})
	if !ok {
		panic(fmt.Sprintf("signal.on expects a function handler, got %T", handler))
	}
	sig := nameToSignal(name)
	s.ensure()
	s.mu.Lock()
	s.handlers[sig] = fn
	s.mu.Unlock()
	signal.Notify(s.ch, sig)
	return nil
}

// Wait blocks the calling goroutine and dispatches handlers as signals
// arrive. Handlers execute on this goroutine (not the OS signal-delivery
// goroutine), so they never race the main program flow. Wait does not return;
// a handler must call exit() to terminate the program.
func (s *Signal) Wait() interface{} {
	s.ensure()
	for sig := range s.ch {
		s.mu.RLock()
		h := s.handlers[sig]
		s.mu.RUnlock()
		if h != nil {
			h()
		}
	}
	return nil
}

// Reset stops invoking the handler for the named signal, undoing a prior
// signal.on. It calls signal.Reset, so the signal is no longer forwarded to
// Rugo. Note: Go does not restore the original SIG_DFL disposition for a
// signal it has already trapped, so use Ignore if you need the signal to have
// no effect.
func (s *Signal) Reset(name string) interface{} {
	sig := nameToSignal(name)
	s.ensure()
	s.mu.Lock()
	delete(s.handlers, sig)
	s.mu.Unlock()
	signal.Reset(sig)
	return nil
}

// Ignore causes the named signal to be ignored by the process.
func (s *Signal) Ignore(name string) interface{} {
	sig := nameToSignal(name)
	s.ensure()
	s.mu.Lock()
	delete(s.handlers, sig)
	s.mu.Unlock()
	signal.Ignore(sig)
	return nil
}
