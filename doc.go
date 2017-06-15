// Package httpmetrics provides an http.Handler for collecting metrics about http servers.
//
// Metrics are prefixed with:
//
//		<method>.<normalized path>
//              all
//
// For example, a request to GET /apps/:foo/bars/:bar_id emits metrics prefixed with:
//
//              get.apps.foo.bars.bar-id
//              all
//
// For each unique path, and under the global all prefix, servers will report:
//
//		requests - counter of requests
//		request-duration.ms - histogram of request durations in milliseconds
//		response-statuses.<status code> - counter of response status codes, eg 200
//
package httpmetrics
