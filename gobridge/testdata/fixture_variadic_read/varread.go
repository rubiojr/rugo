package varread

import "io"

type GPX struct {
	Version string
}

type readOptions struct {
	strict bool
}

type ReadOption func(*readOptions)

func WithStrict(v bool) ReadOption {
	return func(opts *readOptions) {
		opts.strict = v
	}
}

func Read(r io.Reader, options ...ReadOption) (*GPX, error) {
	_ = r
	_ = options
	return &GPX{Version: "1.1"}, nil
}
