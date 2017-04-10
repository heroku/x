# scrub

Includes `scrub/url` and `scrub/header` for URL and HTTP header scrubbing of sensitive fields. Both packages define a set of fields/headers respectively to scrub. These values are pulled from [rollbar-blanket](github.com/heroku/rollbar-blanket/) with the goal of providing consistent log scrubbing  across our various applications.
