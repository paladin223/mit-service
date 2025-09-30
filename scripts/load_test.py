#!/usr/bin/env python3
"""
MIT Service Load Testing Script
For each task: INSERT -> UPDATE -> GET
ID = MD5(abcdefg + task_number)
"""

import asyncio
import aiohttp
import argparse
import hashlib
import json
import time
from dataclasses import dataclass, field
from typing import Dict, List
import sys


@dataclass
class TestStats:
    """Статистика тестирования"""
    total_requests: int = 0
    successful_requests: int = 0
    failed_requests: int = 0
    insert_success: int = 0
    insert_failed: int = 0
    update_success: int = 0
    update_failed: int = 0
    get_success: int = 0
    get_failed: int = 0
    response_times: List[float] = field(default_factory=list)
    start_time: float = 0
    end_time: float = 0
    
    @property
    def duration(self) -> float:
        return self.end_time - self.start_time
    
    @property
    def actual_rps(self) -> float:
        return self.total_requests / self.duration if self.duration > 0 else 0
    
    @property
    def success_rate(self) -> float:
        return (self.successful_requests / self.total_requests * 100) if self.total_requests > 0 else 0
    
    @property
    def avg_response_time(self) -> float:
        return sum(self.response_times) / len(self.response_times) if self.response_times else 0


