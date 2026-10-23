// Sign-up with Echo.
//
//	go run . &
//	curl -i -H 'Content-Type: application/json' -d '{"email":"a@mailinator.com"}' localhost:8080/register   # 400
//	curl -i -H 'Content-Type: application/json' -d '{"email":"a@gmail.com"}' localhost:8080/register        # 201
package main

import (
	"log"
	"net/http"

	disposable "github.com/hymns/go-disposable-email"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	e.POST("/register", register)
	log.Fatal(e.Start(":8080"))
}

func register(c echo.Context) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON")
	}
	if err := disposable.Check(req.Email); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// create the account
	return c.NoContent(http.StatusCreated)
}
