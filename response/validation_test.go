package response

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

func TestValidateStruct_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name: "valid example",
			input: ValidationExample{
				Email:    "user@example.com",
				Password: "securepass123",
				Age:      25,
			},
		},
		{
			name: "valid with minimum age",
			input: ValidationExample{
				Email:    "test@example.org",
				Password: "password1",
				Age:      0,
			},
		},
		{
			name: "valid with maximum age",
			input: ValidationExample{
				Email:    "elder@example.com",
				Password: "validpassword",
				Age:      130,
			},
		},
		{
			name: "valid with minimum password length",
			input: ValidationExample{
				Email:    "min@example.com",
				Password: "12345678",
				Age:      50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestValidateStruct_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		errField string
	}{
		{
			name: "invalid email",
			input: ValidationExample{
				Email:    "not-an-email",
				Password: "securepass123",
				Age:      25,
			},
			errField: "Email",
		},
		{
			name: "empty email",
			input: ValidationExample{
				Email:    "",
				Password: "securepass123",
				Age:      25,
			},
			errField: "Email",
		},
		{
			name: "password too short",
			input: ValidationExample{
				Email:    "user@example.com",
				Password: "short",
				Age:      25,
			},
			errField: "Password",
		},
		{
			name: "empty password",
			input: ValidationExample{
				Email:    "user@example.com",
				Password: "",
				Age:      25,
			},
			errField: "Password",
		},
		{
			name: "age below minimum",
			input: ValidationExample{
				Email:    "user@example.com",
				Password: "securepass123",
				Age:      -1,
			},
			errField: "Age",
		},
		{
			name: "age above maximum",
			input: ValidationExample{
				Email:    "user@example.com",
				Password: "securepass123",
				Age:      131,
			},
			errField: "Age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if err == nil {
				t.Error("Expected validation error, got nil")
				return
			}

			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				found := false
				for _, fieldErr := range validationErrors {
					if fieldErr.Field() == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error for field %s, but it was not found in: %v", tt.errField, validationErrors)
				}
			} else {
				t.Errorf("Expected validator.ValidationErrors, got: %T", err)
			}
		})
	}
}

func TestValidateStruct_MultipleErrors(t *testing.T) {
	input := ValidationExample{
		Email:    "invalid",
		Password: "short",
		Age:      -5,
	}

	err := ValidateStruct(input)
	if err == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("Expected validator.ValidationErrors, got: %T", err)
	}

	if len(validationErrors) < 2 {
		t.Errorf("Expected at least 2 validation errors, got %d", len(validationErrors))
	}
}

func TestValidateStruct_CustomStruct(t *testing.T) {
	type CustomStruct struct {
		Name string `validate:"required,min=2,max=50"`
		Age  int    `validate:"required,gte=18,lte=100"`
	}

	tests := []struct {
		name      string
		input     CustomStruct
		shouldErr bool
	}{
		{
			name:      "valid custom struct",
			input:     CustomStruct{Name: "John", Age: 25},
			shouldErr: false,
		},
		{
			name:      "name too short",
			input:     CustomStruct{Name: "J", Age: 25},
			shouldErr: true,
		},
		{
			name:      "age too low",
			input:     CustomStruct{Name: "John", Age: 17},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if tt.shouldErr && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateAndRespond_Valid(t *testing.T) {
	app := fiber.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		input := ValidationExample{
			Email:    "user@example.com",
			Password: "securepass123",
			Age:      25,
		}

		if err := ValidateAndRespond(c, input); err != nil {
			return err
		}

		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestValidateAndRespond_Invalid(t *testing.T) {
	app := fiber.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		input := ValidationExample{
			Email:    "invalid-email",
			Password: "short",
			Age:      -5,
		}

		return ValidateAndRespond(c, input)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "error") {
		t.Errorf("Expected error in response body, got: %s", bodyStr)
	}
}

func TestValidateAndRespond_ErrorMessage(t *testing.T) {
	app := fiber.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		input := ValidationExample{
			Email:    "invalid",
			Password: "validpassword",
			Age:      25,
		}

		return ValidateAndRespond(c, input)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Email") {
		t.Errorf("Expected error message to mention Email field, got: %s", bodyStr)
	}
}

func TestValidateAndRespond_MultipleErrors(t *testing.T) {
	app := fiber.New()

	app.Post("/test", func(c *fiber.Ctx) error {
		input := ValidationExample{
			Email:    "bad",
			Password: "bad",
			Age:      -1,
		}

		return ValidateAndRespond(c, input)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "error") {
		t.Errorf("Expected error in response, got: %s", bodyStr)
	}
}

func TestValidationExample_Tags(t *testing.T) {
	validExample := ValidationExample{
		Email:    "test@example.com",
		Password: "password123",
		Age:      25,
	}

	err := ValidateStruct(validExample)
	if err != nil {
		t.Errorf("Valid example should pass validation, got: %v", err)
	}
}

func TestValidateStruct_NilInput(t *testing.T) {
	err := ValidateStruct(nil)
	if err == nil {
		t.Error("Expected error when validating nil, got nil")
	}
}

func TestValidateStruct_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}

	// Empty struct with no validation tags should pass
	err := ValidateStruct(EmptyStruct{})
	if err != nil {
		t.Errorf("Expected no error for empty struct, got: %v", err)
	}
}

func TestValidateAndRespond_WithDifferentInputs(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		app := fiber.New()
		app.Post("/test", func(c *fiber.Ctx) error {
			input := ValidationExample{
				Email:    "valid@example.com",
				Password: "validpass",
				Age:      30,
			}
			if err := ValidateAndRespond(c, input); err != nil {
				return err
			}
			return c.SendStatus(200)
		})

		req := httptest.NewRequest("POST", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		app := fiber.New()
		app.Post("/test", func(c *fiber.Ctx) error {
			input := ValidationExample{
				Email:    "invalid",
				Password: "validpass",
				Age:      30,
			}
			return ValidateAndRespond(c, input)
		})

		req := httptest.NewRequest("POST", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		app := fiber.New()
		app.Post("/test", func(c *fiber.Ctx) error {
			input := ValidationExample{
				Email:    "valid@example.com",
				Password: "short",
				Age:      30,
			}
			return ValidateAndRespond(c, input)
		})

		req := httptest.NewRequest("POST", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid age", func(t *testing.T) {
		app := fiber.New()
		app.Post("/test", func(c *fiber.Ctx) error {
			input := ValidationExample{
				Email:    "valid@example.com",
				Password: "validpass",
				Age:      200,
			}
			return ValidateAndRespond(c, input)
		})

		req := httptest.NewRequest("POST", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}
