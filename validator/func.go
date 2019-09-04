package validator

import (
	"regexp"

	tagexpr "github.com/bytedance/go-tagexpr"
	"github.com/nyaruka/phonenumbers"
)

// MustRegFunc registers validator function expression.
// NOTE:
//  panic if exist error;
//  example: phone($) or phone($,'CN');
//  If @force=true, allow to cover the existed same @funcName;
//  The go number types always are float64;
//  The go string types always are string.
func MustRegFunc(funcName string, fn func(args ...interface{}) bool, force ...bool) {
	err := RegFunc(funcName, fn, force...)
	if err != nil {
		panic(err)
	}
}

// RegFunc registers validator function expression.
// NOTE:
//  example: phone($) or phone($,'CN');
//  If @force=true, allow to cover the existed same @funcName;
//  The go number types always are float64;
//  The go string types always are string.
func RegFunc(funcName string, fn func(args ...interface{}) bool, force ...bool) error {
	return tagexpr.RegFunc(funcName, func(args ...interface{}) interface{} {
		return fn(args...)
	}, force...)
}

func init() {
	var pattern = "^([A-Za-z0-9_\\-\\.\u4e00-\u9fa5])+\\@([A-Za-z0-9_\\-\\.])+\\.([A-Za-z]{2,8})$"
	emailRegexp := regexp.MustCompile(pattern)
	MustRegFunc("email", func(args ...interface{}) bool {
		if len(args) != 1 {
			return false
		}
		s, ok := args[0].(string)
		if !ok {
			return false
		}
		return emailRegexp.MatchString(s)
	}, true)
}

func init() {
	MustRegFunc("phone", func(args ...interface{}) bool {
		var numberToParse, defaultRegion string
		var ok bool
		switch len(args) {
		default:
			return false
		case 2:
			defaultRegion, ok = args[1].(string)
			if !ok {
				return false
			}
			fallthrough
		case 1:
			numberToParse, ok = args[0].(string)
			if !ok {
				return false
			}
		}
		num, err := phonenumbers.Parse(numberToParse, defaultRegion)
		if err != nil {
			return false
		}
		return phonenumbers.IsValidNumber(num)
	}, true)
}
