// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"code.gitea.io/gitea/modules/validation"
)

type FederatedUser struct {
	ID               int64  `xorm:"pk autoincr"`
	UserID           int64  `xorm:"NOT NULL"`
	ExternalID       string `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	FederationHostID int64  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
}

func NewFederatedUser(userID int64, externalID string, federationHostID int64) (FederatedUser, error) {
	result := FederatedUser{
		UserID:           userID,
		ExternalID:       externalID,
		FederationHostID: federationHostID,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUser{}, err
	}
	return result, nil
}

func (user FederatedUser) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	return result
}
