# Быстрый старт MCP Server

Интегрируйте Cerber-Memory с Claude AI за 5 минут.

## ⚡ Установка в 3 шага

### Шаг 1: Соберите MCP сервер

```bash
cd /path/to/cerber-memory
go build -o ./bin/cerber-mcp ./cmd/mcp
```

или

```bash
bash scripts/start_mcp.sh
```

### Шаг 2: Укажите переменные окружения

Создайте файл `.env`:

```env
DB_PATH=./data/cerber_memory.db
QDRANT_HOST=localhost
QDRANT_PORT=6333
GEMINI_API_KEY=your-api-key-here
```

### Шаг 3: Добавьте в Claude Desktop config

**macOS/Linux:**
```bash
nano ~/.config/Claude/claude_desktop_config.json
```

**Windows:**
```
%APPDATA%\Claude\claude_desktop_config.json
```

Добавьте это в файл:

```json
{
  "mcpServers": {
    "cerber-memory": {
      "command": "bash",
      "args": ["-c", "cd /path/to/cerber-memory && ./scripts/start_mcp.sh"],
      "env": {
        "DB_PATH": "./data/cerber_memory.db",
        "QDRANT_HOST": "localhost",
        "QDRANT_PORT": "6333",
        "GEMINI_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

Замените `/path/to/cerber-memory` на полный путь к папке проекта.

## ✅ Проверка

Перезапустите Claude Desktop. В левой панели внизу должен появиться значок "🔌" с активной интеграцией.

## 🎯 Первые команды в Claude

После подключения попробуйте:

```
Сохрани мне новую задачу:
- Название: "Изучить MCP Protocol"
- Описание: "Разобраться как работает Model Context Protocol"
- Приоритет: high
```

Или:

```
Найди все мои pending задачи с высоким приоритетом
```

Или:

```
Покажи граф всех моих идей и их связи с проектами
```

## 🆘 Проблемы?

### "MCP Server не найден"
- Проверьте что путь в `claude_desktop_config.json` корректный
- Убедитесь что `.env` файл существует
- Перезагрузитесь Claude Desktop

### "Vector search failed"
- Проверьте что Qdrant запущен: `docker-compose up`
- Проверьте QDRANT_HOST и QDRANT_PORT в .env

### "Failed to generate embedding"
- Проверьте GEMINI_API_KEY
- Убедитесь что ключ активен в Google Cloud Console

### Смотрите логи
```bash
# Запустите MCP сервер в отдельном окне
./bin/cerber-mcp 2>mcp.log

# Смотрите логи
tail -f mcp.log
```

## 📚 Дальнейшие шаги

- Прочитайте полную документацию: [docs/MCP_INTEGRATION.md](./MCP_INTEGRATION.md)
- Изучите все доступные tools в разделе "Доступные инструменты"
- Настройте безопасность и шифрование для credentials

---

**Нужна помощь?** Проверьте документацию или посмотрите примеры использования в MCP_INTEGRATION.md
