package encoding

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// MaxFrameLength is the maximum message size to parse
const MaxFrameLength = 10240

// OptimalFrameLength is the initial buffer size for scanning
const OptimalFrameLength = 1024

// ErrBadFrame is returned when the scanner cannot parse syslog message boundaries
var ErrBadFrame = errors.New("bad frame")

// ErrInvalidStructuredData is returned when structure data has any value other than '-' (blank)
var ErrInvalidStructuredData = errors.New("invalid structured data")

// ErrInvalidPriVal is returned when pri-val is not properly formatted
var ErrInvalidPriVal = errors.New("invalid pri-val")

var privalVersionRe = regexp.MustCompile(`<(\d+)>(\d)`)

// Decode converts a rfc5424 message to our model
func Decode(raw []byte, hasStructuredData bool) (Message, error) {
	msg := Message{}

	b := bytes.NewBuffer(raw)
	priVal, err := syslogField(b)
	if err != nil {
		return msg, err
	}

	privalVersion := privalVersionRe.FindAllSubmatch(priVal, -1)
	if len(privalVersion) != 1 || len(privalVersion[0]) != 3 {
		return msg, ErrInvalidPriVal
	}
	prio, err := strconv.ParseUint(string(privalVersion[0][1]), 10, 8)
	if err != nil {
		return msg, err
	}
	msg.Priority = uint8(prio)

	version, err := strconv.ParseUint(string(privalVersion[0][2]), 10, 16)
	if err != nil {
		return msg, err
	}
	msg.Version = uint16(version)

	rawTime, err := syslogField(b)
	if err != nil {
		return msg, err
	}
	msg.Timestamp, err = time.Parse(FlexibleSyslogTimeFormat, string(rawTime))
	if err != nil {
		return msg, err
	}

	hostname, err := syslogField(b)
	if err != nil {
		return msg, err
	}
	msg.Hostname = string(hostname)

	application, err := syslogField(b)
	if err != nil {
		return msg, err
	}
	msg.Application = string(application)

	process, err := syslogField(b)
	if err != nil {
		return msg, err
	}
	msg.Process = string(process)

	id, err := syslogField(b)
	if err != nil {
		return msg, err
	}
	msg.ID = string(id)

	if hasStructuredData {
		// trash structured data, as we don't use it ever
		if err = trashStructuredData(b); err != nil {
			return msg, err
		}
	}

	msg.Message = b.String()

	return msg, nil
}

// syslogScanner is a octet-frame syslog parser
type syslogScanner struct {
	parser       *bufio.Scanner
	item         Message
	err          error
	rfcCompliant bool
}

// Scanner is the general purpose primitive for parsing message bodies coming
// from log-shuttle, logfwd, logplex and all sorts of logging components.
type Scanner interface {
	Scan() bool
	Err() error
	Message() Message
}

// NewScanner is a syslog octet frame stream parser
func newSyslogScanner(r io.Reader, rfcCompliant bool) Scanner {
	s := &syslogScanner{
		parser:       bufio.NewScanner(r),
		rfcCompliant: rfcCompliant,
	}
	s.parser.Buffer(make([]byte, OptimalFrameLength), MaxFrameLength)
	s.parser.Split(syslogParser())

	return s
}

// NewScanner is a syslog octet frame stream parser
func NewScanner(r io.Reader) Scanner {
	return newSyslogScanner(r, true)
}

// Message returns the current message
func (s *syslogScanner) Message() Message {
	return s.item
}

// Err returns the last scanner error
func (s *syslogScanner) Err() error {
	if err := s.parser.Err(); err != nil {
		return err
	}

	return s.err
}

// Scan returns true until all messages are parsed or an error occurs.
// When an error occur, the underlying error will be presented as `Err()`
func (s *syslogScanner) Scan() bool {
	if !s.parser.Scan() {
		return false
	}

	s.item, s.err = Decode(s.parser.Bytes(), s.rfcCompliant)
	return s.err == nil
}

// NewDrainScanner returns a scanner for use with drain endpoints. The primary
// difference is that it's lose and doesn't check for structured data.
func NewDrainScanner(r io.ReadCloser) Scanner {
	return newSyslogScanner(r, false)
}

func syslogField(b *bytes.Buffer) ([]byte, error) {
	g, err := b.ReadBytes(' ')
	if err != nil {
		return nil, err
	}
	if len(g) > 0 {
		g = g[:len(g)-1]
	}
	return g, nil
}

func trashStructuredData(b *bytes.Buffer) error {
	// notice the quoting
	// [meta sequenceId=\"518\"][meta somethingElse=\"bl\]ah\"]
	firstChar, err := b.ReadByte()
	if err != nil {
		return err
	}

	if firstChar == '-' {
		// trash the following space too
		_, err = b.ReadByte()
		return err
	}

	if firstChar != '[' {
		return ErrInvalidStructuredData
	}

	quoting := false
	bracketing := true

	for {
		c, err := b.ReadByte()
		if err != nil {
			return err
		}

		if !bracketing {
			if c == ' ' {
				// we done!
				// consumed the last ']' and hit a space
				break
			}

			if c != '[' {
				return ErrInvalidStructuredData
			}

			bracketing = true
			continue
		}

		// makes sure we dont catch '\]' as per RFC
		// PARAM-VALUE     = UTF-8-STRING ; characters '"', '\' and ']' MUST be escaped.
		if quoting {
			quoting = false
			continue
		}

		switch c {
		case '\\':
			quoting = true
		case ']':
			bracketing = false
		}
	}

	return nil
}
