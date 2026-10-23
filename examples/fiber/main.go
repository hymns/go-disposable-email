// Sign-up with Fiber v3.
//
//	go run . &
//	curl -i -H 'Content-Type: application/json' -d '{"email":"a@mailinator.com"}' localhost:8080/register   # 400
//	curl -i -H 'Content-Type: application/json' -d '{"email":"a@gmail.com"}' localhost:8080/register        # 201
package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	disposable "github.com/hymns/go-disposable-email"
)

func main() {
	app := fiber.New()
	app.Post("/register", register)
	log.Fatal(app.Listen(":8080"))
}

func register(c fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON")
	}
	if err := disposable.Check(req.Email); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	// create the account
	return c.SendStatus(fiber.StatusCreated)
}
