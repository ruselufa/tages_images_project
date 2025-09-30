#!/bin/bash

# Скрипт для тестирования лимитов конкурентности (только запросы)
# Сервер должен быть запущен отдельно

echo "=== Тест лимитов конкурентности (только запросы) ==="
echo "Убедитесь, что сервер запущен на localhost:8080"
echo ""

# Создаем тестовый файл
echo "=== Подготовка: Создание тестового файла ==="
mkdir -p file_client/files
echo "Test content for concurrency testing" > file_client/files/test.txt
echo "✅ Тестовый файл создан"
echo ""

# Загружаем файл для тестирования скачивания
echo "=== Загрузка файла для тестирования ==="
cd file_client
UPLOAD_RESULT=$(go run cmd/client/main.go -batch "upload files/test.txt" 2>&1)
if echo "$UPLOAD_RESULT" | grep -q "File ID"; then
    FILE_ID=$(echo "$UPLOAD_RESULT" | grep "File ID" | cut -d' ' -f3)
    echo "✅ Файл загружен успешно, ID: $FILE_ID"
else
    echo "❌ Ошибка загрузки файла: $UPLOAD_RESULT"
    exit 1
fi
cd ..
echo ""

# Тест 1: 15 одновременных скачиваний (лимит: 10)
echo "=== Тест 1: 15 одновременных скачиваний (лимит: 10) ==="
echo "Ожидаем: 10 успешных, 5 с ошибкой"
echo ""

success=0
error=0

for i in {1..15}; do
    (
        result=$(cd file_client && go run cmd/client/main.go -batch "download $FILE_ID /tmp/test_$i.txt" 2>&1)
        if echo "$result" | grep -q "Downloaded"; then
            echo "Скачивание $i: ✅ Успешно"
            echo "success" >> /tmp/download_results.txt
        elif echo "$result" | grep -q "Too many conc"; then
            echo "Скачивание $i: ❌ Ошибка (лимит)"
            echo "error" >> /tmp/download_results.txt
        else
            echo "Скачивание $i: ❌ Другая ошибка"
            echo "error" >> /tmp/download_results.txt
        fi
    ) &
done

wait

# Подсчитываем результаты скачивания
if [ -f /tmp/download_results.txt ]; then
    success=$(grep -c "success" /tmp/download_results.txt)
    error=$(grep -c "error" /tmp/download_results.txt)
    rm -f /tmp/download_results.txt
fi

echo ""
echo "Результат скачивания: $success успешных, $error с ошибкой"
if [ $success -eq 10 ] && [ $error -eq 5 ]; then
    echo "✅ Лимит скачивания работает корректно"
    DOWNLOAD_OK=true
else
    echo "❌ Лимит скачивания работает некорректно"
    DOWNLOAD_OK=false
fi
echo ""

# Тест 2: 110 одновременных запросов списка (лимит: 100)
echo "=== Тест 2: 110 одновременных запросов списка (лимит: 100) ==="
echo "Ожидаем: 100 успешных, 10 с ошибкой"
echo ""

success=0
error=0

for i in {1..110}; do
    (
        result=$(cd file_client && go run cmd/client/main.go -batch list 2>&1)
        if echo "$result" | grep -q "Found"; then
            echo "Список $i: ✅ Успешно"
            echo "success" >> /tmp/list_results.txt
        elif echo "$result" | grep -q "Too many conc"; then
            echo "Список $i: ❌ Ошибка (лимит)"
            echo "error" >> /tmp/list_results.txt
        else
            echo "Список $i: ❌ Другая ошибка"
            echo "error" >> /tmp/list_results.txt
        fi
    ) &
done

wait

# Подсчитываем результаты списка
if [ -f /tmp/list_results.txt ]; then
    success=$(grep -c "success" /tmp/list_results.txt)
    error=$(grep -c "error" /tmp/list_results.txt)
    rm -f /tmp/list_results.txt
fi

echo ""
echo "Результат списка: $success успешных, $error с ошибкой"
if [ $success -eq 100 ] && [ $error -eq 10 ]; then
    echo "✅ Лимит списка работает корректно"
    LIST_OK=true
else
    echo "❌ Лимит списка работает некорректно"
    LIST_OK=false
fi
echo ""

# Итоговый результат
echo "=== Итоговый результат ==="
if [ "$DOWNLOAD_OK" = true ] && [ "$LIST_OK" = true ]; then
    echo "🎉 ВСЕ ЛИМИТЫ РАБОТАЮТ КОРРЕКТНО!"
    echo "✅ Лимит скачивания: 10 одновременных запросов"
    echo "✅ Лимит списка: 100 одновременных запросов"
    echo ""
    echo "Каждый клиент имеет свои лимиты:"
    echo "- Клиент 1: 10 загрузок/скачиваний + 100 списков"
    echo "- Клиент 2: 10 загрузок/скачиваний + 100 списков"
    echo "- И так далее..."
else
    echo "❌ Некоторые лимиты работают некорректно"
    echo "Проверьте настройки сервера"
fi

echo ""
echo "=== Тест завершен ==="
