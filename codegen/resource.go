package codegen

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func generateRouteWithAuth(method, path, handler, resource, httpMethod string, authCfg *AuthConfig) string {
	requiresAuth := authCfg != nil && authCfg.RequiresAuth(resource, httpMethod)

	if requiresAuth {
		return fmt.Sprintf(`
	if authMiddleware != nil {
		router.%s("/%s", authMiddleware, %s)
	} else {
		router.%s("/%s", %s)
	}
`, method, path, handler, method, path, handler)
	}

	return fmt.Sprintf(`	router.%s("/%s", %s)
`, method, path, handler)
}

func generateResourceForStruct(apiDir string, structName string, authCfg *AuthConfig) {
	resourceFile := filepath.Join(apiDir, strings.ToLower(structName)+".go")

	cfg, _ := LoadConfig()
	projectRoot, _ := findProjectRoot()
	modelsDir := cfg.Codegen.Output.Models
	if !filepath.IsAbs(modelsDir) {
		modelsDir = filepath.Join(projectRoot, modelsDir)
	}
	modelPath := filepath.Join(modelsDir, strings.ToLower(structName)+".go")
	fields := extractStructFields(modelPath, structName)

	code := generateResourceFromModel(structName, fields, authCfg)
	if err := os.WriteFile(resourceFile, []byte(code), 0644); err != nil {
		log.Fatalf("failed to write resource for %s: %v", structName, err)
	}
	log.Printf("🧩 Generated API resource for model: %s → %s", structName, resourceFile)
}

