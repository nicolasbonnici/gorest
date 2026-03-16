package processor

// ModelConverter defines the interface for converting between DTOs and Models.
type ModelConverter[TModel any, TCreateDTO any, TUpdateDTO any, TResponseDTO any] interface {
	// CreateDTOToModel converts a CreateDTO to a Model instance
	CreateDTOToModel(dto TCreateDTO) TModel

	// UpdateDTOToModel converts an UpdateDTO to a Model instance
	UpdateDTOToModel(dto TUpdateDTO) TModel

	// ModelToResponseDTO converts a Model to a ResponseDTO
	ModelToResponseDTO(model TModel) TResponseDTO

	// ModelsToResponseDTOs converts a slice of Models to a slice of ResponseDTOs
	ModelsToResponseDTOs(models []TModel) []TResponseDTO
}

// FuncConverter is a simple implementation of ModelConverter using functions.
// This allows for easy construction without defining a full struct type.
type FuncConverter[TModel any, TCreateDTO any, TUpdateDTO any, TResponseDTO any] struct {
	CreateToModel func(TCreateDTO) TModel
	UpdateToModel func(TUpdateDTO) TModel
	ModelToDTO    func(TModel) TResponseDTO
}

// CreateDTOToModel converts a CreateDTO to a Model using the configured function.
func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) CreateDTOToModel(dto TCreateDTO) TModel {
	return f.CreateToModel(dto)
}

// UpdateDTOToModel converts an UpdateDTO to a Model using the configured function.
func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) UpdateDTOToModel(dto TUpdateDTO) TModel {
	return f.UpdateToModel(dto)
}

// ModelToResponseDTO converts a Model to a ResponseDTO using the configured function.
func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) ModelToResponseDTO(model TModel) TResponseDTO {
	return f.ModelToDTO(model)
}

// ModelsToResponseDTOs converts a slice of Models to a slice of ResponseDTOs.
func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) ModelsToResponseDTOs(models []TModel) []TResponseDTO {
	dtos := make([]TResponseDTO, len(models))
	for i, model := range models {
		dtos[i] = f.ModelToDTO(model)
	}
	return dtos
}
