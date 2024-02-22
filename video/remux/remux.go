// Package remux provides functions for remuxing videos.
package remux

import "github.com/Darkness4/fc2-live-dl-go/video/concat"

// Option is the option for remux.
type Option concat.Option

// WithAudioOnly sets the remux to audio only.
func WithAudioOnly() Option {
	return Option(concat.WithAudioOnly())
}

// Do remuxes the input file to the output file.
func Do(output string, input string, opts ...Option) error {
	o := make([]concat.Option, 0, len(opts))
	for _, opt := range opts {
		o = append(o, concat.Option(opt))
	}

	return concat.Do(output, []string{input}, o...)
}
