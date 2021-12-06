package code_error

import "fmt"

type Error struct {
	Code   string
	Detail string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Detail)
}

func (e Error) GetCode() string {
	return e.Code
}

func (e Error) GetDetail() string {
	return e.Detail
}
