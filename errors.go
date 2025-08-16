package store

import (
	"errors"
	"fmt"
	"strings"

	"core/validation"
)

// Sentinel errors for common storage operations.
var (
	// Connection errors
	ErrConnectionFailed  = errors.New("connection failed")
	ErrConnectionTimeout = errors.New("connection timeout")
	ErrConnectionClosed  = errors.New("connection closed")
	ErrInvalidConnection = errors.New("invalid connection")

	// Driver errors
	ErrDriverNotFound     = errors.New("driver not found")
	ErrDriverNotSupported = errors.New("driver not supported")
	ErrDriverInitFailed   = errors.New("driver initialization failed")

	// Transaction errors
	ErrTransactionFailed  = errors.New("transaction failed")
	ErrTransactionAborted = errors.New("transaction aborted")
	ErrTransactionTimeout = errors.New("transaction timeout")
	ErrInvalidTransaction = errors.New("invalid transaction")

	// Query errors
	ErrQueryFailed  = errors.New("query failed")
	ErrQueryTimeout = errors.New("query timeout")
	ErrInvalidQuery = errors.New("invalid query")
	ErrQuerySyntax  = errors.New("query syntax error")

	// Record errors
	ErrRecordNotFound = errors.New("record not found")
	ErrRecordExists   = errors.New("record already exists")
	ErrInvalidRecord  = errors.New("invalid record")

	// Constraint errors
	ErrUniqueConstraint     = errors.New("unique constraint violation")
	ErrForeignKeyConstraint = errors.New("foreign key constraint violation")
	ErrCheckConstraint      = errors.New("check constraint violation")
	ErrNotNullConstraint    = errors.New("not null constraint violation")

	// Validation errors
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidInput     = errors.New("invalid input")
	ErrMissingRequired  = errors.New("missing required field")

	// Configuration errors
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrMissingConfig = errors.New("missing configuration")

	// Generic errors
	ErrNotImplemented = errors.New("not implemented")
	ErrNotSupported   = errors.New("operation not supported")
	ErrInternal       = errors.New("internal error")
)

// ConnectionError represents connection-related errors.
type ConnectionError struct {
	Operation string
	Driver    string
	Host      string
	Err       error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error during %s with %s driver at %s: %v",
		e.Operation, e.Driver, e.Host, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// DriverError represents driver-related errors.
type DriverError struct {
	Driver    string
	Operation string
	Err       error
}

func (e *DriverError) Error() string {
	return fmt.Sprintf("driver error with %s during %s: %v",
		e.Driver, e.Operation, e.Err)
}

func (e *DriverError) Unwrap() error {
	return e.Err
}

// TransactionError represents transaction-related errors.
type TransactionError struct {
	Operation string
	Err       error
}

func (e *TransactionError) Error() string {
	return fmt.Sprintf("transaction error during %s: %v", e.Operation, e.Err)
}

func (e *TransactionError) Unwrap() error {
	return e.Err
}

// QueryError represents query execution errors.
type QueryError struct {
	Operation string
	Table     string
	Query     string
	Args      []any
	Err       error
}

func (e *QueryError) Error() string {
	if e.Table != "" {
		return fmt.Sprintf("query error during %s on table %s: %v",
			e.Operation, e.Table, e.Err)
	}
	return fmt.Sprintf("query error during %s: %v", e.Operation, e.Err)
}

func (e *QueryError) Unwrap() error {
	return e.Err
}

// RecordNotFoundError represents a record not found error.
type RecordNotFoundError struct {
	Table string
	ID    string
}

func (e *RecordNotFoundError) Error() string {
	return fmt.Sprintf("record not found in table %s with ID %s", e.Table, e.ID)
}

// ValidationError represents validation errors.
type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// ConfigError represents configuration errors.
type ConfigError struct {
	Field   string
	Value   any
	Message string
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config error for field %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("config error: %s", e.Message)
}

// Constructor functions for custom errors

// NewConnectionError creates a new connection error.
func NewConnectionError(err error, operation, driver, host string) *ConnectionError {
	return &ConnectionError{
		Operation: operation,
		Driver:    driver,
		Host:      host,
		Err:       err,
	}
}

// NewDriverError creates a new driver error.
func NewDriverError(err error, driver, operation string) *DriverError {
	return &DriverError{
		Driver:    driver,
		Operation: operation,
		Err:       err,
	}
}

// NewTransactionError creates a new transaction error.
func NewTransactionError(err error, operation string) *TransactionError {
	return &TransactionError{
		Operation: operation,
		Err:       err,
	}
}

// NewQueryError creates a new query error.
func NewQueryError(err error, operation, table, query string, args []any) *QueryError {
	return &QueryError{
		Operation: operation,
		Table:     table,
		Query:     query,
		Args:      args,
		Err:       err,
	}
}

// NewRecordNotFoundError creates a new record not found error.
func NewRecordNotFoundError(table, id string) *RecordNotFoundError {
	return &RecordNotFoundError{
		Table: table,
		ID:    id,
	}
}

// NewValidationError creates a new validation error.
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		Message: message,
	}
}

