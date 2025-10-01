#!/bin/bash

# MIT Service Performance Monitor
# Usage: ./monitor.sh [interval_seconds] [server_url]

INTERVAL=${1:-5}
SERVER_URL=${2:-"http://localhost:8080"}

echo "MIT Service Performance Monitor"
echo "Server: $SERVER_URL"
echo "Update interval: ${INTERVAL}s"
echo "Press Ctrl+C to stop"
echo ""

while true; do
    clear
    echo "=== MIT Service Performance Dashboard ==="
    echo "Time: $(date)"
    echo ""
    
    # Get performance data
    PERF_DATA=$(curl -s "${SERVER_URL}/performance" 2>/dev/null)
    
    if [ $? -eq 0 ] && [ ! -z "$PERF_DATA" ]; then
        # Parse health status
        HEALTH_STATUS=$(echo "$PERF_DATA" | jq -r '.health.status // "unknown"')
        HEALTH_SCORE=$(echo "$PERF_DATA" | jq -r '.health.score // 0')
        
        # Color-code health status
        case $HEALTH_STATUS in
            "healthy")
                STATUS_COLOR="\033[32m" # Green
                ;;
            "warning") 
                STATUS_COLOR="\033[33m" # Yellow
                ;;
            "critical")
                STATUS_COLOR="\033[31m" # Red
                ;;
            *)
                STATUS_COLOR="\033[37m" # White
                ;;
        esac
        
        echo -e "Health Status: ${STATUS_COLOR}$(echo "$HEALTH_STATUS" | tr '[:lower:]' '[:upper:]') (${HEALTH_SCORE}/100)\033[0m"
        
        # Show issues if any
        ISSUES=$(echo "$PERF_DATA" | jq -r '.health.issues[]?' 2>/dev/null)
        if [ ! -z "$ISSUES" ]; then
            echo -e "\033[31mIssues:\033[0m"
            echo "$ISSUES" | while read issue; do
                echo "  - $issue"
            done
        fi
        
        echo ""
        
        # HTTP Metrics
        echo "HTTP Metrics:"
        RPS=$(echo "$PERF_DATA" | jq -r '.metrics.requests_per_second // 0')
        AVG_RESPONSE=$(echo "$PERF_DATA" | jq -r '.metrics.avg_response_time_ms // 0')
        TOTAL_REQUESTS=$(echo "$PERF_DATA" | jq -r '.metrics.total_requests // 0')
        SUCCESS_REQUESTS=$(echo "$PERF_DATA" | jq -r '.metrics.successful_requests // 0')
        FAILED_REQUESTS=$(echo "$PERF_DATA" | jq -r '.metrics.failed_requests // 0')
        ACTIVE_CONN=$(echo "$PERF_DATA" | jq -r '.metrics.active_connections // 0')
        
        printf "  RPS: %.1f req/s | Avg Response: %.1f ms | Active Connections: %s\n" "$RPS" "$AVG_RESPONSE" "$ACTIVE_CONN"
        printf "  Total: %s | Success: %s | Failed: %s\n" "$TOTAL_REQUESTS" "$SUCCESS_REQUESTS" "$FAILED_REQUESTS"
        
        # Calculate success rate
        if [ "$TOTAL_REQUESTS" -gt 0 ]; then
            SUCCESS_RATE=$(echo "scale=2; $SUCCESS_REQUESTS * 100 / $TOTAL_REQUESTS" | bc 2>/dev/null || echo "0")
            printf "  Success Rate: %.2f%%\n" "$SUCCESS_RATE"
        fi
        
        echo ""
        
        # Inbox Metrics
        echo "Inbox Metrics:"
        TPS=$(echo "$PERF_DATA" | jq -r '.metrics.tasks_per_second // 0')
        AVG_TASK_TIME=$(echo "$PERF_DATA" | jq -r '.metrics.avg_task_time_ms // 0')
        QUEUE_DEPTH=$(echo "$PERF_DATA" | jq -r '.metrics.queue_depth // 0')
        MAX_QUEUE=$(echo "$PERF_DATA" | jq -r '.metrics.max_queue_depth // 0')
        TOTAL_TASKS=$(echo "$PERF_DATA" | jq -r '.metrics.total_tasks // 0')
        COMPLETED_TASKS=$(echo "$PERF_DATA" | jq -r '.metrics.completed_tasks // 0')
        
        printf "  TPS: %.1f tasks/s | Avg Task Time: %.1f ms | Queue: %s/%s\n" "$TPS" "$AVG_TASK_TIME" "$QUEUE_DEPTH" "$MAX_QUEUE"
        printf "  Total Tasks: %s | Completed: %s\n" "$TOTAL_TASKS" "$COMPLETED_TASKS"
        
        echo ""
        
        # System Metrics
        echo "System Metrics:"
        MEMORY_MB=$(echo "$PERF_DATA" | jq -r '.metrics.memory_usage_mb // 0')
        GOROUTINES=$(echo "$PERF_DATA" | jq -r '.metrics.goroutine_count // 0')
        UPTIME=$(echo "$PERF_DATA" | jq -r '.metrics.uptime_seconds // 0')
        
        # Convert uptime to human readable
        UPTIME_HUMAN=$(printf "%02d:%02d:%02d" $((UPTIME/3600)) $(((UPTIME%3600)/60)) $((UPTIME%60)))
        
        printf "  Memory: %.1f MB | Goroutines: %s | Uptime: %s\n" "$MEMORY_MB" "$GOROUTINES" "$UPTIME_HUMAN"
        
        # Show recommendations if any
        RECOMMENDATIONS=$(echo "$PERF_DATA" | jq -r '.health.recommendations[]?' 2>/dev/null)
        if [ ! -z "$RECOMMENDATIONS" ]; then
            echo ""
            echo -e "\033[36mRecommendations:\033[0m"
            echo "$RECOMMENDATIONS" | while read rec; do
                echo "  - $rec"
            done
        fi
        
    else
        echo -e "\033[31mError: Cannot connect to server at $SERVER_URL\033[0m"
        echo "Make sure the server is running and accessible."
    fi
    
    echo ""
    echo "Next update in ${INTERVAL}s... (Press Ctrl+C to exit)"
    
    sleep "$INTERVAL"
done

