package remux

import "github.com/Darkness4/fc2-live-dl-go/video/concat"

type Option concat.Option

func WithAudioOnly() Option {
	return Option(concat.WithAudioOnly())
}

func Do(output string, input string, opts ...Option) error {
	o := make([]concat.Option, 0, len(opts))
	for _, opt := range opts {
		o = append(o, concat.Option(opt))
	}

	return concat.Do(output, []string{input}, o...)
}
