// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederationHost))
}

func GetFederationHost(ctx context.Context, ID int64) (*FederationHost, error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where("id=?", ID).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationInfo record %v does not exist", ID)
	}
	if res, err := validation.IsValid(host); !res {
		return nil, fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	return host, nil
}

func FindFederationHostByFqdn(ctx context.Context, fqdn string) (*FederationHost, error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where("host_fqdn=?", strings.ToLower(fqdn)).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	return host, nil
}

func CreateFederationHost(ctx context.Context, host *FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).Insert(host)
	return err
}

func UpdateFederationHost(ctx context.Context, host *FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return fmt.Errorf("FederationInfo is not valid: %v", err)
	}
	_, err := db.GetEngine(ctx).ID(host.ID).Update(host)
	return err
}
