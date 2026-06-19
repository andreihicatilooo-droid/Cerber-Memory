# Архитектура MCP Server для Cerber-Memory

Полная архитектура системы интеграции между Cerber-Memory и Claude AI через Model Context Protocol.

## 📐 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Claude AI / Brain                        │
│              (Использует MCP tools для памяти)               │
└──────────────────────────┬──────────────────────────────────┘
                           │
                    MCP Protocol (JSON-RPC)
                    Transport: STDIO
                           │
┌──────────────────────────▼──────────────────────────────────┐
│         Cerber Memory MCP Server (Go Binary)                │
├──────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Router     │  │  Handlers    │  │ Middleware   │      │
│  ├──────────────┤  ├──────────────┤  ├──────────────┤      │
│  │ JSON-RPC     │  │ 6 Tools      │  │ Validation   │      │
│  │ Parser       │  │ Processors   │  │ Error Hdlng  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Search      │  │   Memory     │  │ Credentials  │      │
│  │  Service     │  │   Service    │  │  Service     │      │
│  ├──────────────┤  ├──────────────┤  ├──────────────┤      │
│  │ Semantic     │  │ CRUD Ops     │  │ Encryption   │      │
│  │ Search       │  │ Categorize   │  │ Audit Log    │      │
│  │ Vector Ops   │  │ Tags         │  │ Validation   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │         Data Access Layer (Repository)            │    │
│  └────────────────────────────────────────────────────┘    │
└──────────────┬──────────────────────────┬──────────────────┘
               │                          │
        ┌──────▼────────┐          ┌─────▼─────────┐
        │    SQLite     │          │    Qdrant     │
        │   Relational  │          │  Vector DB    │
        ├───────────────┤          ├───────────────┤
        │  Tables:      │          │  Collection:  │
        │ - tasks       │          │ cerber_memory │
        │ - ideas       │          │ (768-dim)     │
        │ - notes       │          │               │
        │ - documents   │          │ Payloads:     │
        │ - projects    │          │ - entity_type │
        │ - core_memory │          │ - entity_id   │
        │   (creds)     │          │ - metadata    │
        └───────────────┘          └───────────────┘
```

## 🔄 Request Flow

```
1. Claude AI отправляет JSON-RPC запрос
   ↓
2. MCP Server получает на stdin
   ↓
3. Router парсит метод и параметры
   ↓
4. Выбирает нужный Handler
   ↓
5. Handler валидирует параметры
   ↓
6. Обращается к Service Layer
   ↓
7. Service Layer использует Repository для CRUD
   ↓
8. Repository работает с SQLite + Qdrant
   ↓
9. Возвращает результат обратно
   ↓
10. Handler форматирует JSON-RPC ответ
    ↓
11. Выводит на stdout для Claude
```

## 🛠️ Components

### MCP Server (internal/mcp/server.go)

**Основные функции:**
- JSON-RPC 2.0 парсер
- Роутинг запросов к обработчикам
- Управление lifecycle сервера
- Логирование запросов/ответов

**Обработчики:**
- `handleInitialize` - инициализация протокола
- `handleToolsList` - описание доступных tools
- `handleSemanticSearch` - векторный поиск
- `handleSaveMemory` - создание памяти
- `handleQueryStructured` - фильтрованные запросы
- `handleCredentialsGet/Set` - работа с паролями
- `handleGetGraph` - граф знаний

### Service Layer (internal/mvc/models/mcp_helpers.go)

**CreateTask / CreateIdea / CreateNote / CreateDocument**
- Создание сущностей с валидацией
- Автоматическое enqueueing для индексации
- Добавление тегов и категоризация

**QueryTasks / QueryIdeas / QueryNotes / QueryDocuments / QueryProjects**
- Фильтрованный поиск с SQL
- Сортировка и лимит
- Полнотекстовый поиск

**GetCredential / SetCredential**
- Безопасное хранение credentials
- Шифрование (в планах - AES-256)
- Логирование доступа

### Repository Layer (internal/mvc/models/db.go)

**Основные функции:**
- InitDB - инициализация схемы БД
- CreateTables - создание таблиц
- CRUD операции для каждой сущности
- Управление связями (universal_links)

## 📊 Data Models

### SQLite Schema

```
tasks
├── id (INTEGER PRIMARY KEY)
├── title TEXT
├── description TEXT
├── status (pending|active|completed|archived)
├── priority (low|medium|high)
├── category TEXT
├── created_at TIMESTAMP

