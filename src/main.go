package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	e := gin.New()
	_ = e
	fmt.Println("Hello Go1.12.4")
}
