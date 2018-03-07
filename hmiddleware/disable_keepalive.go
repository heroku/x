/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import (
	"context"
	"net/http"
)

// DisableKeepalive ...
func DisableKeepalive(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	w.Header().Set("Connection", "close")
	return ctx
}
