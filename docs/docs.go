package docs

import (
	_ "embed"

	"github.com/swaggo/swag"
)

//go:embed swagger.json
var swaggerJSON string

func init() {
	spec := &swag.Spec{
		Version:          "1.0.0",
		Host:             "",
		BasePath:         "/api/v1",
		Schemes:          []string{},
		Title:            "FoodRescue API",
		Description:      "REST API backend untuk aplikasi mobile Flutter \"FoodRescue\" — platform penyelamatan dan donasi surplus makanan.",
		InfoInstanceName: "swagger",
		SwaggerTemplate:  swaggerJSON,
		LeftDelim:        "{{",
		RightDelim:       "}}",
	}
	swag.Register(spec.InstanceName(), spec)
}