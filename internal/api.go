package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupAPI(app fiber.Router, db *pgxpool.Pool, tables map[string]TableSchema, jwtSecret string) {
	for _, t := range tables {
		registerTableRoutes(app, db, t, tables)
	}
}

func registerTableRoutes(app fiber.Router, db *pgxpool.Pool, t TableSchema, tables map[string]TableSchema) {
	base := "/" + t.TableName

	// GET /table
	app.Get(base, func(c *fiber.Ctx) error {
		query := fmt.Sprintf("SELECT * FROM %s", t.TableName)
		args := []interface{}{}
		conds := []string{}
		i := 1

		for _, col := range t.Columns {
			if val := c.Query(col); val != "" {
				conds = append(conds, fmt.Sprintf("%s=$%d", col, i))
				args = append(args, val)
				i++
			}
		}
		if len(conds) > 0 {
			query += " WHERE " + strings.Join(conds, " AND ")
		}

		limit := c.Query("limit", "100")
		offset := c.Query("offset", "0")
		query += fmt.Sprintf(" LIMIT %s OFFSET %s", limit, offset)

		rows, err := db.Query(context.Background(), query, args...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		results := []map[string]interface{}{}
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}

			rowMap := map[string]interface{}{}
			for i, col := range rows.FieldDescriptions() {
				rowMap[string(col.Name)] = values[i]
			}

			// Handle expand relations
			exp := c.Query("expand")
			if exp != "" {
				for _, r := range t.Relations {
					if !strings.Contains(exp, r.ParentTable) {
						continue
					}
					val := rowMap[r.ChildColumn]
					if val != nil {
						parentQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s=$1", r.ParentTable, r.ParentColumn)
						fieldNames, fieldValues, err := scanRowDynamic(db, parentQuery, val)
						if err != nil {
							continue
						}
						parentMap := map[string]interface{}{}
						for j, name := range fieldNames {
							parentMap[name] = fieldValues[j]
						}
						rowMap[r.ParentTable] = parentMap
					}
				}
			}

			results = append(results, rowMap)
		}
		return c.JSON(results)
	})

	// GET /table/:id
	app.Get(base+"/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		query := fmt.Sprintf("SELECT * FROM %s WHERE id=$1", t.TableName)
		fieldNames, fieldValues, err := scanRowDynamic(db, query, id)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		rowMap := map[string]interface{}{}
		for i, name := range fieldNames {
			rowMap[name] = fieldValues[i]
		}
		return c.JSON(rowMap)
	})

	// POST /table
	app.Post(base, func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		cols := []string{}
		placeholders := []string{}
		values := []interface{}{}
		i := 1
		for _, col := range t.Columns {
			if col == "id" {
				continue
			}
			if v, ok := body[col]; ok {
				cols = append(cols, col)
				placeholders = append(placeholders, fmt.Sprintf("$%d", i))
				values = append(values, v)
				i++
			}
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *",
			t.TableName, strings.Join(cols, ","), strings.Join(placeholders, ","))

		fieldNames, fieldValues, err := scanRowDynamic(db, query, values...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		rowMap := map[string]interface{}{}
		for i, name := range fieldNames {
			rowMap[name] = fieldValues[i]
		}
		return c.JSON(rowMap)
	})

	// PUT /table/:id
	app.Put(base+"/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		setClauses := []string{}
		values := []interface{}{}
		i := 1
		for _, col := range t.Columns {
			if col != "id" {
				if v, ok := body[col]; ok {
					setClauses = append(setClauses, fmt.Sprintf("%s=$%d", col, i))
					values = append(values, v)
					i++
				}
			}
		}
		values = append(values, id)
		query := fmt.Sprintf("UPDATE %s SET %s WHERE id=$%d RETURNING *",
			t.TableName, strings.Join(setClauses, ","), i)

		fieldNames, fieldValues, err := scanRowDynamic(db, query, values...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		rowMap := map[string]interface{}{}
		for i, name := range fieldNames {
			rowMap[name] = fieldValues[i]
		}
		return c.JSON(rowMap)
	})

	// DELETE /table/:id
	app.Delete(base+"/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		query := fmt.Sprintf("DELETE FROM %s WHERE id=$1", t.TableName)
		_, err := db.Exec(context.Background(), query, id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(204)
	})
}

// scanRowDynamic scans a single row dynamically from a query
func scanRowDynamic(db *pgxpool.Pool, query string, args ...interface{}) ([]string, []interface{}, error) {
	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil, fmt.Errorf("no row found")
	}

	values, err := rows.Values()
	if err != nil {
		return nil, nil, err
	}

	colNames := []string{}
	for _, f := range rows.FieldDescriptions() {
		colNames = append(colNames, string(f.Name))
	}

	return colNames, values, nil
}
