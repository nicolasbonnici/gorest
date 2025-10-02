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
			return c.Status(500).JSON(err.Error())
		}
		defer rows.Close()

		cols := rows.FieldDescriptions()
		results := []map[string]interface{}{}

		for rows.Next() {
			values, _ := rows.Values()
			rowMap := map[string]interface{}{}
			for j, col := range cols {
				rowMap[string(col.Name)] = values[j]
			}

			exp := c.Query("expand")
			if exp != "" {
				for _, r := range t.Relations {
					if r.ChildColumn == "id" || !strings.Contains(exp, r.ParentTable) {
						continue
					}
					val := rowMap[r.ChildColumn]
					if val != nil {
						parentQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s=$1", r.ParentTable, r.ParentColumn)
						row := db.QueryRow(context.Background(), parentQuery, val)
						parentVals, _ := row.Values()
						parentCols := row.FieldDescriptions()
						parentMap := map[string]interface{}{}
						for k, col := range parentCols {
							parentMap[string(col.Name)] = parentVals[k]
						}
						rowMap[r.ParentTable] = parentMap
					}
				}
			}

			results = append(results, rowMap)
		}
		return c.JSON(results)
	})

	app.Get(base+"/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		query := fmt.Sprintf("SELECT * FROM %s WHERE id=$1", t.TableName)
		row := db.QueryRow(context.Background(), query, id)

		vals, err := row.Values()
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}

		cols := row.FieldDescriptions()
		rowMap := map[string]interface{}{}
		for i, col := range cols {
			rowMap[string(col.Name)] = vals[i]
		}
		return c.JSON(rowMap)
	})

	app.Post(base, func(c *fiber.Ctx) error {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(400).JSON(err.Error())
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
		row := db.QueryRow(context.Background(), query, values...)
		vals, err := row.Values()
		if err != nil {
			return c.Status(500).JSON(err.Error())
		}
		colsDesc := row.FieldDescriptions()
		rowMap := map[string]interface{}{}
		for i, col := range colsDesc {
			rowMap[string(col.Name)] = vals[i]
		}
		return c.JSON(rowMap)
	})

	app.Put(base+"/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			return c.Status(400).JSON(err.Error())
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
		row := db.QueryRow(context.Background(), query, values...)
		vals, err := row.Values()
		if err != nil {
			return c.Status(500).JSON(err.Error())
		}
		cols := row.FieldDescriptions()
		rowMap := map[string]interface{}{}
		for i, col := range cols {
			rowMap[string(col.Name)] = vals[i]
		}
		return c.JSON(rowMap)
	})

	app.Delete(base+"/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		query := fmt.Sprintf("DELETE FROM %s WHERE id=$1", t.TableName)
		_, err := db.Exec(context.Background(), query, id)
		if err != nil {
			return c.Status(500).JSON(err.Error())
		}
		return c.SendStatus(204)
	})

	for _, r := range t.Relations {
		parent := r.ParentTable
		child := r.ChildTable
		route := fmt.Sprintf("/%s/:id/%s", parent, child)
		app.Get(route, func(c *fiber.Ctx) error {
			id := c.Params("id")
			query := fmt.Sprintf("SELECT * FROM %s WHERE %s=$1", child, r.ChildColumn)
			rows, err := db.Query(context.Background(), query, id)
			if err != nil {
				return c.Status(500).JSON(err.Error())
			}
			defer rows.Close()
			results := []map[string]interface{}{}
			cols := rows.FieldDescriptions()
			for rows.Next() {
				values, _ := rows.Values()
				rowMap := map[string]interface{}{}
				for i, col := range cols {
					rowMap[string(col.Name)] = values[i]
				}
				results = append(results, rowMap)
			}
			return c.JSON(results)
		})
	}
}
