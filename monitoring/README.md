# Мониторинг MIT Service

## Обзор

Стек мониторинга включает:
- **Prometheus** - сбор метрик
- **Grafana** - визуализация и дашборды  
- **cAdvisor** - метрики Docker контейнеров

## Запуск

```bash
docker-compose up -d
```

## Доступ к сервисам

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **MIT Service**: http://localhost:8080
- **cAdvisor**: http://localhost:8081

## Дашборд Grafana

Дашборд "MIT Service Dashboard" автоматически загружается и включает:

### Основные метрики HTTP
- **HTTP Requests per Second** - количество запросов в секунду
- **Active HTTP Connections** - активные соединения  
- **HTTP Error Rate** - частота ошибок HTTP
- **HTTP Request Duration Percentiles** - процентили времени ответа (50%, 95%)

### Метрики задач (Inbox Worker)
- **Tasks per Second** - задачи в секунду
- **Queue Depth** - глубина очереди

### Системные метрики приложения
- **Memory Usage** - использование памяти приложением
- **Goroutines** - количество горутин
- **Service Uptime** - время работы сервиса

### Дополнительные метрики приложения
- **HTTP Error Rate** - частота HTTP ошибок  
- **HTTP Request Duration Percentiles** - процентили времени ответа
- **Service Uptime** - время работы сервиса

**Примечание**: Метрики Docker контейнеров (cAdvisor) могут не работать на некоторых системах (например, macOS) из-за ограничений доступа к Docker daemon. В таком случае можно отключить cAdvisor в docker-compose.yml.

## Метрики приложения

Приложение экспортирует метрики на `/metrics` endpoint:

### HTTP метрики
- `mit_service_http_requests_total` - общее количество HTTP запросов
- `mit_service_http_request_duration_seconds` - длительность HTTP запросов
- `mit_service_http_active_connections` - активные HTTP соединения

### Метрики задач
- `mit_service_tasks_total` - общее количество обработанных задач
- `mit_service_task_duration_seconds` - длительность обработки задач
- `mit_service_queue_depth` - текущая глубина очереди

### Системные метрики
- `mit_service_goroutines` - количество горутин
- `mit_service_memory_usage_bytes` - использование памяти
- `mit_service_uptime_seconds` - время работы сервиса

## Настройка алертов

Для добавления алертов можно создать файлы правил в `prometheus/rules/` и обновить конфигурацию Prometheus.

Пример правила для высокого времени ответа:

```yaml
groups:
  - name: mit_service
    rules:
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(mit_service_http_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        annotations:
          summary: "Высокое время ответа MIT Service"
          description: "95-й процентиль времени ответа превышает 500ms"
```

## Трассировка проблем

1. **Медленные запросы**: проверьте панель "HTTP Request Duration Percentiles"
2. **Ошибки**: панель "HTTP Error Rate" 
3. **Очередь задач**: "Queue Depth" - если растет, возможно нужно увеличить количество воркеров
4. **Память**: проверьте обе панели памяти - приложения и контейнера
5. **CPU**: панель "Docker Container CPU Usage"

## Конфигурация

### Prometheus
- Конфигурация: `monitoring/prometheus.yml`
- Интервал сбора: 5s для MIT Service, 10s для cAdvisor

### Grafana
- Источники данных: `monitoring/grafana/provisioning/datasources/`
- Дашборды: `monitoring/grafana/provisioning/dashboards/`

## Масштабирование мониторинга

Для продакшена рекомендуется:
1. Настроить persistent storage для Prometheus
2. Добавить алерт-менеджер
3. Настроить ротацию логов
4. Использовать внешний Grafana для HA
