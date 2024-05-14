// Copyright 2024 The Forgejo Authors. All rights reserved.
// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"code.gitea.io/gitea/modules/timeutil"
)

// ErrNotValid represents an validation error
type ErrNotValid struct {
	Message string
}

func (err ErrNotValid) Error() string {
	return fmt.Sprintf("Validation Error: %v", err.Message)
}

// IsErrNotValid checks if an error is a ErrNotValid.
func IsErrNotValid(err error) bool {
	_, ok := err.(ErrNotValid)
	return ok
}

type Validateable interface {
	Validate() []string
}

func IsValid(v Validateable) (bool, error) {
	if err := v.Validate(); len(err) > 0 {
		typeof := reflect.TypeOf(v)
		errString := strings.Join(err, "\n")
		return false, ErrNotValid{fmt.Sprint(typeof, ": ", errString)}
	}

	return true, nil
}

func ValidateNotEmpty(value any, name string) []string {
	isValid := true
	switch v := value.(type) {
	case string:
		if v == "" {
			isValid = false
		}
	case timeutil.TimeStamp:
		if v.IsZero() {
			isValid = false
		}
	case int64:
		if v == 0 {
			isValid = false
		}
	default:
		isValid = false
	}

	if isValid {
		return []string{}
	}
	return []string{fmt.Sprintf("%v should not be empty", name)}
}

func ValidateMaxLen(value string, maxLen int, name string) []string {
	if utf8.RuneCountInString(value) > maxLen {
		return []string{fmt.Sprintf("Value %v was longer than %v", name, maxLen)}
	}
	return []string{}
}

func ValidateOneOf(value any, allowed []any, name string) []string {
	for _, allowedElem := range allowed {
		if value == allowedElem {
			return []string{}
		}
	}
	return []string{fmt.Sprintf("Value %v is not contained in allowed values %v", value, allowed)}
}
