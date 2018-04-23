package carriers

import (
	"fmt"
	"os"
	"strings"
)

const defaultPrefix = "yoonit-tracer-"

type Environ struct {
	prefix string
}

func NewEnvironCarrier() *Environ {
	return &Environ{prefix: defaultPrefix}
}

func (e *Environ) Set(key, val string) {
	os.Setenv(fmt.Sprintf("%s%s", e.prefix, key), val)
}

func (e *Environ) ForeachKey(handler func(key, val string) error) error {
	allEnv := os.Environ()
	for _, env := range allEnv {
		ep := strings.SplitN(env, "=", 2)
		if len(ep) != 2 {
			continue
		}
		if !strings.HasPrefix(ep[0], e.prefix) {
			continue
		}
		if err := handler(ep[0][len(e.prefix):], ep[1]); err != nil {
			return err
		}
	}
	return nil
}