func generateResourceFromModel(structName string, fields []StructField, authCfg *AuthConfig) string {
	resourceName := strings.ToLower(structName)
	lowerStructName := strings.ToLower(structName)
	pluralResourceName := Pluralize(resourceName)

	// Generate routes with conditional auth middleware
	listRoute := generateRouteWithAuth("Get", pluralResourceName, "res.List", pluralResourceName, "GET", authCfg)
	getRoute := generateRouteWithAuth("Get", pluralResourceName+"/:id", "res.Get", pluralResourceName, "GET", authCfg)
	postRoute := generateRouteWithAuth("Post", pluralResourceName, "res.Create", pluralResourceName, "POST", authCfg)
	putRoute := generateRouteWithAuth("Put", pluralResourceName+"/:id", "res.Update", pluralResourceName, "PUT", authCfg)
	deleteRoute := generateRouteWithAuth("Delete", pluralResourceName+"/:id", "res.Delete", pluralResourceName, "DELETE", authCfg)

	needsAuthContext := authCfg != nil && (authCfg.RequiresAuth(pluralResourceName, "GET") ||
		authCfg.RequiresAuth(pluralResourceName, "POST") ||
		authCfg.RequiresAuth(pluralResourceName, "PUT") ||
		authCfg.RequiresAuth(pluralResourceName, "DELETE"))

	routesSignature := "router fiber.Router, db database.Database, paginationLimit, paginationMaxLimit int, pluginRegistry *plugin.PluginRegistry"

	authMiddlewareSetup := ""
	if needsAuthContext {
		authMiddlewareSetup = `
	var authMiddleware fiber.Handler
	if authPlugin, ok := pluginRegistry.Get("auth"); ok {
		authMiddleware = authPlugin.Handler()
	}
`
	}

	hasUserIdField := false
	for _, field := range fields {
		if field.Name == "UserId" {
			hasUserIdField = true
			break
		}
	}

	contextFunc := "c.Context()"
	if needsAuthContext || hasUserIdField {
		contextFunc = "auth.Context(c)"
	}

	userIdAutoPopulate := ""
	if hasUserIdField {
		userIdAutoPopulate = `
	// Auto-populate user_id from authenticated user
	if user := auth.GetAuthenticatedUser(c); user != nil {
		item.UserId = &user.UserID
	}
`
	}

	conversionFuncs := generateConversionFunctions(structName, fields)

	var allowedFieldsList []string
	for _, field := range fields {
		if field.DBTag != "" {
			if field.DBTag == "password" || field.DTOTag == "write" {
				continue
			}
			allowedFieldsList = append(allowedFieldsList, fmt.Sprintf(`"%s"`, field.DBTag))
		}
	}
	allowedFieldsStr := strings.Join(allowedFieldsList, ", ")

	projectRoot, _ := findProjectRoot()
	hookFilePath := filepath.Join(projectRoot, "hooks", lowerStructName+".go")
	hasHooks := false
	if _, err := os.Stat(hookFilePath); err == nil {
		hasHooks = true
	}

	moduleName := getModuleName()
	cfg, _ := LoadConfig()
	_, _ = findProjectRoot()

	modelsImport := moduleName
	dtosImport := moduleName

	if cfg.Codegen.Output.Models != "models" && cfg.Codegen.Output.Models != "" {
		modelsPath := strings.TrimPrefix(cfg.Codegen.Output.Models, "./")
		modelsImport = moduleName + "/" + strings.ReplaceAll(modelsPath, string(filepath.Separator), "/")
	}
	if cfg.Codegen.Output.DTOs != "dtos" && cfg.Codegen.Output.DTOs != "" {
		dtosPath := strings.TrimPrefix(cfg.Codegen.Output.DTOs, "./")
		dtosImport = moduleName + "/" + strings.ReplaceAll(dtosPath, string(filepath.Separator), "/")
	}

	modelsImport = strings.TrimSuffix(modelsImport, "/models") + "/models"
	dtosImport = strings.TrimSuffix(dtosImport, "/dtos") + "/dtos"

	importsSection := fmt.Sprintf(`import (
	"net/url"

	"%s"
	"%s"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/filter"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/pagination"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/response"`, dtosImport, modelsImport)

	if needsAuthContext || hasUserIdField {
		importsSection += `
	auth "github.com/nicolasbonnici/gorest-auth"`
	}
	if hasHooks {
		hooksImport := moduleName + "/hooks"
		importsSection += fmt.Sprintf(`
	"%s"`, hooksImport)
	}
	importsSection += `
)`

	crudInit := fmt.Sprintf("crud.New[models.%s](db)", structName)
	if hasHooks {
		crudInit = fmt.Sprintf("crud.NewWithHooks[models.%s](db, &hooks.%sHooks{})", structName, structName)
	}

	return fmt.Sprintf(`// Code generated by GoREST. DO NOT EDIT.

package resources

%s

type %sResource struct {
	DB                 database.Database
	CRUD               *crud.CRUD[models.%s]
	PaginationLimit    int
	PaginationMaxLimit int
}

func Register%sRoutes(%s) {
	res := &%sResource{
		DB:                 db,
		CRUD:               %s,
		PaginationLimit:    paginationLimit,
		PaginationMaxLimit: paginationMaxLimit,
	}
%s%s%s%s%s%s
}

%s

// List %s
// @Summary List %s
// @Tags %s
// @Produce json,application/ld+json
// @Success 200 {object} pagination.HydraCollection
// @Router /%s [get]
func (r *%sResource) List(c *fiber.Ctx) error {
	limit := pagination.ParseIntQuery(c, "limit", r.PaginationLimit, r.PaginationMaxLimit)
	page := pagination.ParseIntQuery(c, "page", 1, 10000)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	includeCount := c.Query("count", "true") != "false"

	allowedFields := []string{%s}

	queryParams := make(url.Values)
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams.Add(string(key), string(value))
	})

	// Parse filters into conditions
	filters := filter.NewFilterSet(allowedFields, r.DB.Dialect())
	if err := filters.ParseFromQuery(queryParams); err != nil {
		return pagination.SendPaginatedError(c, 400, err.Error())
	}
	conditions := filters.Conditions()

	// Parse ordering into OrderBy clauses
	ordering := filter.NewOrderSet(allowedFields)
	if err := ordering.ParseFromQuery(queryParams); err != nil {
		return pagination.SendPaginatedError(c, 400, err.Error())
	}
	orderClauses := ordering.OrderClauses()

	// Convert filter.OrderClause to crud.OrderByClause
	orderBy := make([]crud.OrderByClause, len(orderClauses))
	for i, oc := range orderClauses {
		orderBy[i] = crud.OrderByClause{
			Column:    oc.Column,
			Direction: oc.Direction,
		}
	}

	result, err := r.CRUD.GetAllPaginated(%s, crud.PaginationOptions{
		Limit:        limit,
		Offset:       offset,
		IncludeCount: includeCount,
		Conditions:   conditions,
		OrderBy:      orderBy,
	})
	if err != nil {
		return pagination.SendPaginatedError(c, 500, err.Error())
	}

	dtoItems := make([]dtos.%sDTO, len(result.Items))
	for i, item := range result.Items {
		dtoItems[i] = modelTo%sDTO(item)
	}

	return pagination.SendHydraCollection(c, dtoItems, result.Total, limit, page, r.PaginationLimit)
}

// Get %s by ID
// @Summary Get %s
// @Tags %s
// @Produce json,application/ld+json
// @Param id path int true "ID"
// @Success 200 {object} dtos.%sDTO
// @Router /%s/{id} [get]
func (r *%sResource) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	item, err := r.CRUD.GetByID(%s, id)
	if err != nil {
		if crud.IsInvalidIDError(err) {
			return response.SendError(c, 400, err.Error())
		}
		return response.SendError(c, 404, "Not found")
	}

	dto := modelTo%sDTO(*item)
	return response.SendFormatted(c,200, dto)
}

// Create %s
// @Summary Create %s
// @Tags %s
// @Accept json
// @Produce json,application/ld+json
// @Param input body dtos.%sCreateDTO true "New %s"
// @Success 201 {object} dtos.%sDTO
// @Router /%s [post]
func (r *%sResource) Create(c *fiber.Ctx) error {
	var createDTO dtos.%sCreateDTO
	if err := c.BodyParser(&createDTO); err != nil {
		logger.Log.Error("Failed to parse request body", "error", err, "path", c.Path())
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	item := %sCreateDTOToModel(createDTO)
%s
	ctx := %s
	if err := r.CRUD.Create(ctx, item); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	created, err := r.CRUD.GetByID(ctx, item.Id)
	if err != nil {
		dto := modelTo%sDTO(item)
		return response.SendFormatted(c,201, dto)
	}

	dto := modelTo%sDTO(*created)
	return response.SendFormatted(c,201, dto)
}

// Update %s
// @Summary Update %s
// @Tags %s
// @Accept json
// @Produce json,application/ld+json
// @Param id path int true "ID"
// @Param input body dtos.%sUpdateDTO true "Updated %s"
// @Success 200 {object} dtos.%sDTO
// @Router /%s/{id} [put]
func (r *%sResource) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var updateDTO dtos.%sUpdateDTO
	if err := c.BodyParser(&updateDTO); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	item := %sUpdateDTOToModel(updateDTO)
%s
	if err := r.CRUD.Update(%s, id, item); err != nil {
		if crud.IsInvalidIDError(err) {
			return response.SendError(c, 400, err.Error())
		}
		if crud.IsNotFoundError(err) {
			return response.SendError(c, 404, "Not found")
		}
		return response.SendError(c, 500, err.Error())
	}

	dto := modelTo%sDTO(item)
	return response.SendFormatted(c,200, dto)
}

// Delete %s
// @Summary Delete %s
// @Tags %s
// @Param id path int true "ID"
// @Success 204
// @Router /%s/{id} [delete]
func (r *%sResource) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := r.CRUD.Delete(%s, id); err != nil {
		if crud.IsInvalidIDError(err) {
			return response.SendError(c, 400, err.Error())
		}
		if crud.IsNotFoundError(err) {
			return response.SendError(c, 404, "Not found")
		}
		return response.SendError(c, 500, err.Error())
	}
	return c.SendStatus(204)
}
`,
		importsSection,
		structName, structName,
		structName, routesSignature, structName,
		crudInit,
		authMiddlewareSetup,
		listRoute, getRoute, postRoute, putRoute, deleteRoute,
		conversionFuncs,
		structName, structName, structName, pluralResourceName, structName,
		allowedFieldsStr,
		contextFunc,
		structName, structName,
		structName, structName, structName, structName, pluralResourceName, structName,
		contextFunc,
		structName,
		structName, structName, structName, structName, structName, structName, pluralResourceName, structName,
		structName,
		lowerStructName,
		userIdAutoPopulate,
		contextFunc,
		structName, structName,
		structName, structName, structName, structName, structName, structName, pluralResourceName, structName,
		structName,
		lowerStructName,
		userIdAutoPopulate,
		contextFunc,
		structName,
		structName, structName, structName, pluralResourceName, structName,
		contextFunc)
}

