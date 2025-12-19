//go:build !windows

package window

import (
	"errors"

	"github.com/example/bidirect/internal/config"
)

type Window struct{}

func NewWindow(cfg config.Config) (*Window, error) {
	return nil, errors.New("window: not implemented on this platform")
}

func (w *Window) Run() error {
	return errors.New("window: not implemented on this platform")
}
