package datagateway

import "github.com/pkg/errors"

// This error is checked for in emitter.loopEmit.
var ErrRecordNotFound = errors.New("record not found")

func WrapErrRecordNotFound(err error, keyNotFound string) error {
	err = errors.Wrap(ErrRecordNotFound, err.Error())
	return errors.Wrapf(err, "key %s not found", keyNotFound)
}