func generateConversionFunctions(structName string, fields []StructField) string {
	var modelToDTOFields strings.Builder
	for _, field := range fields {
		if field.DTOTag == "-" || field.DTOTag == "write" {
			continue
		}
		modelToDTOFields.WriteString(fmt.Sprintf("\t\t%s: m.%s,\n", field.Name, field.Name))
	}

	var createDTOToModelFields strings.Builder
	for _, field := range fields {
		dbTag := strings.ToLower(field.DBTag)
		if dbTag != FieldID && dbTag != FieldCreatedAt && dbTag != FieldUpdatedAt {
			if field.DTOTag == "-" || field.DTOTag == "read" {
				continue
			}
			createDTOToModelFields.WriteString(fmt.Sprintf("\t\t%s: dto.%s,\n", field.Name, field.Name))
		}
	}

	var updateDTOToModelFields strings.Builder
	for _, field := range fields {
		dbTag := strings.ToLower(field.DBTag)
		if dbTag != FieldID && dbTag != FieldCreatedAt && dbTag != FieldUpdatedAt {
			if field.DTOTag == "-" || field.DTOTag == "read" {
				continue
			}
			updateDTOToModelFields.WriteString(fmt.Sprintf("\t\t%s: dto.%s,\n", field.Name, field.Name))
		}
	}

	lowerStructName := strings.ToLower(structName)
	return fmt.Sprintf(`func modelTo%sDTO(m models.%s) dtos.%sDTO {
	return dtos.%sDTO{
%s	}
}

func %sCreateDTOToModel(dto dtos.%sCreateDTO) models.%s {
	return models.%s{
%s	}
}

func %sUpdateDTOToModel(dto dtos.%sUpdateDTO) models.%s {
	return models.%s{
%s	}
}
`, structName, structName, structName, structName, modelToDTOFields.String(),
		lowerStructName, structName, structName, structName, createDTOToModelFields.String(),
		lowerStructName, structName, structName, structName, updateDTOToModelFields.String())
}
