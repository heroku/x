package encoding

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/bmizerany/lpx"
	syslog "github.com/influxdata/go-syslog"
	"github.com/influxdata/go-syslog/octetcounting"
	"github.com/pkg/errors"
)

// ErrBadFrame is returned when the scanner cannot parse syslog message boundaries
var ErrBadFrame = errors.New("bad frame")

// Decode converts a rfc5424 message to our model
func Decode(res syslog.Message) Message {
	return Message{
		Priority:    *res.Priority(),
		Version:     res.Version(),
		Timestamp:   *res.Timestamp(),
		Hostname:    nilStringPointer(res.Hostname()),
		Application: nilStringPointer(res.Appname()),
		Process:     nilStringPointer(res.ProcID()),
		ID:          nilStringPointer(res.MsgID()),
		Message:     nilStringPointer(res.Message()),
	}
}

// syslogScanner is a octet-frame syslog parser
type syslogScanner struct {
	parser syslog.Parser
	item   Message
	err    error
	more   chan *syslog.Result
}

// Scanner is the general purpose primitive for parsing message bodies coming
// from log-shuttle, logfwd, logplex and all sorts of logging components.
type Scanner interface {
	Scan() bool
	Err() error
	Message() Message
}

// NewScanner is a syslog octet frame stream parser
func NewScanner(r io.Reader) Scanner {
	s := &syslogScanner{
		more: make(chan *syslog.Result, 1),
	}
	s.parser = octetcounting.NewParser(syslog.WithListener(s.next))

	go func() {
		s.parser.Parse(r)
		close(s.more)
	}()

	return s
}

func (s *syslogScanner) next(r *syslog.Result) {
	s.more <- r
}

// Message returns the curent message
func (s *syslogScanner) Message() Message {
	return s.item
}

// Err returns the last scanner error
func (s *syslogScanner) Err() error {
	return s.err
}

// Scan returns true until all messages are parsed or an error occurs.
// When an error occur, the underlying error will be presented as `Err()`
func (s *syslogScanner) Scan() bool {
	r, ok := <-s.more
	if !ok {
		return false
	}

	if r.Error != nil {
		s.err = errors.Wrap(ErrBadFrame, r.Error.Error())
		return false
	}

	s.item = Decode(r.Message)
	return true
}

var privalVersionRe = regexp.MustCompile(`<(\d+)>(\d)+`)

// NewDrainScanner returns a scanner for use with drain endpoints. The primary
// difference is that it's lose and doesn't check for structured data.
func NewDrainScanner(r io.ReadCloser) Scanner {
	return &drainScanner{
		lp: lpx.NewReader(bufio.NewReader(r)),
	}
}

type drainScanner struct {
	message Message
	lp      *lpx.Reader
	err     error
}

// Message returns the last parsed message.
func (s *drainScanner) Message() Message {
	return s.message
}

// Err returns the last known error.
func (s *drainScanner) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.lp.Err()
}

// Scan returns true when a message was parsed. It returns false otherwise.
func (s *drainScanner) Scan() bool {
	if !s.lp.Next() {
		return false
	}

	hdr := s.lp.Header()
	ts, err := time.Parse(SyslogTimeFormat, string(hdr.Time))
	if err != nil {
		s.err = err
		return false
	}

	privalVersion := privalVersionRe.FindAllSubmatch(hdr.PrivalVersion, -1)

	priority, err := strconv.Atoi(string(privalVersion[0][1]))
	if err != nil {
		s.err = err
		return false
	}

	version, err := strconv.Atoi(string(privalVersion[0][2]))
	if err != nil {
		s.err = err
		return false
	}

	s.message = Message{
		Priority:    uint8(priority),
		Version:     uint16(version),
		ID:          string(hdr.Msgid),
		Timestamp:   ts,
		Hostname:    string(hdr.Hostname),
		Application: string(hdr.Name),
		Process:     string(hdr.Procid),
		Message:     string(s.lp.Bytes()),
	}
	return true
}
