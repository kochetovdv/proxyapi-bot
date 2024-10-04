package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strings"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    yaml "gopkg.in/yaml.v2"

    "log/slog"
)

// Структура для хранения настроек из config.yaml
type Config struct {
    ApiURL           string   `yaml:"api_url"`
    APIKey           string   `yaml:"api_key"`
    TelegramBotToken string   `yaml:"telegram_bot_token"`
    FilesPath        string   `yaml:"files_path"`
    Name             string   `yaml:"name"`
    Instructions     string   `yaml:"instructions"`
    Model            string   `yaml:"model"`
    Tools            []string `yaml:"tools"`
}

var (
    config Config
)

// Функция для чтения конфигурационного файла
func loadConfig(configPath string) error {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return fmt.Errorf("Ошибка чтения файла конфигурации: %v", err)
    }

    err = yaml.Unmarshal(data, &config)
    if err != nil {
        return fmt.Errorf("Ошибка разбора файла конфигурации: %v", err)
    }

    return nil
}

type AssistantCreateRequest struct {
    Name         string `json:"name"`
    Instructions string `json:"instructions"`
    Model        string `json:"model"`
    Tools        []Tool `json:"tools"`
}

type Tool struct {
    Type string `json:"type"`
}

type AssistantCreateResponse struct {
    ID string `json:"id"`
}

type VectorStoreCreateResponse struct {
    ID string `json:"id"`
}

// Функция для создания ассистента с поддержкой File Search
func createAssistant() (string, error) {
    // Преобразуем список инструментов в нужный формат
    tools := []Tool{}
    for _, toolType := range config.Tools {
        tools = append(tools, Tool{Type: toolType})
    }

    requestBody := AssistantCreateRequest{
        Name:         config.Name,
        Instructions: config.Instructions,
        Model:        config.Model,
        Tools:        tools,
    }

    reqBody, err := json.Marshal(requestBody)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", config.ApiURL+"assistants", bytes.NewBuffer(reqBody))
    if err != nil {
        return "", err
    }

    req.Header.Set("Authorization", "Bearer "+config.APIKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("OpenAI-Beta", "assistants=v2")

    // Логирование запроса
    slog.Debug("Создание ассистента: отправка запроса", "url", req.URL)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    slog.Debug("Получен ответ при создании ассистента", "body", string(body))

    var assistantResponse AssistantCreateResponse
    if err := json.Unmarshal(body, &assistantResponse); err != nil {
        return "", err
    }

    slog.Info("Ассистент создан", "assistant_id", assistantResponse.ID)
    return assistantResponse.ID, nil
}

// Функция для загрузки файла с корректным использованием multipart/form-data
func uploadFile(filePath string) (string, error) {
    // Логирование чтения файла
    slog.Debug("Чтение файла для загрузки", "file_path", filePath)

    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    var b bytes.Buffer
    w := multipart.NewWriter(&b)

    // Добавляем файл в запрос
    fw, err := w.CreateFormFile("file", filepath.Base(filePath))
    if err != nil {
        return "", err
    }
    _, err = io.Copy(fw, file)
    if err != nil {
        return "", err
    }

    // Добавляем параметр 'purpose' в запрос
    err = w.WriteField("purpose", "assistants")
    if err != nil {
        return "", err
    }

    w.Close()

    req, err := http.NewRequest("POST", config.ApiURL+"files", &b)
    if err != nil {
        return "", err
    }

    req.Header.Set("Authorization", "Bearer "+config.APIKey)
    req.Header.Set("Content-Type", w.FormDataContentType())
    req.Header.Set("OpenAI-Beta", "assistants=v2")

    slog.Debug("Загрузка файла", "url", req.URL, "file_name", filepath.Base(filePath))

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    if resp.StatusCode != http.StatusOK {
        slog.Error("Ошибка загрузки файла", "status_code", resp.StatusCode, "body", string(body))
        return "", fmt.Errorf("Ошибка загрузки файла: %s", string(body))
    }

    slog.Debug("Файл успешно загружен", "file_name", filepath.Base(filePath))

    // Получаем file_id
    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return "", err
    }

    fileID, ok := response["id"].(string)
    if !ok {
        slog.Error("Не удалось получить file_id для файла", "body", string(body))
        return "", fmt.Errorf("Не удалось получить file_id для файла %s", filePath)
    }

    return fileID, nil
}

