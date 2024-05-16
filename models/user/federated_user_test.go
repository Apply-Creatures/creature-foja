// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	"code.gitea.io/gitea/modules/validation"
)

func Test_FederatedUserValidation(t *testing.T) {
	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederatedUser{
		ExternalID:       "12",
		FederationHostID: 1,
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid")
	}
}
