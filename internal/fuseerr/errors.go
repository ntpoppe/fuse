package fuseerr

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound         = errors.New("connection not found")
	ErrAlreadyExists    = errors.New("connection already exists")
	ErrQueryRowLimit    = errors.New("query row limit exceeded")
)

type NotFoundError struct {
	ID string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("connection %q not found", e.ID)
}

func (e NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

type AlreadyExistsError struct {
	ID string
}

func (e AlreadyExistsError) Error() string {
	return fmt.Sprintf("connection %q already exists", e.ID)
}

func (e AlreadyExistsError) Is(target error) bool {
	return target == ErrAlreadyExists
}

type QueryRowLimitError struct {
	Limit int
}

func (e QueryRowLimitError) Error() string {
	return fmt.Sprintf("query returned more than %d rows", e.Limit)
}

func (e QueryRowLimitError) Is(target error) bool {
	return target == ErrQueryRowLimit
}