ideas
├── id (INTEGER PRIMARY KEY)
├── title TEXT
├── description TEXT
├── status (raw|elaborated|implemented|rejected)
├── priority (low|medium|high)
├── category TEXT
├── created_at TIMESTAMP

notes, documents, projects...

core_memory (для credentials)
├── key TEXT UNIQUE
├── content BLOB (encrypted)
├── category = 'credentials'

universal_links (граф)
├── from_type TEXT
├── from_id INTEGER
├── to_type TEXT
├── to_id INTEGER
├── relation_type TEXT
```

### Qdrant Payload

```json
{
  "entity_type": "task|idea|note|document|project",
  "entity_id": 123,
  "title": "Task title",
  "description": "Full text",
  "category": "feature|bug|optimization",
  "status": "pending|active",
  "priority": "low|medium|high",
  "tags": ["tag1", "tag2"],
  "created_at": "2024-06-19T10:00:00Z"
}
```

## 🔐 Security Architecture

### Authentication & Authorization
- Пока нет встроенной auth (Claude->MCP запросы)
- Рекомендуется: SSH туннель для удаленного доступа
- Или: HTTP wrapper с OAuth2

### Data Protection
- SQLite на локальной машине
- Encryption for credentials (планируется)
- No plaintext storage of sensitive data

### Audit & Logging
- Все операции логируются в stderr
- Запланирована база логов с timestamp
- Отслеживание доступа к credentials

## 🚀 Scalability Considerations

### Current Limitations
- SQLite не масштабируется для >>1M records
- Все запросы синхронные
- Нет кэширования результатов
- Batch операции не поддерживаются

### Future Improvements
```
Phase 1 (weeks 1-2):
- [ ] Connection pooling
- [ ] Query result caching
- [ ] Batch operations support
- [ ] Full-text search indexes

Phase 2 (weeks 3-4):
- [ ] PostgreSQL support (drop-in replacement for SQLite)
- [ ] Redis caching layer
- [ ] Async request processing
- [ ] Streaming responses for large datasets

Phase 3 (weeks 5-6):
- [ ] Horizontal scaling
- [ ] Load balancing
- [ ] Rate limiting
- [ ] API versioning
```

## 🔄 Integration Points

### With Claude AI
- JSON-RPC over STDIO
- Can be extended with streaming
- Tool schemas auto-discovered

### With Qdrant
- HTTP REST API
- Vector operations (upsert, search)
- Metadata filtering via payloads

### With Gemini API
- Text embeddings generation
- LLM parsing (in pipeline layer)
- Semantic analysis

### With SQLite
- Direct CRUD operations
- Transaction support
- Foreign keys for relationships

## 📈 Performance Metrics

### Expected Performance
| Operation | Speed | Notes |
|-----------|-------|-------|
| Save item | <100ms | Sync to SQLite + queue for vector |
| Semantic search | 100-500ms | Depends on Qdrant response time |
| Structured query | 10-100ms | SQLite index performance |
| Credentials get | <50ms | Simple SELECT |

### Optimization Opportunities
- Index on `status`, `priority`, `category`
- Qdrant collection replication
- Response streaming for large result sets

## 🧪 Testing Strategy

### Unit Tests
- Service layer functions
- Validation logic
- Error handling

### Integration Tests
- MCP Server communication
- SQLite transactions
- Qdrant connectivity

### Load Tests
- Concurrent requests
- Large result sets
- Memory usage

## 📝 API Versioning

Current: v1 (embedded in server version)

Future:
```
- Semantic versioning: 1.0.0
- Breaking changes → major version bump
- New tools → minor version bump
- Bug fixes → patch version bump
```

## 🔗 Related Documentation

- [MCP_INTEGRATION.md](./MCP_INTEGRATION.md) - User guide
- [QUICKSTART_MCP.md](./QUICKSTART_MCP.md) - Setup instructions
- [examples/mcp_requests.json](../examples/mcp_requests.json) - Request examples

---

**Version:** 1.0.0  
**Last Updated:** 2024-06-19  
**Maintainers:** Cerber-Memory Team
