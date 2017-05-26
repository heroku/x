package header

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestScrubber(t *testing.T) {
	suite.Run(t, new(ScrubberSuite))
}

type ScrubberSuite struct {
	suite.Suite
}

func (s *ScrubberSuite) TestAuthorizationHeaderScrubbingCaseInsensitive() {
	fakeHeaders := http.Header{
		"authORIZATion": []string{"Basic adflskjafjklfds"},
	}
	headers := Header(fakeHeaders)

	assert.Equal(s.T(), []string{"Basic [SCRUBBED]"}, headers["authORIZATion"])
}

func (s *ScrubberSuite) TestAuthorizationHeaderScrubbingBearer() {
	fakeHeaders := http.Header{
		"Authorization": []string{"Bearer adflskjafjklfds"},
	}
	headers := Header(fakeHeaders)

	assert.Equal(s.T(), []string{"Bearer [SCRUBBED]"}, headers["Authorization"])
}

func (s *ScrubberSuite) TestAuthorizationHeaderScrubbingOther() {
	fakeHeaders := http.Header{
		"Authorization": []string{"adflskjafjklfds"},
	}
	headers := Header(fakeHeaders)

	assert.Equal(s.T(), []string{"[SCRUBBED]"}, headers["Authorization"])
}

func (s *ScrubberSuite) TestMultipleAuthorizationHeaderScrubbing() {
	fakeHeaders := http.Header{
		"Authorization": []string{"Basic abc", "Bearer xyz"},
	}
	headers := Header(fakeHeaders)

	assert.Equal(s.T(), []string{"Basic [SCRUBBED]", "Bearer [SCRUBBED]"}, headers["Authorization"])
}

func (s *ScrubberSuite) TestCookieHeaderScrubbing() {
	fakeHeaders := http.Header{
		"Cookie": []string{"adflskjafjklfds"},
	}
	headers := Header(fakeHeaders)

	assert.Equal(s.T(), []string{"[SCRUBBED]"}, headers["Cookie"])
}

func (s *ScrubberSuite) TestCookieHeaderScrubbingCaseInsensitive() {
	fakeHeaders := http.Header{
		"cOOkie": []string{"adflskjafjklfds"},
	}
	headers := Header(fakeHeaders)

	assert.Equal(s.T(), []string{"[SCRUBBED]"}, headers["cOOkie"])
}
