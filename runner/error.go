package runner

import (
	"fmt"
	"strings"
)

type ErrGroup struct {
	Errs []error
}

func (e ErrGroup) Error() string {
	var errStrings []string
	for _, err := range e.Errs {
		errStrings = append(errStrings, fmt.Sprintf("(%s)", err.Error()))
	}

	return fmt.Sprintf("multiple errors: %s", strings.Join(errStrings, " "))
}
