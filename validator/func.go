package validator

import (
	"regexp"

	tagexpr "github.com/bytedance/go-tagexpr"
)

// RegValidateFunc registers simple validate function expression.
// NOTE:
//  example: email($) or email();
//  If @force=true, allow to cover the existed same @funcName;
//  The go number types always are float64;
//  The go string types always are string.
func RegValidateFunc(funcName string, fn func(v interface{}) bool, force ...bool) error {
	return tagexpr.RegSimpleFunc(funcName, func(v interface{}) interface{} {
		return fn(v)
	}, force...)
}

func init() {
	var pattern = "^([A-Za-z0-9_\\-\\.\u4e00-\u9fa5])+\\@([A-Za-z0-9_\\-\\.])+\\.([A-Za-z]{2,8})$"
	emailRegexp := regexp.MustCompile(pattern)
	tagexpr.RegSimpleFunc("email", func(v interface{}) interface{} {
		s, ok := v.(string)
		if !ok {
			return false
		}
		return emailRegexp.MatchString(s)
	}, true)
}
