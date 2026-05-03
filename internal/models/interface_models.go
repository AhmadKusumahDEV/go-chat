package models

type Entity interface {
	GetID() any
	Validate() error
	TableName() string
}
