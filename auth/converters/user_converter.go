package converters

import (
	"time"

	"github.com/google/uuid"
	"github.com/nicolasbonnici/gorest/auth/dtos"
	"github.com/nicolasbonnici/gorest/auth/models"
)

type UserConverter struct{}

func (c *UserConverter) CreateDTOToModel(dto dtos.UserCreateDTO) models.User {
	password := dto.Password
	return models.User{
		ID:        uuid.New(),
		Email:     dto.Email,
		Password:  &password,
		Firstname: dto.Firstname,
		Lastname:  dto.Lastname,
		CreatedAt: time.Now(),
	}
}

func (c *UserConverter) UpdateDTOToModel(dto dtos.UserUpdateDTO) models.User {
	user := models.User{}
	if dto.Email != nil {
		user.Email = *dto.Email
	}
	if dto.Password != nil {
		user.Password = dto.Password
	}
	if dto.Firstname != nil {
		user.Firstname = *dto.Firstname
	}
	if dto.Lastname != nil {
		user.Lastname = *dto.Lastname
	}
	return user
}

func (c *UserConverter) ModelToResponseDTO(model models.User) dtos.UserResponseDTO {
	return dtos.UserResponseDTO{
		ID:        model.ID,
		Email:     model.Email,
		Firstname: model.Firstname,
		Lastname:  model.Lastname,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func (c *UserConverter) ModelsToResponseDTOs(models []models.User) []dtos.UserResponseDTO {
	dtoList := make([]dtos.UserResponseDTO, len(models))
	for i, model := range models {
		dtoList[i] = c.ModelToResponseDTO(model)
	}
	return dtoList
}
