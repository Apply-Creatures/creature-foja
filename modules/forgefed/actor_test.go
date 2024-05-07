// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"reflect"
	"strings"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

func TestNewPersonId(t *testing.T) {
	expected := PersonID{}
	expected.ID = "1"
	expected.Source = "forgejo"
	expected.Schema = "https"
	expected.Path = "api/v1/activitypub/user-id"
	expected.Host = "an.other.host"
	expected.Port = ""
	expected.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/1"
	sut, _ := NewPersonID("https://an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}

	expected = PersonID{}
	expected.ID = "1"
	expected.Source = "forgejo"
	expected.Schema = "https"
	expected.Path = "api/v1/activitypub/user-id"
	expected.Host = "an.other.host"
	expected.Port = "443"
	expected.UnvalidatedInput = "https://an.other.host:443/api/v1/activitypub/user-id/1"
	sut, _ = NewPersonID("https://an.other.host:443/api/v1/activitypub/user-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}
}

func TestNewRepositoryId(t *testing.T) {
	setting.AppURL = "http://localhost:3000/"
	expected := RepositoryID{}
	expected.ID = "1"
	expected.Source = "forgejo"
	expected.Schema = "http"
	expected.Path = "api/activitypub/repository-id"
	expected.Host = "localhost"
	expected.Port = "3000"
	expected.UnvalidatedInput = "http://localhost:3000/api/activitypub/repository-id/1"
	sut, _ := NewRepositoryID("http://localhost:3000/api/activitypub/repository-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}
}

func TestActorIdValidation(t *testing.T) {
	sut := ActorID{}
	sut.Source = "forgejo"
	sut.Schema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/"
	if sut.Validate()[0] != "userId should not be empty" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}

	sut = ActorID{}
	sut.ID = "1"
	sut.Source = "forgejo"
	sut.Schema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/1?illegal=action"
	if sut.Validate()[0] != "not all input was parsed, \nUnvalidated Input:\"https://an.other.host/api/v1/activitypub/user-id/1?illegal=action\" \nParsed URI: \"https://an.other.host/api/v1/activitypub/user-id/1\"" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate()[0])
	}
}

func TestPersonIdValidation(t *testing.T) {
	sut := PersonID{}
	sut.ID = "1"
	sut.Source = "forgejo"
	sut.Schema = "https"
	sut.Path = "path"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/path/1"
	if _, err := validation.IsValid(sut); err.Error() != "path: \"path\" has to be a person specific api path" {
		t.Errorf("validation error expected but was: %v\n", err)
	}

	sut = PersonID{}
	sut.ID = "1"
	sut.Source = "forgejox"
	sut.Schema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.Port = ""
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/1"
	if sut.Validate()[0] != "Value forgejox is not contained in allowed values [forgejo gitea]" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate()[0])
	}
}

func TestWebfingerId(t *testing.T) {
	sut, _ := NewPersonID("https://codeberg.org/api/v1/activitypub/user-id/12345", "forgejo")
	if sut.AsWebfinger() != "@12345@codeberg.org" {
		t.Errorf("wrong webfinger: %v", sut.AsWebfinger())
	}

	sut, _ = NewPersonID("https://Codeberg.org/api/v1/activitypub/user-id/12345", "forgejo")
	if sut.AsWebfinger() != "@12345@codeberg.org" {
		t.Errorf("wrong webfinger: %v", sut.AsWebfinger())
	}
}

func TestShouldThrowErrorOnInvalidInput(t *testing.T) {
	var err any
	// TODO: remove after test
	//_, err = NewPersonId("", "forgejo")
	//if err == nil {
	//	t.Errorf("empty input should be invalid.")
	//}

	_, err = NewPersonID("http://localhost:3000/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("localhost uris are not external")
	}
	_, err = NewPersonID("./api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("relative uris are not allowed")
	}
	_, err = NewPersonID("http://1.2.3.4/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("uri may not be ip-4 based")
	}
	_, err = NewPersonID("http:///[fe80::1ff:fe23:4567:890a%25eth0]/api/v1/something", "forgejo")
	if err == nil {
		t.Errorf("uri may not be ip-6 based")
	}
	_, err = NewPersonID("https://codeberg.org/api/v1/activitypub/../activitypub/user-id/12345", "forgejo")
	if err == nil {
		t.Errorf("uri may not contain relative path elements")
	}
	_, err = NewPersonID("https://myuser@an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if err == nil {
		t.Errorf("uri may not contain unparsed elements")
	}

	_, err = NewPersonID("https://an.other.host/api/v1/activitypub/user-id/1", "forgejo")
	if err != nil {
		t.Errorf("this uri should be valid but was: %v", err)
	}
}

func Test_PersonMarshalJSON(t *testing.T) {
	sut := ForgePerson{}
	sut.Type = "Person"
	sut.PreferredUsername = ap.NaturalLanguageValuesNew()
	sut.PreferredUsername.Set("en", ap.Content("MaxMuster"))
	result, _ := sut.MarshalJSON()
	if string(result) != "{\"type\":\"Person\",\"preferredUsername\":\"MaxMuster\"}" {
		t.Errorf("MarshalJSON() was = %q", result)
	}
}

func Test_PersonUnmarshalJSON(t *testing.T) {
	expected := &ForgePerson{
		Actor: ap.Actor{
			Type: "Person",
			PreferredUsername: ap.NaturalLanguageValues{
				ap.LangRefValue{Ref: "en", Value: []byte("MaxMuster")},
			},
		},
	}
	sut := new(ForgePerson)
	err := sut.UnmarshalJSON([]byte(`{"type":"Person","preferredUsername":"MaxMuster"}`))
	if err != nil {
		t.Errorf("UnmarshalJSON() unexpected error: %v", err)
	}
	x, _ := expected.MarshalJSON()
	y, _ := sut.MarshalJSON()
	if !reflect.DeepEqual(x, y) {
		t.Errorf("UnmarshalJSON() expected: %q got: %q", x, y)
	}

	expectedStr := strings.ReplaceAll(strings.ReplaceAll(`{
		"id":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/10",
		"type":"Person",
		"icon":{"type":"Image","mediaType":"image/png","url":"https://federated-repo.prod.meissa.de/avatar/fa7f9c4af2a64f41b1bef292bf872614"},
		"url":"https://federated-repo.prod.meissa.de/stargoose9",
		"inbox":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/10/inbox",
		"outbox":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/10/outbox",
		"preferredUsername":"stargoose9",
		"publicKey":{"id":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/10#main-key",
			"owner":"https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/10",
			"publicKeyPem":"-----BEGIN PUBLIC KEY-----\nMIIBoj...XAgMBAAE=\n-----END PUBLIC KEY-----\n"}}`,
		"\n", ""),
		"\t", "")
	err = sut.UnmarshalJSON([]byte(expectedStr))
	if err != nil {
		t.Errorf("UnmarshalJSON() unexpected error: %v", err)
	}
	result, _ := sut.MarshalJSON()
	if expectedStr != string(result) {
		t.Errorf("UnmarshalJSON() expected: %q got: %q", expectedStr, result)
	}
}

func TestForgePersonValidation(t *testing.T) {
	sut := new(ForgePerson)
	sut.UnmarshalJSON([]byte(`{"type":"Person","preferredUsername":"MaxMuster"}`))
	if res, _ := validation.IsValid(sut); !res {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}
}
