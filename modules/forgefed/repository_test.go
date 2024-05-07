// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"reflect"
	"testing"

	"code.gitea.io/gitea/modules/json"

	ap "github.com/go-ap/activitypub"
)

func Test_RepositoryMarshalJSON(t *testing.T) {
	type testPair struct {
		item    Repository
		want    []byte
		wantErr error
	}

	tests := map[string]testPair{
		"empty": {
			item: Repository{},
			want: nil,
		},
		"with ID": {
			item: Repository{
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
				Team: nil,
			},
			want: []byte(`{"id":"https://example.com/1"}`),
		},
		"with Team as IRI": {
			item: Repository{
				Team: ap.IRI("https://example.com/1"),
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":"https://example.com/1"}`),
		},
		"with Team as IRIs": {
			item: Repository{
				Team: ap.ItemCollection{
					ap.IRI("https://example.com/1"),
					ap.IRI("https://example.com/2"),
				},
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":["https://example.com/1","https://example.com/2"]}`),
		},
		"with Team as Object": {
			item: Repository{
				Team: ap.Object{ID: "https://example.com/1"},
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":{"id":"https://example.com/1"}}`),
		},
		"with Team as slice of Objects": {
			item: Repository{
				Team: ap.ItemCollection{
					ap.Object{ID: "https://example.com/1"},
					ap.Object{ID: "https://example.com/2"},
				},
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":[{"id":"https://example.com/1"},{"id":"https://example.com/2"}]}`),
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

func Test_RepositoryUnmarshalJSON(t *testing.T) {
	type testPair struct {
		data    []byte
		want    *Repository
		wantErr error
	}

	tests := map[string]testPair{
		"nil": {
			data:    nil,
			wantErr: fmt.Errorf("cannot parse JSON: %w", fmt.Errorf("cannot parse empty string; unparsed tail: %q", "")),
		},
		"empty": {
			data:    []byte{},
			wantErr: fmt.Errorf("cannot parse JSON: %w", fmt.Errorf("cannot parse empty string; unparsed tail: %q", "")),
		},
		"with Type": {
			data: []byte(`{"type":"Repository"}`),
			want: &Repository{
				Actor: ap.Actor{
					Type: RepositoryType,
				},
			},
		},
		"with Type and ID": {
			data: []byte(`{"id":"https://example.com/1","type":"Repository"}`),
			want: &Repository{
				Actor: ap.Actor{
					ID:   "https://example.com/1",
					Type: RepositoryType,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := new(Repository)
			err := got.UnmarshalJSON(tt.data)
			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("UnmarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if tt.want != nil && !reflect.DeepEqual(got, tt.want) {
				jGot, _ := json.Marshal(got)
				jWant, _ := json.Marshal(tt.want)
				t.Errorf("UnmarshalJSON() got = %s, want %s", jGot, jWant)
			}
		})
	}
}
