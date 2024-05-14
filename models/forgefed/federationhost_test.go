// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/validation"
)

func Test_FederationHostValidation(t *testing.T) {
	sut := FederationHost{
		HostFqdn: "host.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederationHost{
		HostFqdn: "",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid: HostFqdn empty")
	}

	sut = FederationHost{
		HostFqdn: strings.Repeat("fill", 64),
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid: HostFqdn too long (len=256)")
	}

	sut = FederationHost{
		HostFqdn:       "host.do.main",
		NodeInfo:       NodeInfo{},
		LatestActivity: time.Now(),
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid: NodeInfo invalid")
	}

	sut = FederationHost{
		HostFqdn: "host.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now().Add(1 * time.Hour),
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid: Future timestamp")
	}

	sut = FederationHost{
		HostFqdn: "hOst.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid: HostFqdn lower case")
	}
}
