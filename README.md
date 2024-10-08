# AI Ассистент на базе OpenAI Assistants API
Данный проект демонстрирует создание кастомного AI-ассистента с использованием OpenAI Assistants API и его интеграцию с Telegram-ботом. Ассистент специализируется на материалах, загружаемых в него при создании, и может отвечать на вопросы пользователей, следуя инструкциям и используя загруженные документы.

*По умолчанию используется proxy-сервис ProxyApi.ru для доступа к OpenAI API.
Вы также можете использовать любой другой proxy-сервис или сам OpenAI API. Для этого потребуется в файле config.yaml изменить URL, а в качестве API-KEY использовать ключ, генерируемый в личном кабинете вашего сервиса.*

## Особенности
- Создание кастомного AI-ассистента на базе модели GPT-4 или любой другой модели от OpenAI, поддерживающей создание ассистентов.
- Интеграция с Telegram для взаимодействия с пользователями.
- Загрузка и использование пользовательских файлов для расширения знаний ассистента.
- Обработка пользовательских запросов в реальном времени с использованием SSE (Server-Sent Events).
- Сохранение контекста.
- Многопоточная обработка сообщений.

## Требования
- **Go** версии 1.16 или выше.
- **OpenAI API ключ** с доступом к Assistants API.
- **Telegram Bot API ключ** для взаимодействия с Telegram.
- **Директория с файлами** для загрузки в ассистента.
- **Библиотеки Go**:
    - github.com/go-telegram-bot-api/telegram-bot-api/v5
    - gopkg.in/yaml.v2

## Установка
1.	**Клонируйте репозиторий:**
```bash
git clone https://github.com/kochetovdv/proxyapi-bot.git
cd proxyapi-bot
```
2. **Установите необходимые зависимости:**
```bash
go get github.com/go-telegram-bot-api/telegram-bot-api/v5
go get gopkg.in/yaml.v2
```

## Настройка
1.	**Создайте файл конфигурации *config.yaml* или отредактируйте существующий файл в корневой директории проекта:**
```yaml
api_url: https://api.proxyapi.ru/openai/v1/ # URL доступа к API. Замените на https://api.openai.com/v1/ для доступа напрямую к OpenAI
api_key: [YOUR-API-KEY]
telegram_bot_token: [YOUR-TELEGRAM-BOT-TOKEN]
files_path: upload # Путь к директории с файлами
name: [YOUR-ASSISTANT-NAME] # Название ассистента
instructions: [YOUR-ASSISTANT-INSTRUCTIONS] # Инструкции для ассистента
model: gpt-4-turbo # Модель для ассистента
tools:
  - file_search
max_context_messages: 10  # Максимальное количество сообщений в контексте
```
**Важно**: 
- Замените api_url на "https://api.openai.com/v1/" при наличии личного кабинета в OpenAI или на ссылку иного proxy-сервиса.
- Замените YOUR-API-KEY и YOUR-TELEGRAM-BOT-TOKEN на ваши реальные ключи. Никогда не публикуйте эти ключи в открытом доступе.
- В качестве модели можно использовать иную от OpenAI, в которой доступно создание ассистентов.

2.	**Создайте директорию для загрузки файлов:**
```bash
mkdir upload
```
Поместите в эту директорию все необходимые документы для ассистента.

## Запуск
1.	**Соберите и запустите приложение:**
```bash
go build -o assistant
./assistant
```
2.	**Убедитесь, что ассистент успешно запущен:**
В терминале должны появиться сообщения о создании ассистента, загрузке файлов и готовности к работе.

## Использование
**Взаимодействие с ассистентом через Telegram:**
1.	Найдите вашего бота в Telegram по его имени, указанному при создании через @BotFather.
2.	Начните диалог и отправьте вопрос.
3.	Ассистент обработает ваш запрос и предоставит ответ, ссылаясь на источники информации.

## Структура проекта
- **main.go**: основной файл с кодом приложения.
- **config.yaml**: файл конфигурации с настройками API и ассистента.
- **upload/**: директория с файлами для загрузки в ассистента (по умолчанию).
- **go.mod** и **go.sum**: файлы управления зависимостями Go.

## Лицензия
Данный проект распространяется под лицензией MIT.