// NewValidationErrorForField creates a new validation error for a specific field.
func NewValidationErrorForField(field string, value any, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewConfigError creates a new config error.
func NewConfigError(message string) *ConfigError {
	return &ConfigError{
		Message: message,
	}
}

// NewConfigErrorForField creates a new config error for a specific field.
func NewConfigErrorForField(field string, value any, message string) *ConfigError {
	return &ConfigError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// Wrapper functions for adding context to errors

// WrapConnectionError wraps an error as a connection error.
func WrapConnectionError(err error, operation, driver, host string) error {
	if err == nil {
		return nil
	}
	return NewConnectionError(err, operation, driver, host)
}

// WrapDriverError wraps an error as a driver error.
func WrapDriverError(err error, driver, operation string) error {
	if err == nil {
		return nil
	}
	return NewDriverError(err, driver, operation)
}

// WrapTransactionError wraps an error as a transaction error.
func WrapTransactionError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return NewTransactionError(err, operation)
}

// WrapQueryError wraps an error as a query error.
func WrapQueryError(err error, operation, table, query string, args []any) error {
	if err == nil {
		return nil
	}
	return NewQueryError(err, operation, table, query, args)
}

// RepositoryError represents repository operation errors.
type RepositoryError struct {
	EntityName string
	Operation  string
	Context    map[string]any
	Err        error
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf("repository error in %s.%s: %v", e.EntityName, e.Operation, e.Err)
}

func (e *RepositoryError) Unwrap() error {
	return e.Err
}

// WrapRepositoryError wraps an error with repository context.
func WrapRepositoryError(err error, entityName, operation string, context map[string]any) error {
	if err == nil {
		return nil
	}
	return &RepositoryError{
		EntityName: entityName,
		Operation:  operation,
		Context:    context,
		Err:        err,
	}
}

// Error checking functions

// IsConnectionError checks if an error is a connection error.
func IsConnectionError(err error) bool {
	var connErr *ConnectionError
	return errors.As(err, &connErr)
}

// IsDriverError checks if an error is a driver error.
func IsDriverError(err error) bool {
	var driverErr *DriverError
	return errors.As(err, &driverErr)
}

// IsTransactionError checks if an error is a transaction error.
func IsTransactionError(err error) bool {
	var txErr *TransactionError
	return errors.As(err, &txErr)
}

// IsQueryError checks if an error is a query error.
func IsQueryError(err error) bool {
	var queryErr *QueryError
	return errors.As(err, &queryErr)
}

// IsRecordNotFoundError checks if an error is a record not found error.
func IsRecordNotFoundError(err error) bool {
	var notFoundErr *RecordNotFoundError
	return errors.As(err, &notFoundErr)
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsConfigError checks if an error is a config error.
func IsConfigError(err error) bool {
	var configErr *ConfigError
	return errors.As(err, &configErr)
}

// NewValidationErrorFromResult creates a validation error from a validation result.
func NewValidationErrorFromResult(result *validation.Result, entity interface{}) *ValidationError {
	if result.IsValid {
		return nil
	}

	messages := make([]string, 0, len(result.Errors))
	for _, err := range result.Errors {
		messages = append(messages, err.Error())
	}

	return &ValidationError{
		Message: "validation failed: " + strings.Join(messages, "; "),
	}
}
