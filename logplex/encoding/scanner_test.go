package encoding

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestScanner(t *testing.T) {
	tests := map[string]struct {
		log   string
		count int
		err   error
	}{
		"single": {
			log:   "66 <190>1 2019-07-21T22:13:34.598992Z shuttle t.http shuttle - - 168\n",
			count: 1,
		},
		"multiple": {
			log:   "64 <190>1 2019-07-20T17:50:10.879238Z shuttle token shuttle - - 99\n65 <190>1 2019-07-20T17:50:10.879238Z shuttle token shuttle - - 100\n",
			count: 2,
		},

		"short read": {
			log:   "64 <190>1 2019-07-20T17:50:10.879238Z shuttle token shuttle - - 99\n10 ---",
			count: 1,
			err:   ErrBadFrame,
		},

		"bad frame size": {
			log:   "64 <190>1 2019-07-20T17:50:10.879238Z shuttle token shuttle - - 99\n70 <190>1 2019-07-20T17:50:10.879238Z shuttle token shuttle - - 100",
			count: 1,
			err:   ErrBadFrame,
		},

		"bad frame position": {
			log:   "64 <190>1 2019-07-20T17:50:10.879238Z shuttle token shuttle - - 99\n 65 <190>1 2019-07-20T17:50:10.879238Z shuttle token shuttle - - 100",
			count: 1,
			err:   ErrBadFrame,
		},

		"bad frame format": {
			log:   "xxxxxxxxxxxxxxxx",
			count: 0,
			err:   ErrBadFrame,
		},
	}

	isCause := func(cause error, err error) bool {
		return !(errors.Cause(err) == cause)
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(test.log))
			i := 0
			for scanner.Scan() {
				i++
			}
			if got, want := i, test.count; got != want {
				t.Errorf("expected %v, got %v", want, got)
			}

			if got, want := scanner.Err(), test.err; isCause(test.err, scanner.Err()) {
				t.Errorf("scanner: expected %v, got %v", want, got)
			}
		})
	}

}
