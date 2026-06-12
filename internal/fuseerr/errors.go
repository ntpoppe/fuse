package fuseerr

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("connection not found")
	ErrAlreadyExists = errors.New("connection already exists")
	ErrQueryRowLimit = errors.New("query row limit exceeded")
	ErrReadOnly      = errors.New("read-only violation")
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

type ReadOnlyError struct {
	Cause error
}

func (e ReadOnlyError) Error() string {
	return fmt.Sprintf("read-only violation: %v", e.Cause)
}

func (e ReadOnlyError) Is(target error) bool {
	return target == ErrReadOnly
}

func (e ReadOnlyError) Unwrap() error {
	return e.Cause
}

type LegExecutionError struct {
	ConnectionID string
	Cause        error
}

func (e LegExecutionError) Error() string {
	return fmt.Sprintf("query leg %q: %v", e.ConnectionID, e.Cause)
}

func (e LegExecutionError) Unwrap() error {
	return e.Cause
}