// Функция для создания Vector Store и загрузки файлов
func createVectorStoreAndUploadFiles() (string, error) {
    // Создаем Vector Store
    req, err := http.NewRequest("POST", config.ApiURL+"vector_stores", nil)
    if err != nil {
        return "", err
    }

    req.Header.Set("Authorization", "Bearer "+config.APIKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("OpenAI-Beta", "assistants=v2")

    slog.Debug("Создание Vector Store", "url", req.URL)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    slog.Debug("Получен ответ при создании Vector Store", "body", string(body))

    var vectorStoreResponse VectorStoreCreateResponse
    if err := json.Unmarshal(body, &vectorStoreResponse); err != nil {
        return "", err
    }

    vectorStoreID := vectorStoreResponse.ID
    slog.Info("Vector Store создан", "vector_store_id", vectorStoreID)

    // Загрузим файлы из директории, указанной в конфигурации
    files, err := os.ReadDir(config.FilesPath)
    if err != nil {
        return "", err
    }

    for _, file := range files {
        if !file.IsDir() {
            filePath := filepath.Join(config.FilesPath, file.Name())

            // Загрузим файл и получим его file_id
            fileID, err := uploadFile(filePath)
            if err != nil {
                slog.Error("Ошибка загрузки файла", "file_name", file.Name(), "error", err)
                continue
            }

            // Регистрируем файл в Vector Store
            if err := registerFileInVectorStore(vectorStoreID, fileID); err != nil {
                slog.Error("Ошибка регистрации файла в Vector Store", "file_name", file.Name(), "error", err)
                continue
            }
        }
    }

    return vectorStoreID, nil
}

// Функция для регистрации файла в Vector Store
func registerFileInVectorStore(vectorStoreID, fileID string) error {
    requestBody := map[string]string{
        "file_id": fileID,
    }

    reqBody, err := json.Marshal(requestBody)
    if err != nil {
        return fmt.Errorf("Ошибка формирования тела запроса для регистрации файла: %v", err)
    }

    req, err := http.NewRequest("POST", config.ApiURL+"vector_stores/"+vectorStoreID+"/files", bytes.NewBuffer(reqBody))
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer "+config.APIKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("OpenAI-Beta", "assistants=v2")

    slog.Debug("Регистрация файла в Vector Store", "vector_store_id", vectorStoreID, "file_id", fileID)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusOK {
        slog.Error("Ошибка регистрации файла", "status_code", resp.StatusCode, "body", string(body))
        return fmt.Errorf("Ошибка регистрации файла: %s", string(body))
    }

    slog.Info("Файл успешно зарегистрирован в Vector Store", "file_id", fileID)
    return nil
}

// Функция для обновления ассистента с Vector Store
func updateAssistantWithVectorStore(assistantID, vectorStoreID string) error {
    updateBody := map[string]interface{}{
        "tool_resources": map[string]interface{}{
            "file_search": map[string]interface{}{
                "vector_store_ids": []string{vectorStoreID},
            },
        },
    }

    reqBody, err := json.Marshal(updateBody)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", config.ApiURL+"assistants/"+assistantID, bytes.NewBuffer(reqBody))
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer "+config.APIKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("OpenAI-Beta", "assistants=v2")

    slog.Debug("Обновление ассистента", "assistant_id", assistantID)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusOK {
        slog.Error("Ошибка обновления ассистента", "status_code", resp.StatusCode, "body", string(body))
        return fmt.Errorf("Ошибка обновления ассистента: %s", string(body))
    }

    slog.Info("Ассистент успешно обновлен", "assistant_id", assistantID)
    return nil
}

func listenToSSEStream(resp *http.Response) (string, error) {
    defer resp.Body.Close()

    reader := bufio.NewReader(resp.Body)
    var finalMessage string

    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                break
            }
            return "", fmt.Errorf("Ошибка чтения события: %v", err)
        }

        line = strings.TrimSpace(line)
        if len(line) == 0 {
            continue
        }

        // Проверяем, начинается ли строка с 'data: '
        if strings.HasPrefix(line, "data: ") {
            eventData := line[6:] // Убираем "data: "

            if eventData == "[DONE]" {
                slog.Debug("Ответ полностью получен")
                break
            }

            var event map[string]interface{}
            if err := json.Unmarshal([]byte(eventData), &event); err != nil {
                slog.Error("Ошибка разбора события", "error", err)
                continue
            }

            // Если это событие завершения сообщения (thread.message.delta)
            if obj, ok := event["object"].(string); ok && obj == "thread.message.delta" {
                if delta, ok := event["delta"].(map[string]interface{}); ok {
                    if content, ok := delta["content"].([]interface{}); ok {
                        for _, part := range content {
                            if textPart, ok := part.(map[string]interface{}); ok {
                                if text, ok := textPart["text"].(map[string]interface{}); ok {
                                    if value, ok := text["value"].(string); ok {
                                        finalMessage += value
                                    }
                                }
                            }
                        }
                    }
                }
            }

            // Проверка на завершение сообщения
            if obj, ok := event["object"].(string); ok && obj == "thread.message.completed" {
                slog.Debug("Сообщение ассистента завершено")
                break
            }
        }
    }

    slog.Debug("Собранное сообщение от ассистента", "message", finalMessage)

    if finalMessage == "" {
        return "", fmt.Errorf("Пустой ответ от ассистента")
    }

    return finalMessage, nil
}