class LoadTester:
    """Класс для проведения нагрузочного тестирования"""
    
    def __init__(self, base_url: str, rps: int, tasks_count: int, timeout: int = 10):
        self.base_url = base_url.rstrip('/')
        self.rps = rps
        self.tasks_count = tasks_count
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self.stats = TestStats()
        self.request_semaphore = asyncio.Semaphore(rps * 2)  # Ограничиваем concurrent requests
        
    def generate_task_id(self, task_number: int) -> str:
        """Генерирует ID задачи как хеш от 'abcdefg' + номер задачи"""
        base_string = f"abcdefg{task_number}"
        return hashlib.md5(base_string.encode()).hexdigest()
    
    def generate_test_data(self, request_number: int) -> Dict:
        """Генерирует тестовые данные для запросов"""
        return {
            "name": f"Test Record {request_number}",
            "description": f"Generated for request #{request_number}",
            "timestamp": time.time(),
            "request_number": request_number,
            "metadata": {
                "test": True,
                "batch": "load_test",
                "created_at": time.strftime("%Y-%m-%d %H:%M:%S")
            }
        }
    
    async def make_request(self, session: aiohttp.ClientSession, method: str, 
                          endpoint: str, data: Dict = None, params: Dict = None) -> tuple:
        """Выполняет HTTP запрос и возвращает результат"""
        start_time = time.time()
        
        async with self.request_semaphore:
            try:
                url = f"{self.base_url}{endpoint}"
                
                if method == 'GET':
                    async with session.get(url, params=params) as response:
                        result = await response.json()
                        status = response.status
                elif method == 'POST':
                    async with session.post(url, json=data) as response:
                        result = await response.json()
                        status = response.status
                else:
                    raise ValueError(f"Unsupported method: {method}")
                
                response_time = time.time() - start_time
                return True, status, result, response_time
                
            except Exception as e:
                response_time = time.time() - start_time
                return False, 0, str(e), response_time
    
    async def execute_task_sequence(self, session: aiohttp.ClientSession, task_number: int):
        """Выполняет последовательность запросов для одной задачи"""
        task_id = self.generate_task_id(task_number)
        request_number = task_number * 2  # Стартовый номер запроса для этой задачи
        
        # 1. INSERT запрос (запись 1)
        request_number += 1
        test_data = self.generate_test_data(request_number)
        insert_data = {"id": task_id, "value": test_data}
        
        success, status, result, response_time = await self.make_request(
            session, 'POST', '/insert', insert_data
        )
        
        self.stats.total_requests += 1
        self.stats.response_times.append(response_time)
        
        if success and 200 <= status < 300:
            self.stats.successful_requests += 1
            self.stats.insert_success += 1
            print(f"✅ INSERT task {task_number}: {status} ({response_time:.3f}s)")
        else:
            self.stats.failed_requests += 1
            self.stats.insert_failed += 1
            print(f"❌ INSERT task {task_number}: {status} - {result} ({response_time:.3f}s)")
        
        # 2. UPDATE запрос (запись 2)
        request_number += 1
        updated_data = self.generate_test_data(request_number)
        updated_data["updated"] = True
        update_data = {"id": task_id, "value": updated_data}
        
        success, status, result, response_time = await self.make_request(
            session, 'POST', '/update', update_data
        )
        
        self.stats.total_requests += 1
        self.stats.response_times.append(response_time)
        
        if success and 200 <= status < 300:
            self.stats.successful_requests += 1
            self.stats.update_success += 1
            print(f"✅ UPDATE task {task_number}: {status} ({response_time:.3f}s)")
        else:
            self.stats.failed_requests += 1
            self.stats.update_failed += 1
            print(f"❌ UPDATE task {task_number}: {status} - {result} ({response_time:.3f}s)")
        
        # 3. GET запрос (чтение)
        success, status, result, response_time = await self.make_request(
            session, 'GET', '/get', params={'id': task_id}
        )
        
        self.stats.total_requests += 1
        self.stats.response_times.append(response_time)
        
        if success and 200 <= status < 300:
            self.stats.successful_requests += 1
            self.stats.get_success += 1
            print(f"✅ GET task {task_number}: {status} ({response_time:.3f}s)")
        else:
            self.stats.failed_requests += 1
            self.stats.get_failed += 1
            print(f"❌ GET task {task_number}: {status} - {result} ({response_time:.3f}s)")
    
    async def run_load_test(self):
        """Запускает нагрузочное тестирование"""
        print(f"🚀 Запуск нагрузочного тестирования:")
        print(f"   URL: {self.base_url}")
        print(f"   RPS: {self.rps}")
        print(f"   Задач: {self.tasks_count}")
        print(f"   Всего запросов: {self.tasks_count * 3}")
        print()
        
        self.stats.start_time = time.time()
        
        # Создаем HTTP сессию
        connector = aiohttp.TCPConnector(limit=self.rps * 2)
        
        async with aiohttp.ClientSession(
            timeout=self.timeout,
            connector=connector
        ) as session:
            
            # Проверяем доступность сервера
            try:
                success, status, result, _ = await self.make_request(session, 'GET', '/health')
                if not success or status != 200:
                    print(f"❌ Сервер недоступен: {self.base_url}/health - {result}")
                    return
                print(f"✅ Сервер доступен: {result}")
                print()
            except Exception as e:
                print(f"❌ Не удалось подключиться к серверу: {e}")
                return
            
            # Создаем задачи с контролем RPS
            tasks = []
            interval = 1.0 / (self.rps / 3)  # 3 запроса на задачу
            
            for task_number in range(self.tasks_count):
                # Планируем выполнение задачи
                delay = task_number * interval
                task = asyncio.create_task(
                    self.delayed_task_execution(session, task_number, delay)
                )
                tasks.append(task)
            
            # Ждем выполнения всех задач
            await asyncio.gather(*tasks, return_exceptions=True)
        
        self.stats.end_time = time.time()
        self.print_stats()
    
    async def delayed_task_execution(self, session: aiohttp.ClientSession, 
                                   task_number: int, delay: float):
        """Выполняет задачу с задержкой для контроля RPS"""
        if delay > 0:
            await asyncio.sleep(delay)
        await self.execute_task_sequence(session, task_number)
    
    def print_stats(self):
        """Выводит статистику тестирования"""
        print("\n" + "="*80)
        print("📊 РЕЗУЛЬТАТЫ НАГРУЗОЧНОГО ТЕСТИРОВАНИЯ")
        print("="*80)
        print(f"Время выполнения: {self.stats.duration:.2f} сек")
        print(f"Заданный RPS: {self.rps}")
        print(f"Фактический RPS: {self.stats.actual_rps:.2f}")
        print(f"Всего запросов: {self.stats.total_requests}")
        print(f"Успешных запросов: {self.stats.successful_requests}")
        print(f"Неудачных запросов: {self.stats.failed_requests}")
        print(f"Успешность: {self.stats.success_rate:.2f}%")
        print(f"Среднее время ответа: {self.stats.avg_response_time:.3f} сек")
        
        print(f"\nДетализация по операциям:")
        print(f"  INSERT: ✅ {self.stats.insert_success} | ❌ {self.stats.insert_failed}")
        print(f"  UPDATE: ✅ {self.stats.update_success} | ❌ {self.stats.update_failed}")
        print(f"  GET:    ✅ {self.stats.get_success} | ❌ {self.stats.get_failed}")
        
        if self.stats.response_times:
            sorted_times = sorted(self.stats.response_times)
            p50 = sorted_times[int(len(sorted_times) * 0.5)]
            p95 = sorted_times[int(len(sorted_times) * 0.95)]
            p99 = sorted_times[int(len(sorted_times) * 0.99)]
            
            print(f"\nПерцентили времени ответа:")
            print(f"  P50: {p50:.3f} сек")
            print(f"  P95: {p95:.3f} сек")
            print(f"  P99: {p99:.3f} сек")
            print(f"  MIN: {min(sorted_times):.3f} сек")
            print(f"  MAX: {max(sorted_times):.3f} сек")


async def main():
    parser = argparse.ArgumentParser(description='MIT Service Load Testing')
    parser.add_argument('--url', default='http://localhost:8080', 
                       help='Base URL сервера (по умолчанию: http://localhost:8080)')
    parser.add_argument('--rps', type=int, default=10,
                       help='Количество запросов в секунду (по умолчанию: 10)')
    parser.add_argument('--tasks', type=int, default=100,
                       help='Количество задач для тестирования (по умолчанию: 100)')
    parser.add_argument('--timeout', type=int, default=10,
                       help='Таймаут запросов в секундах (по умолчанию: 10)')
    
    args = parser.parse_args()
    
    if args.rps <= 0 or args.tasks <= 0:
        print("❌ RPS и количество задач должны быть положительными числами")
        sys.exit(1)
    
    tester = LoadTester(
        base_url=args.url,
        rps=args.rps,
        tasks_count=args.tasks,
        timeout=args.timeout
    )
    
    try:
        await tester.run_load_test()
    except KeyboardInterrupt:
        print("\n\n⚠️  Тестирование прервано пользователем")
    except Exception as e:
        print(f"\n\n❌ Ошибка при выполнении тестирования: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
