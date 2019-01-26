package validator

import (
	"errors"

	tagexpr "github.com/bytedance/go-tagexpr"
)

const matchExprName = "@"
const errMsgExprName = "msg"

// Validator struct fields validator
type Validator struct {
	vm *tagexpr.VM
}

// New creates a struct fields validator.
func New(tagName string) *Validator {
	v := &Validator{
		vm: tagexpr.New(tagName),
	}
	return v
}

// Validate validates whether the fields of structPtr is valid.
func (v *Validator) Validate(structPtr interface{}) error {
	expr, err := v.vm.Run(structPtr)
	if err != nil {
		return err
	}
	var errSelector string
	expr.Range(func(selector string, eval func() interface{}) bool {
		if !isMatchSelector(selector) {
			return true
		}
		valid, _ := eval().(bool)
		if !valid {
			errSelector = selector
		}
		return valid
	})
	if errSelector == "" {
		return nil
	}
	errMsg := expr.EvalString(errSelector + errMsgExprName)
	if errMsg != "" {
		return errors.New(errMsg)
	}
	return errors.New("Invalid parameter: " + errSelector[:len(errSelector)-1])
}

func isMatchSelector(selector string) bool {
	n := len(selector)
	return n > 1 && selector[n-1] == '@' && selector[n-2] != '@'
}
