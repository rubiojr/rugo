package varread

import "io"

type GPX struct {
	Version    string
	Metadata   *MetadataType
	Wpt        []*WptType
	Rte        []*RteType
	Trk        []*TrkType
	Extensions *ExtensionsType
}

type MetadataType struct {
	Name string
}

type WptType struct {
	Name string
}

type RteType struct {
	Name string
}

type TrkType struct {
	Name string
}

type ExtensionsType struct {
	XML []byte
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
	return &GPX{
		Version:    "1.1",
		Metadata:   &MetadataType{Name: "meta"},
		Wpt:        []*WptType{{Name: "wpt1"}},
		Rte:        []*RteType{{Name: "rte1"}},
		Trk:        []*TrkType{{Name: "trk1"}},
		Extensions: &ExtensionsType{XML: []byte("x")},
	}, nil
}
