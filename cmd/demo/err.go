package main

import (
	"errors"
	"fmt"
	"github.com/imattdu/orbit/errorx"
)

func main() {
	errNotFound := errorx.CodeEntry{
		Code:    404,
		Message: "not found",
	}
	err := errorx.Wrap(errors.New("not found"), errNotFound, errorx.WithSuccess(true))
	fmt.Println(err.Error())

}
