package processor

type ModelConverter[TModel any, TCreateDTO any, TUpdateDTO any, TResponseDTO any] interface {
	CreateDTOToModel(dto TCreateDTO) TModel
	UpdateDTOToModel(dto TUpdateDTO) TModel
	ModelToResponseDTO(model TModel) TResponseDTO
	ModelsToResponseDTOs(models []TModel) []TResponseDTO
}

type FuncConverter[TModel any, TCreateDTO any, TUpdateDTO any, TResponseDTO any] struct {
	CreateToModel func(TCreateDTO) TModel
	UpdateToModel func(TUpdateDTO) TModel
	ModelToDTO    func(TModel) TResponseDTO
}

func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) CreateDTOToModel(dto TCreateDTO) TModel {
	return f.CreateToModel(dto)
}

func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) UpdateDTOToModel(dto TUpdateDTO) TModel {
	return f.UpdateToModel(dto)
}

func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) ModelToResponseDTO(model TModel) TResponseDTO {
	return f.ModelToDTO(model)
}

func (f *FuncConverter[TModel, TCreateDTO, TUpdateDTO, TResponseDTO]) ModelsToResponseDTOs(models []TModel) []TResponseDTO {
	dtos := make([]TResponseDTO, len(models))
	for i, model := range models {
		dtos[i] = f.ModelToDTO(model)
	}
	return dtos
}
