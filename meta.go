package apperr

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

// MetaPair is one key/value pair of wire metadata, built with Meta.
type MetaPair struct {
	Key   string
	Value string
}

// Meta returns one key/value pair of wire metadata for WithMeta.
func Meta(key, value string) MetaPair {
	return MetaPair{Key: key, Value: value}
}

// metaError attaches wire metadata to an error without changing its message or
// chain.
type metaError struct {
	err  error
	meta map[string]string
}

// WithMeta returns err carrying the given wire metadata: key/value pairs that
// describe this one failure and travel with it to the client, such as the ID a
// request was refused for. A transport adapter (see the apperrgrpc package)
// puts them on the wire, and a user-safe Entry's Message fills its {key}
// placeholders from them. Recover them with Metadata.
//
// Wire metadata is not the same as request Fields. Fields (ContextWithFields)
// describe the request, live on the context, and reach only the Recorder and
// Logger; they are never sent to the client. Metadata describes the error,
// lives on the error, and is meant to be sent. Put nothing in it the client
// may not see.
//
// The result's Error, Code, errors.Is and errors.As behave exactly as for err.
// A later pair overrides an earlier one with the same key. WithMeta returns nil
// for a nil err, and err itself when there are no pairs.
func WithMeta(err error, pairs ...MetaPair) error {
	if err == nil || len(pairs) == 0 {
		return err
	}
	meta := make(map[string]string, len(pairs))
	for _, p := range pairs {
		meta[p.Key] = p.Value
	}
	return &metaError{err: err, meta: meta}
}

// Error returns the wrapped error's message unchanged; metadata is not part of
// the log text.
func (e *metaError) Error() string { return e.err.Error() }

// Unwrap returns the wrapped error so errors.Is, errors.As and Code traverse
// past the metadata.
func (e *metaError) Unwrap() error { return e.err }

// Metadata returns the wire metadata attached to err with WithMeta anywhere in
// its chain, including inside errors.Join trees. When two layers set the same
// key, the outer one (the one seen first, walking the chain as errors.As does)
// wins. It returns nil when err is nil or carries no metadata. The result is a
// copy.
func Metadata(err error) map[string]string {
	var out map[string]string
	walk(err, func(e error) {
		me, ok := e.(*metaError)
		if !ok {
			return
		}
		if out == nil {
			out = make(map[string]string, len(me.meta))
		}
		for k, v := range me.meta {
			if _, seen := out[k]; !seen {
				out[k] = v
			}
		}
	})
	return out
}

// walk visits err and every error it wraps, depth first, in errors.As order.
func walk(err error, visit func(error)) {
	if err == nil {
		return
	}
	visit(err)
	switch u := err.(type) {
	case interface{ Unwrap() error }:
		walk(u.Unwrap(), visit)
	case interface{ Unwrap() []error }:
		for _, e := range u.Unwrap() {
			walk(e, visit)
		}
	}
}
