# Serializer & JSON-LD Support

GoREST provides automatic semantic web support with JSON-LD (JSON for Linked Data), offering content negotiation between standard JSON and JSON-LD formats.

## Content Negotiation

The API automatically detects the requested format via the `Accept` header:

### Regular JSON

```bash
curl -H "Accept: application/json" http://localhost:3000/todos/123
```

Response:
```json
{
  "id": "abc-123",
  "user_id": "def-456",
  "title": "Buy groceries"
}
```

### JSON-LD

```bash
curl -H "Accept: application/ld+json" http://localhost:3000/todos/123
```

Response:
```json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/abc-123",
  "id": "abc-123",
  "user": "/users/def-456",
  "title": "Buy groceries"
}
```

## JSON-LD Features

### Semantic Context

- **@context**: Links to vocabulary (Schema.org by default)
- **@type**: Resource type (e.g., `TodoDTO`, `UserDTO`)
- **@id**: Unique IRI identifier for the resource

### Clean Relation Names

Foreign keys automatically convert to semantic relation names:
- `userId` → `user` with IRI value `/users/def-456`
- `categoryId` → `category` with IRI value `/categories/cat-789`

This makes relationships more intuitive and follows Linked Data principles.

## Relation Expansion

Combine JSON-LD with relation expansion to get full nested objects:

```bash
curl -H "Accept: application/ld+json" \
     "http://localhost:3000/todos/123?expand[]=user"
```

Response:
```json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/abc-123",
  "id": "abc-123",
  "user": {
    "@type": "UserDTO",
    "@id": "/users/def-456",
    "id": "def-456",
    "name": "Alice",
    "email": "alice@example.com"
  },
  "title": "Buy groceries"
}
```

## Collections

JSON-LD collections use Hydra vocabulary for pagination:

```bash
curl -H "Accept: application/ld+json" http://localhost:3000/todos?limit=10
```

Response:
```json
{
  "@context": "https://schema.org/",
  "@type": "hydra:Collection",
  "hydra:totalItems": 42,
  "hydra:member": [
    {
      "@type": "TodoDTO",
      "@id": "/todos/abc-123",
      "id": "abc-123",
      "title": "Buy groceries"
    }
  ],
  "hydra:view": {
    "@type": "hydra:PartialCollectionView",
    "hydra:first": "/todos?limit=10&offset=0",
    "hydra:next": "/todos?limit=10&offset=10",
    "hydra:last": "/todos?limit=10&offset=40"
  }
}
```

## Use Cases

### Machine-Readable APIs

JSON-LD makes your API machine-readable and semantically rich, enabling:
- Automatic API clients
- Data integration across systems
- Knowledge graph construction
- SEO improvements

### Semantic Search

Search engines and semantic web crawlers can better understand your data structure.

### Standard Vocabularies

By using Schema.org or other standard vocabularies, your data becomes interoperable with other systems using the same vocabulary.

## Related Documentation

- [EXPAND_USAGE.md](EXPAND_USAGE.md) - Detailed guide on expanding IRI relations to objects
- [FILTERING.md](../FILTERING.md) - Query filtering and ordering
- [DTOS.md](../DTOS.md) - DTO field control and visibility
