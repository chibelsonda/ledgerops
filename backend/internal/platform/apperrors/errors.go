package apperrors

import "errors"

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("record conflicts with existing data")
var ErrInvalidReference = errors.New("referenced record is invalid")
var ErrInvalidState = errors.New("operation would create an invalid state")

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }
