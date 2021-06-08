package godbf

import "errors"

var (
	record_index_out_of_range = errors.New("record index out of range")
	field_not_exists = errors.New("field name not exists")
	empty_fields = errors.New("no fields found")
	errLocked = errors.New("file already locked by other process")
)