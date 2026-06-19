# MCP Server Integration для Cerber-Memory

Полная интеграция Cerber-Memory с Claude AI через Model Context Protocol (MCP).

## 📋 Описание

MCP Server предоставляет Claude AI полный доступ к системе управления памятью Cerber-Memory через набор инструментов для:
- 🔍 Семантического поиска по всем данным
- 💾 Сохранения новых задач, идей, документов
- 📊 Структурированных запросов с фильтрацией
- 🔐 Безопасного хранения и доступа к паролям/ключам API
- 📈 Получения графа знаний (mindmap)

## 🚀 Установка и запуск

### 1. Требования
- Go 1.22+
- SQLite3
- Qdrant (для векторного поиска)
- Google Gemini API ключ

### 2. Сборка MCP сервера

```bash
cd /path/to/cerber-memory
go build -o ./bin/cerber-mcp ./cmd/mcp
```

или использовать готовый скрипт:
```bash
bash scripts/start_mcp.sh
```

### 3. Подключение к Claude Desktop

#### Опция A: На локальной машине

Отредактируйте `~/.config/Claude/claude_desktop_config.json` (macOS/Linux) или
`%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "cerber-memory": {
      "command": "bash",
      "args": [
        "-c",
        "cd /path/to/cerber-memory && ./scripts/start_mcp.sh"
      ],
      "env": {
        "DB_PATH": "./data/cerber_memory.db",
        "QDRANT_HOST": "localhost",
        "QDRANT_PORT": "6333",
        "GEMINI_API_KEY": "your-gemini-api-key"
      }
    }
  }
}
```

#### Опция B: На удаленном сервере

Если запускаете на сервере, используйте SSH:

```json
{
  "mcpServers": {
    "cerber-memory": {
      "command": "ssh",
      "args": [
        "user@server.com",
        "cd /path/to/cerber-memory && ./scripts/start_mcp.sh"
      ]
    }
  }
}
```

### 4. Переменные окружения

```bash
# Database
DB_PATH=./data/cerber_memory.db

# Qdrant Vector DB
QDRANT_HOST=localhost
QDRANT_PORT=6333

# Google Gemini API
GEMINI_API_KEY=your-api-key-here
GEMINI_EMBEDDING_MODEL=text-embedding-004
```

## 🛠️ Доступные инструменты

### 1. `memory_semantic_search`

Поиск по всем данным используя семантическое подобие.

**Параметры:**
```json
{
  "query": "найди все задачи про API",
  "limit": 5
}
```

**Ответ:**
```json
{
  "query": "найди все задачи про API",
  "count": 2,
  "results": [
    {
      "id": 1,
      "type": "task",
      "title": "Реализовать REST API",
      "content": "Создать REST API для управления памятью...",
      "similarity": 0.92,
      "payload": { ... }
    }
  ]
}
```

### 2. `memory_save`

Сохранить новый элемент памяти (задача, идея, заметка, документ, проект).

**Параметры:**
```json
{
  "type": "task",
  "title": "Интегрировать MCP",
  "content": "Добавить MCP сервер для Claude интеграции",
  "category": "integration",
  "priority": "high",
  "tags": ["claude", "mcp", "urgent"]
}
```

**Типы:**
- `task` - Задача
- `idea` - Идея
- `note` - Заметка
- `document` - Документ
- `project` - Проект

**Приоритеты:** `low`, `medium`, `high`

**Ответ:**
```json
{
  "id": 123,
  "type": "task",
  "status": "created",
  "queued_index": true
}
```

### 3. `memory_query_structured`

Запрос памяти с фильтрами по статусу, категории, приоритету.

**Параметры:**
```json
{
  "entity_type": "task",
  "status": "pending",
  "priority": "high",
  "category": "feature",
  "limit": 10
}
```

**Статусы:** `pending`, `active`, `completed`, `archived`

**Ответ:**
```json
{
  "entity_type": "task",
  "filters": {
    "status": "pending",
    "category": "feature",
    "priority": "high"
  },
  "results": [
    {
      "id": 1,
      "title": "Реализовать MCP",
      "description": "...",
      "status": "pending",
      "priority": "high",
      "category": "feature",
      "created_at": "2024-06-19T10:00:00Z"
    }
  ]
}
```

