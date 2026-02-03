# Filtering & Ordering

GoREST provides powerful query filtering and ordering capabilities through URL query parameters.

## Filters

### Equality

```bash
GET /todos?status=active
```

### Multiple Values (OR)

```bash
GET /todos?status[]=active&status[]=pending
```

### Comparison Operators

```bash
GET /todos?priority[gte]=5
GET /todos?priority[lt]=10
```

**Supported operators:**
- `gte` - Greater than or equal
- `gt` - Greater than
- `lte` - Less than or equal
- `lt` - Less than

### Text Search

```bash
GET /todos?title[like]=meeting
GET /todos?title[ilike]=MEETING  # Case-insensitive
```

### Combining Filters (AND)

```bash
GET /todos?status=active&priority[gte]=7
```

## Ordering

### Single Field

```bash
GET /todos?order[createdAt]=desc
```

### Multiple Fields

```bash
GET /todos?order[priority]=desc&order[createdAt]=asc
```

**Sort directions:**
- `asc` - Ascending order
- `desc` - Descending order

## Combining Features

You can combine filters, ordering, pagination, and expansion:

```bash
GET /todos?status=active&priority[gte]=5&order[createdAt]=desc&limit=10&expand[]=user
```