// createAndRunAssistantWithStreaming создаёт поток и запускает ассистента с обработкой SSE
func createAndRunAssistantWithStreaming(assistantID, query, vectorStoreID string) (string, error) {
    requestBody := map[string]interface{}{
        "assistant_id": assistantID,
        "thread": map[string]interface{}{
            "messages": []map[string]interface{}{
                {"role": "user", "content": query},
            },
        },
        "tool_resources": map[string]interface{}{
            "file_search": map[string]interface{}{
                "vector_store_ids": []string{vectorStoreID},
            },
        },
        "temperature": 1.0,
        "top_p":       1.0,
        "stream":      true, // Активируем поток
    }

    reqBody, err := json.Marshal(requestBody)
    if err != nil {
        return "", fmt.Errorf("Ошибка создания тела запроса: %v", err)
    }

    req, err := http.NewRequest("POST", config.ApiURL+"threads/runs", bytes.NewBuffer(reqBody))
    if err != nil {
        return "", fmt.Errorf("Ошибка создания HTTP-запроса: %v", err)
    }

    req.Header.Set("Authorization", "Bearer "+config.APIKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("OpenAI-Beta", "assistants=v2")

    slog.Debug("Отправка запроса к ассистенту", "assistant_id", assistantID, "query", query)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("Ошибка выполнения HTTP-запроса: %v", err)
    }

    return listenToSSEStream(resp)
}

// handleTelegramUpdates обрабатывает запросы Telegram и передает их ассистенту
func handleTelegramUpdates(bot *tgbotapi.BotAPI, assistantID, vectorStoreID string) {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message != nil && update.Message.Text != "" {
            // Создаем локальные копии переменных
            localUpdate := update
            query := localUpdate.Message.Text
            slog.Info("Получен запрос от пользователя", "user_id", localUpdate.Message.From.ID, "query", query)

            // Обрабатываем каждый запрос в отдельной горутине
            go func(update tgbotapi.Update, query string) {
                response, err := createAndRunAssistantWithStreaming(assistantID, query, vectorStoreID)
                if err != nil {
                    slog.Error("Ошибка выполнения запроса ассистентом", "error", err)
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка обработки запроса.")
                    bot.Send(msg)
                    return
                }

                if response == "" {
                    slog.Error("Получен пустой ответ от ассистента")
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ассистент не смог предоставить ответ.")
                    bot.Send(msg)
                    return
                }

                msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
                bot.Send(msg)
                slog.Info("Ответ отправлен пользователю", "user_id", update.Message.From.ID)

            }(localUpdate, query) // Передаем параметры в горутину
        }
    }
}

func main() {
    // Настройка логгера
    handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo, // Установите нужный уровень логирования
    })
    slog.SetDefault(slog.New(handler))

    // Загружаем конфигурацию
    err := loadConfig("config.yaml")
    if err != nil {
        slog.Error("Ошибка загрузки конфигурации", "error", err)
        os.Exit(1)
    }

    // Инициализируем Telegram бота
    bot, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
    if err != nil {
        slog.Error("Ошибка инициализации Telegram бота", "error", err)
        os.Exit(1)
    }
    bot.Debug = false // Отключаем отладку самого бота
    slog.Info("Telegram бот авторизован", "username", bot.Self.UserName)

    // Создание ассистента
    assistantID, err := createAssistant()
    if err != nil {
        slog.Error("Ошибка создания ассистента", "error", err)
        os.Exit(1)
    }

    // Создание Vector Store и загрузка файлов
    vectorStoreID, err := createVectorStoreAndUploadFiles()
    if err != nil {
        slog.Error("Ошибка создания Vector Store и загрузки файлов", "error", err)
        os.Exit(1)
    }

    // Привязка Vector Store к ассистенту
    if err := updateAssistantWithVectorStore(assistantID, vectorStoreID); err != nil {
        slog.Error("Ошибка обновления ассистента", "error", err)
        os.Exit(1)
    }

    slog.Info("Ассистент готов к работе", "assistant_id", assistantID)

    // Обработка запросов от Telegram пользователей
    handleTelegramUpdates(bot, assistantID, vectorStoreID)
}
