# Работа с переменными окружения

## Общая идея

Конфигурация приложения может загружаться из:

1. ENV файла

2. Переменных окружения процесса (OS ENV)

3. Значений по умолчанию в коде

---

## Приоритет источников

OS ENV > ENV FILE

Если значение есть и в ENV и в файле — используется ENV.

---

## Быстрый старт

### Локально

Создай файл .env:

APP_NAME=service
PORT=8097
LOG_LEVEL=debug
TIME_ZONE=Asia/Almaty

В коде:

app := app.NewApp()

SDK попробует загрузить:
.env  
OS ENV

---

### Production / Vault

Vault обычно монтирует:

/vault/secrets/.env

Можно явно указать:

app := app.NewApp().SetEnvFiles("/vault/secrets/.env")

---

### Передача ENV при запуске контейнера

docker run -e PORT=9000 -e LOG_LEVEL=info my-service

или

docker run --env-file .env my-service

---

## Указание кастомных путей

app := app.NewApp().SetEnvFiles(
"/vault/secrets/.env",
".env",
)

Используется первый найденный файл.

---

## Поведение

Есть только ENV → используется ENV  
Есть только файл → используется файл  
Есть ENV и файл → ENV override  
Нет ничего → нулевые значения

---

## Работа с Vault

SDK не подключается к Vault напрямую.

Vault доставляет файл через:
- sidecar injector
- init container
- CSI volume

SDK просто читает файл:

/vault/secrets/.env

---

## Best Practices

- Использовать fallback путей
- Не копировать .env при старте контейнера
- Делать validation обязательных переменных
- ENV должен иметь приоритет над файлом
