#!/usr/bin/env python3
"""
Load Testing Script
Generate controlled load with N requests per second
"""

import asyncio
import aiohttp
import hashlib
import time
import sys
import signal
from typing import Optional


class LoadTester:
    def __init__(self, base_url: str, rps: int = 100, duration: int = 60, start_id: int = 1):
        self.base_url = base_url.rstrip('/')
        self.rps = rps  # requests per second
        self.duration = duration  # test duration in seconds
        self.start_id = start_id
        
        # Statistics
        self.total_requests = 0
        self.successful_requests = 0
        self.failed_requests = 0
        self.current_record_id = start_id
        
        # Control
        self.running = True
        self.start_time = None
    
    def generate_record_id(self, record_number: int) -> str:
        """Generate record ID as MD5(abcdefg + number)"""
        base_string = f"abcdefg{record_number}"
        return hashlib.md5(base_string.encode()).hexdigest()
    
    def generate_record_data(self, record_number: int) -> dict:
        """Generate test data for record"""
        return {
            "name": f"Load Test User {record_number}",
            "email": f"loadtest{record_number}@example.com",
            "age": 20 + (record_number % 60),
            "city": f"City {record_number % 100}",
            "created_at": time.strftime("%Y-%m-%d %H:%M:%S"),
            "record_number": record_number,
            "department": f"Dept {record_number % 10}",
            "salary": 30000 + (record_number % 70000),
            "metadata": {
                "load_test": True,
                "test_data": True,
                "timestamp": time.time()
            }
        }
    
    async def send_request(self, session: aiohttp.ClientSession, record_number: int) -> bool:
        """Send single insert request"""
        record_id = self.generate_record_id(record_number)
        data = self.generate_record_data(record_number)
        
        payload = {
            "id": record_id,
            "value": data
        }
        
        try:
            async with session.post(f"{self.base_url}/insert", json=payload) as response:
                if 200 <= response.status < 300:
                    self.successful_requests += 1
                    return True
                else:
                    self.failed_requests += 1
                    return False
        except Exception:
            self.failed_requests += 1
            return False
    
    async def send_batch(self, session: aiohttp.ClientSession, batch_size: int) -> tuple[int, int]:
        """Send batch of requests concurrently"""
        tasks = []
        start_record = self.current_record_id
        
        for i in range(batch_size):
            task = self.send_request(session, self.current_record_id + i)
            tasks.append(task)
        
        self.current_record_id += batch_size
        self.total_requests += batch_size
        
        results = await asyncio.gather(*tasks, return_exceptions=True)
        successful = sum(1 for r in results if r is True)
        failed = batch_size - successful
        
        return successful, failed
    
    def print_status(self, elapsed_time: float, current_rps: float):
        """Print current status"""
        success_rate = (self.successful_requests / self.total_requests * 100) if self.total_requests > 0 else 0
        avg_rps = self.total_requests / elapsed_time if elapsed_time > 0 else 0
        
        remaining_time = max(0, self.duration - elapsed_time)
        
        print(f"\r⚡ Load Test | "
              f"Time: {elapsed_time:5.1f}s/{self.duration}s | "
              f"RPS: {current_rps:5.1f} (target: {self.rps}) | "
              f"Avg RPS: {avg_rps:5.1f} | "
              f"Total: {self.total_requests:6,} | "
              f"✅ {self.successful_requests:5,} | "
              f"❌ {self.failed_requests:4,} | "
              f"Success: {success_rate:5.1f}% | "
              f"ETA: {remaining_time:4.0f}s", end="", flush=True)
    
    async def run_load_test(self):
        """Run the load test"""
        print(f"🚀 Starting Load Test:")
        print(f"   Target RPS: {self.rps}")
        print(f"   Duration: {self.duration}s")
        print(f"   URL: {self.base_url}")
        print(f"   Expected total requests: ~{self.rps * self.duration:,}")
        print()
        
        # Check if service is available
        try:
            async with aiohttp.ClientSession() as session:
                async with session.get(f"{self.base_url}/health") as response:
                    if response.status != 200:
                        print(f"❌ Service not available: {response.status}")
                        return
                print("✅ Service is available")
        except Exception as e:
            print(f"❌ Cannot connect to service: {e}")
            return
        
        print("\nStarting load test... Press Ctrl+C to stop early\n")
        
        self.start_time = time.time()
        connector = aiohttp.TCPConnector(limit=100)
        
        async with aiohttp.ClientSession(
            connector=connector,
            timeout=aiohttp.ClientTimeout(total=10)
        ) as session:
            
            batch_number = 0
            
            while self.running:
                batch_start_time = time.time()
                elapsed_time = batch_start_time - self.start_time
                
                # Stop if duration exceeded
                if elapsed_time >= self.duration:
                    break
                
                batch_number += 1
                
                # Send batch
                try:
                    successful, failed = await self.send_batch(session, self.rps)
                except Exception as e:
                    print(f"\n❌ Error in batch {batch_number}: {e}")
                    failed += self.rps
                
                batch_end_time = time.time()
                batch_duration = batch_end_time - batch_start_time
                actual_rps = self.rps / batch_duration if batch_duration > 0 else 0
                
                # Print status every second
                self.print_status(elapsed_time, actual_rps)
                
                # Sleep to maintain 1 batch per second
                sleep_time = max(0, 1.0 - batch_duration)
                if sleep_time > 0:
                    await asyncio.sleep(sleep_time)
        
        print("\n")  # New line after status updates
        self.print_summary()
    
    def print_summary(self):
        """Print load test summary"""
        if self.start_time is None:
            return
            
        total_time = time.time() - self.start_time
        avg_rps = self.total_requests / total_time if total_time > 0 else 0
        success_rate = (self.successful_requests / self.total_requests * 100) if self.total_requests > 0 else 0
        
        print("\n" + "="*80)
        print("📊 LOAD TEST SUMMARY")
        print("="*80)
        print(f"Duration: {total_time:.2f} seconds")
        print(f"Target RPS: {self.rps}")
        print(f"Actual Average RPS: {avg_rps:.1f}")
        print(f"RPS Accuracy: {(avg_rps / self.rps * 100):.1f}%")
        print()
        print(f"Total Requests: {self.total_requests:,}")
        print(f"Successful: {self.successful_requests:,}")
        print(f"Failed: {self.failed_requests:,}")
        print(f"Success Rate: {success_rate:.2f}%")
        
        if self.successful_requests > 0:
            print(f"\n✅ Load test completed!")
            print(f"📈 Created sustained load of ~{avg_rps:.0f} RPS")
        else:
            print(f"\n❌ Load test failed!")
    
    def stop(self):
        """Stop the load test"""
        self.running = False


async def main():
    import argparse
    
    parser = argparse.ArgumentParser(description='Generate controlled load for testing')
    parser.add_argument('--url', default='http://localhost:8080', 
                       help='Service URL (default: http://localhost:8080)')
    parser.add_argument('--rps', type=int, default=100,
                       help='Requests per second (default: 100)')
    parser.add_argument('--duration', type=int, default=60,
                       help='Test duration in seconds (default: 60)')
    parser.add_argument('--start', type=int, default=1,
                       help='Start ID number (default: 1)')
    
    args = parser.parse_args()
    
    if args.rps <= 0 or args.duration <= 0 or args.start < 0:
        print("❌ RPS and duration must be positive, start ID must be >= 0")
        sys.exit(1)
    
    load_tester = LoadTester(
        base_url=args.url,
        rps=args.rps,
        duration=args.duration,
        start_id=args.start
    )
    
    # Handle Ctrl+C gracefully
    def signal_handler(sig, frame):
        print(f"\n\n⚠️  Load test interrupted by user")
        load_tester.stop()
    
    signal.signal(signal.SIGINT, signal_handler)
    
    try:
        await load_tester.run_load_test()
    except Exception as e:
        print(f"\n\n❌ Error during load test: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())