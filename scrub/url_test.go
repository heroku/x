package url

import (
	"net/url"
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

func (s *ScrubberSuite) TestScrubURL() {
	u, _ := url.Parse("https://api.heroku.com/login?username=foo&password=bar")
	uu := URL(u)

	query := uu.Query()
	assert.Equal(s.T(), "foo", query.Get("username"))
	assert.Equal(s.T(), scrubbedValue, query.Get("password"))

	originalQuery := u.Query()
	assert.Equal(s.T(), "foo", originalQuery.Get("username"))
	assert.Equal(s.T(), "bar", originalQuery.Get("password"))
}

func (s *ScrubberSuite) TestScrubURLCaseInsensitive() {
	u, _ := url.Parse("https://api.heroku.com/login?username=foo&passWord=bar")
	uu := URL(u)

	query := uu.Query()
	assert.Equal(s.T(), "foo", query.Get("username"))
	assert.Equal(s.T(), scrubbedValue, query.Get("passWord"))

	originalQuery := u.Query()
	assert.Equal(s.T(), "foo", originalQuery.Get("username"))
	assert.Equal(s.T(), "bar", originalQuery.Get("passWord"))
}

func (s *ScrubberSuite) TestScrubURLQueryWithURL() {
	u, _ := url.Parse("https://api.heroku.com/login?url=https://user:password@api.heroku.com/login")
	uu := URL(u)

	query := uu.Query()
	uu, err := url.Parse(query.Get("url"))
	assert.NoError(s.T(), err)

	userInfo := uu.User
	password, _ := userInfo.Password()
	assert.Equal(s.T(), "user", userInfo.Username())
	assert.Equal(s.T(), scrubbedValue, password)

	originalQuery := u.Query()
	uu, err = url.Parse(originalQuery.Get("url"))
	assert.NoError(s.T(), err)

	originalUserInfo := uu.User
	originalPassword, _ := originalUserInfo.Password()
	assert.Equal(s.T(), "user", originalUserInfo.Username())
	assert.Equal(s.T(), "password", originalPassword)
}

func (s *ScrubberSuite) TestScrubURLUserInfoPassword() {
	u, _ := url.Parse("https://user:password@api.heroku.com/login")
	uu := URL(u)

	userInfo := uu.User
	password, _ := userInfo.Password()

	assert.Equal(s.T(), "user", userInfo.Username())
	assert.Equal(s.T(), scrubbedValue, password)

	originalUserInfo := u.User
	originalPassword, _ := originalUserInfo.Password()

	assert.Equal(s.T(), "user", originalUserInfo.Username())
	assert.Equal(s.T(), "password", originalPassword)
}
