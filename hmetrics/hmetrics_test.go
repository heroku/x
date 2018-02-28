/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */
package hmetrics

import "testing"

func TestStartable_EmptyEndpoint(t *testing.T) {
	err := startable("")
	if err == nil {
		t.Errorf("Expected an error, but got nil instead")
	}
}
