// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

func Test_NewForgeLike(t *testing.T) {
	actorIRI := "https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"
	objectIRI := "https://codeberg.org/api/v1/activitypub/repository-id/1"
	want := []byte(`{"type":"Like","startTime":"2024-03-27T00:00:00Z","actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1","object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}`)

	startTime, _ := time.Parse("2006-Jan-02", "2024-Mar-27")
	sut, err := NewForgeLike(actorIRI, objectIRI, startTime)
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}
	if valid, _ := validation.IsValid(sut); !valid {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}

	got, err := sut.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON() error = \"%v\"", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MarshalJSON() got = %q, want %q", got, want)
	}
}

func Test_LikeMarshalJSON(t *testing.T) {
	type testPair struct {
		item    ForgeLike
		want    []byte
		wantErr error
	}

	tests := map[string]testPair{
		"empty": {
			item: ForgeLike{},
			want: nil,
		},
		"with ID": {
			item: ForgeLike{
				Activity: ap.Activity{
					Actor:  ap.IRI("https://repo.prod.meissa.de/api/v1/activitypub/user-id/1"),
					Type:   "Like",
					Object: ap.IRI("https://codeberg.org/api/v1/activitypub/repository-id/1"),
				},
			},
			want: []byte(`{"type":"Like","actor":"https://repo.prod.meissa.de/api/v1/activitypub/user-id/1","object":"https://codeberg.org/api/v1/activitypub/repository-id/1"}`),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.item.MarshalJSON()
			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("MarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_LikeUnmarshalJSON(t *testing.T) {
	type testPair struct {
		item    []byte
		want    *ForgeLike
		wantErr error
	}

	//revive:disable
	tests := map[string]testPair{
		"with ID": {
			item: []byte(`{"type":"Like","actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1","object":"https://codeberg.org/api/activitypub/repository-id/1"}`),
			want: &ForgeLike{
				Activity: ap.Activity{
					Actor:  ap.IRI("https://repo.prod.meissa.de/api/activitypub/user-id/1"),
					Type:   "Like",
					Object: ap.IRI("https://codeberg.org/api/activitypub/repository-id/1"),
				},
			},
			wantErr: nil,
		},
		"invalid": {
			item:    []byte(`{"type":"Invalid","actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1","object":"https://codeberg.org/api/activitypub/repository-id/1"`),
			want:    &ForgeLike{},
			wantErr: fmt.Errorf("cannot parse JSON:"),
		},
	}
	//revive:enable

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := new(ForgeLike)
			err := got.UnmarshalJSON(test.item)
			if (err != nil || test.wantErr != nil) && !strings.Contains(err.Error(), test.wantErr.Error()) {
				t.Errorf("UnmarshalJSON() error = \"%v\", wantErr \"%v\"", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UnmarshalJSON() got = %q, want %q, err %q", got, test.want, err.Error())
			}
		})
	}
}

func TestActivityValidation(t *testing.T) {
	sut := new(ForgeLike)
	sut.UnmarshalJSON([]byte(`{"type":"Like",
	"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if res, _ := validation.IsValid(sut); !res {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}

	sut.UnmarshalJSON([]byte(`{"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if sut.Validate()[0] != "type should not be empty" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate()[0])
	}

	sut.UnmarshalJSON([]byte(`{"type":"bad-type",
		"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if sut.Validate()[0] != "Value bad-type is not contained in allowed values [Like]" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate()[0])
	}

	sut.UnmarshalJSON([]byte(`{"type":"Like",
		"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "not a date"}`))
	if sut.Validate()[0] != "StartTime was invalid." {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}

	sut.UnmarshalJSON([]byte(`{"type":"Wrong",
		"actor":"https://repo.prod.meissa.de/api/activitypub/user-id/1",
	"object":"https://codeberg.org/api/activitypub/repository-id/1",
	"startTime": "2014-12-31T23:00:00-08:00"}`))
	if sut.Validate()[0] != "Value Wrong is not contained in allowed values [Like]" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}
}

func TestActivityValidation_Attack(t *testing.T) {
	sut := new(ForgeLike)
	sut.UnmarshalJSON([]byte(`{rubbish}`))
	if len(sut.Validate()) != 5 {
		t.Errorf("5 validateion errors expected but was: %v\n", len(sut.Validate()))
	}
}
