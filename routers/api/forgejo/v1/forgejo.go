// SPDX-License-Identifier: MIT

package v1

import (
	"net/http"

	"code.gitea.io/gitea/modules/json"
)

type Forgejo struct{}

var _ ServerInterface = &Forgejo{}

func NewForgejo() *Forgejo {
	return &Forgejo{}
}

var ForgejoVersion = "development"

func (f *Forgejo) GetVersion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Version{&ForgejoVersion})
}
