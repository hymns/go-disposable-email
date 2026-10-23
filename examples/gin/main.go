// Sign-up with Gin. Gin binds with go-playground/validator, so the check is
// a struct tag: notdisposable.
//
//	go run . &
//	curl -i -d '{"email":"a@mailinator.com"}' localhost:8080/register   # 400
//	curl -i -d '{"email":"a@gmail.com"}' localhost:8080/register        # 201
package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	disposable "github.com/hymns/go-disposable-email"
)

type signUp struct {
	Email string `json:"email" binding:"required,email,notdisposable"`
}

func main() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("notdisposable", func(fl validator.FieldLevel) bool {
			return !disposable.IsDisposable(fl.Field().String())
		})
	}
	r := gin.Default()
	r.POST("/register", register)
	log.Fatal(r.Run(":8080"))
}

func register(c *gin.Context) {
	var req signUp
	if err := c.ShouldBindJSON(&req); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) && errs[0].Tag() == "notdisposable" {
			err = disposable.ErrDisposable
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// create the account
	c.Status(http.StatusCreated)
}
