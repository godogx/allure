package allure

import "github.com/spf13/afero"

// FormatterOption sets formatter option.
type FormatterOption interface {
	applyFormatterOption(*formatter)
}

type formatterOptionFunc func(*formatter)

func (f formatterOptionFunc) applyFormatterOption(formatter *formatter) {
	f(formatter)
}

// WithFs sets filesystem for formatter.
func WithFs(fs afero.Fs) FormatterOption {
	return formatterOptionFunc(func(formatter *formatter) {
		formatter.fs = fs
	})
}
