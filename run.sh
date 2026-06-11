#!/bin/bash
set -e

# Цвета для красивого вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== CERBER Memory Docker Installer ===${NC}"

# Проверка наличия docker-compose
if ! [ -x "$(command -v docker-compose)" ] && ! [ -x "$(command -v docker)" ]; then
  echo -e "${RED}Ошибка: Docker или Docker Compose не установлены. Установите их перед запуском.${NC}" >&2
  exit 1
fi

# Проверка и создание .env
if [ ! -f .env ]; then
  echo -e "${YELLOW}Файл .env не найден. Создаю из шаблона .env-dist...${NC}"
  cp .env-dist .env
fi

# Чтение или запрос ключа Gemini
GEMINI_KEY=$(grep "GEMINI_API_KEY" .env | cut -d '=' -f2)

if [ -z "$GEMINI_KEY" ] || [ "$GEMINI_KEY" == "your_gemini_api_key_here" ]; then
  echo -e "${YELLOW}Требуется Google Gemini API Key.${NC}"
  echo -n "Введите ваш GEMINI_API_KEY: "
  read -r USER_KEY
  
  if [ -z "$USER_KEY" ]; then
    echo -e "${RED}Ошибка: Ключ не может быть пустым.${NC}"
    exit 1
  fi
  
  # Обновление ключа в .env
  sed -i "s|GEMINI_API_KEY=.*|GEMINI_API_KEY=$USER_KEY|g" .env
  echo -e "${GREEN}Ключ успешно сохранен в .env!${NC}"
fi

# Генерация случайного 32-символьного мастер-ключа шифрования, если он дефолтный
CURRENT_MASTER_KEY=$(grep "CERBER_MASTER_KEY" .env | cut -d '=' -f2)
if [ -z "$CURRENT_MASTER_KEY" ] || [ "$CURRENT_MASTER_KEY" == "default-master-key-32-characters" ]; then
  echo -e "${YELLOW}Генерирую уникальный мастер-ключ шифрования...${NC}"
  NEW_MASTER_KEY=$(LC_ALL=C tr -dc 'A-Za-z0-9!@#%^&*()-_+' < /dev/urandom | head -c 32 || true)
  sed -i "s|CERBER_MASTER_KEY=.*|CERBER_MASTER_KEY=$NEW_MASTER_KEY|g" .env
fi

echo -e "${GREEN}Сборка и запуск контейнеров...${NC}"
if command -v docker-compose &> /dev/null; then
  docker-compose up --build -d
else
  docker compose up --build -d
fi

echo -e "${GREEN}===========================================${NC}"
echo -e "${GREEN}Установка завершена!${NC}"
echo -e "Приложение запущено на: ${YELLOW}http://localhost:8080${NC}"
echo -e "Векторная база Qdrant: ${YELLOW}http://localhost:6333${NC}"
echo -e "Для просмотра логов используйте: ${YELLOW}docker-compose logs -f${NC}"
echo -e "${GREEN}===========================================${NC}"
