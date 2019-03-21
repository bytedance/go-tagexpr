// Copyright 2019 Bytedance Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tagexpr_test

import (
	"regexp"
	"testing"

	"github.com/bytedance/go-tagexpr"
)

func TestFunc(t *testing.T) {
	var pattern = "^([A-Za-z0-9_\\-\\.\u4e00-\u9fa5])+\\@([A-Za-z0-9_\\-\\.])+\\.([A-Za-z]{2,8})$"
	emailRegexp := regexp.MustCompile(pattern)
	tagexpr.RegSimpleFunc("email", func(v interface{}) interface{} {
		s, ok := v.(string)
		if !ok {
			return false
		}
		t.Log(s)
		return emailRegexp.MatchString(s)
	})
	err := tagexpr.RegSimpleFunc("email", func(v interface{}) interface{} {
		s, ok := v.(string)
		if !ok {
			return false
		}
		t.Log(s)
		return emailRegexp.MatchString(s)
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	type T struct {
		Email string `te:"email($)"`
	}
	vm := tagexpr.New("te")
	cases := []struct {
		email  string
		expect bool
	}{
		{"", false},
		{"henrylee2cn@gmail.com", true},
	}
	obj := new(T)
	for _, c := range cases {
		obj.Email = c.email
		te := vm.MustRun(obj)
		got := te.EvalBool("Email")
		if got != c.expect {
			t.Fatalf("email: %s, expect: %v, but got: %v", c.email, c.expect, got)
		}
	}
}
