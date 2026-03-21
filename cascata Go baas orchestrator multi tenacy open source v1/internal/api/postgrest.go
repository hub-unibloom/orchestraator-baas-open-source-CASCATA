package api

import (
	"fmt"
	"net/url"
	"strings"
)

// PostgrestQuery represents the parsed SQL components from a URL query.
type PostgrestQuery struct {
	Select string
	Where  string
	Order  string
	Limit  int
	Offset int
	Params []interface{}
}

// ParsePostgrest converts URL query parameters like ?select=id,name&status=eq.active
// into a structured SQL representation for the query builder.
func ParsePostgrest(query url.Values) (*PostgrestQuery, error) {
	pq := &PostgrestQuery{
		Select: "*",
		Limit:  1000,
		Offset: 0,
		Params: make([]interface{}, 0),
	}

	var whereClauses []string
	paramIndex := 1

	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		val := values[0]

		switch key {
		case "select":
			// very naive splitting, doesn't support nested relations like users(id, name)
			cols := strings.Split(val, ",")
			var safeCols []string
			for _, c := range cols {
				safeCols = append(safeCols, fmt.Sprintf("%q", strings.TrimSpace(c)))
			}
			pq.Select = strings.Join(safeCols, ", ")
		case "order":
			// e.g. created_at.desc
			parts := strings.Split(val, ".")
			col := parts[0]
			dir := "ASC"
			if len(parts) > 1 && strings.ToLower(parts[1]) == "desc" {
				dir = "DESC"
			}
			pq.Order = fmt.Sprintf("%q %s", col, dir)
		case "limit":
			fmt.Sscanf(val, "%d", &pq.Limit)
		case "offset":
			fmt.Sscanf(val, "%d", &pq.Offset)
		default:
			// Parse filters like ?status=eq.active
			parts := strings.SplitN(val, ".", 2)
			if len(parts) == 2 {
				operator := parts[0]
				operand := parts[1]
				sqlOp := "="
				switch operator {
				case "eq": sqlOp = "="
				case "gt": sqlOp = ">"
				case "lt": sqlOp = "<"
				case "gte": sqlOp = ">="
				case "lte": sqlOp = "<="
				case "neq": sqlOp = "!="
				case "like":
					sqlOp = "LIKE"
					operand = strings.ReplaceAll(operand, "*", "%")
				case "ilike":
					sqlOp = "ILIKE"
					operand = strings.ReplaceAll(operand, "*", "%")
				case "in":
					sqlOp = "IN"
					// handle lists (val1,val2)
					operand = strings.Trim(operand, "()")
					listVals := strings.Split(operand, ",")
					
					var placeholders []string
					for _, lv := range listVals {
						placeholders = append(placeholders, fmt.Sprintf("$%d", paramIndex))
						pq.Params = append(pq.Params, lv)
						paramIndex++
					}
					whereClauses = append(whereClauses, fmt.Sprintf("%q %s (%s)", key, sqlOp, strings.Join(placeholders, ",")))
					continue
				}

				whereClauses = append(whereClauses, fmt.Sprintf("%q %s $%d", key, sqlOp, paramIndex))
				pq.Params = append(pq.Params, operand)
				paramIndex++
			}
		}
	}

	if len(whereClauses) > 0 {
		pq.Where = strings.Join(whereClauses, " AND ")
	}

	return pq, nil
}
