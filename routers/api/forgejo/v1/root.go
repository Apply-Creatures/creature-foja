// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package v1

import (
	"net/http"
)

func Root(w http.ResponseWriter, r *http.Request) {
	// https://www.rfc-editor.org/rfc/rfc8631
	w.Header().Set("Link", "</assets/forgejo/api.v1.yml>; rel=\"service-desc\"")
	w.WriteHeader(http.StatusNoContent)
}