### 4. `credentials_get`

Получить сохраненный пароль или API ключ.

⚠️ **Используется для доступа, логируется для аудита!**

**Параметры:**
```json
{
  "key": "openai_api_key"
}
```

**Ответ:**
```json
{
  "key": "openai_api_key",
  "value": "sk-..."
}
```

### 5. `credentials_set`

Сохранить пароль или API ключ в защищенном виде.

**Параметры:**
```json
{
  "key": "openai_api_key",
  "value": "sk-proj-...",
  "category": "api_key"
}
```

**Категории:** `api_key`, `password`, `token`, `database`

**Ответ:**
```json
{
  "id": 45,
  "key": "openai_api_key",
  "status": "saved",
  "category": "api_key"
}
```

### 6. `memory_get_graph`

Получить полный граф знаний (всех связей между элементами).

**Параметры:** нет

**Ответ:**
```json
{
  "nodes": [
    { "id": "task_1", "type": "task", "title": "Задача 1" },
    { "id": "idea_5", "type": "idea", "title": "Идея 5" }
  ],
  "edges": [
    { "source": "task_1", "target": "idea_5", "label": "relates_to" }
  ],
  "count": {
    "nodes": 150,
    "edges": 280
  }
}
```

## 💬 Примеры использования с Claude

### Пример 1: Создание задачи и поиск связанных

```
Ты: "Мне нужно реализовать API для управления памятью. 
     Поищи связанные задачи и идеи, потом создай новую задачу."

Claude использует:
1. memory_semantic_search с query="API для управления памятью"
2. memory_save для создания новой задачи
3. memory_query_structured с фильтром status=pending
```

### Пример 2: Поиск всех ключей API

```
Ты: "Какие API ключи у нас сохранены?"

Claude использует:
1. memory_query_structured с entity_type="core_memory", category="credentials"
2. credentials_get для каждого найденного ключа
```

### Пример 3: Анализ знаний

```
Ты: "Покажи мне граф всех моих идей и как они связаны с проектами"

Claude использует:
1. memory_get_graph
2. Визуализирует связи между ideas и projects
```

## 🔐 Безопасность

### Хранение паролей

Пароли хранятся в таблице `core_memory` с категорией `credentials`:
- Зашифрованы (AES-256 в перспективе)
- Изолированы от обычной памяти
- Логируются все обращения

### Аудит доступа

Каждый доступ к credentials логируется с:
- Timestamp
- Ключ доступа
- IP Claude сессии (если доступно)
- Статус (успешно/ошибка)

## 📊 Архитектура

```
┌─────────────────┐
│   Claude AI     │
├─────────────────┤
│  MCP Protocol   │ (JSON-RPC 2.0)
├─────────────────┤
│  MCP Server     │ (Go)
├─────────────────┤
│     SQLite      │  Qdrant    │
│   Структура     │  Поиск     │
└─────────────────┴────────────┘
```

## 🐛 Отладка

### Логи MCP сервера

Логи выводятся в stderr:
```bash
./bin/cerber-mcp 2>mcp.log
```

### Проверка соединения

```bash
# Проверить что MCP запущен
echo '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}' | \
  ./bin/cerber-mcp
```

### Распространенные ошибки

| Ошибка | Решение |
|--------|---------|
| "Method not found" | Проверить имя метода в tools/list |
| "vector search failed" | Проверить что Qdrant запущен и доступен |
| "credential not found" | Сохранить credential перед обращением |
| "database connection" | Проверить DB_PATH и права доступа |

## 📚 Дальнейшее развитие

- [ ] Реальное AES-256 шифрование для credentials
- [ ] Аудит лог базу данных
- [ ] Rate limiting для API
- [ ] Batch операции
- [ ] Streaming результатов для больших наборов
- [ ] Full-text search с индексацией
- [ ] Расширенная фильтрация (date ranges, tags)
- [ ] Export/Import функции

## 📞 Поддержка

Если возникли проблемы:
1. Проверьте логи MCP сервера
2. Убедитесь что все зависимости установлены
3. Проверьте переменные окружения
4. Попробуйте перестартовать Claude Desktop

---

**Версия:** 1.0.0  
**Последнее обновление:** 2024-06-19
