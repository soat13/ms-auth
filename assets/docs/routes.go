package docs

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed index.html
var swaggerIndex []byte

func Register(app *fiber.App) {
	app.Get("/openapi.yaml", func(c *fiber.Ctx) error {
		c.Type("yaml")
		return c.Send(openapiSpec)
	})
	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Type("html")
		return c.Send(swaggerIndex)
	})
}
