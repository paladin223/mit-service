#!/usr/bin/env python3
"""
Read Load Testing Script
Generate controlled read load with 200 RPS from ID 1 to infinity
"""

import asyncio
import aiohttp
import time
import sys
import signal
import hashlib
from typing import Optional
import asyncio_throttle


class ReadLoadTester:
    def __init__(self, base_url: str, rps: int = 200, start_id: int = 1, duration: Optional[int] = None):
        self.base_url = base_url.rstrip('/')
        self.rps = rps  # requests per second
        self.start_id = start_id
        self.duration = duration  # None = infinite
        
        # Statistics
        self.total_requests = 0
        self.successful_requests = 0
        self.failed_requests = 0
        self.not_found_requests = 0
        self.current_id = start_id
        
        # Control
        self.running = True
        self.start_time = None
        
        # Rate limiting
        self.throttler = asyncio_throttle.Throttler(rate_limit=rps, period=1.0)
        
        # Response time tracking
        self.response_times = []
        self.last_stats_time = time.time()
    
    def generate_record_id(self, record_number: int) -> str:
        """Generate record ID as MD5(abcdefg + number) - same as populate_db.py"""
        base_string = f"abcdefg{record_number}"
        return hashlib.md5(base_string.encode()).hexdigest()

    def signal_handler(self, signum, frame):
        """Handle Ctrl+C gracefully"""
        print(f"\nReceived signal {signum}, stopping...")
        self.running = False

    async def make_request(self, session: aiohttp.ClientSession, record_number: int) -> bool:
        """Make a single GET request"""
        record_id = self.generate_record_id(record_number)
        url = f"{self.base_url}/get?id={record_id}"
        
        start_time = time.time()
        try:
            async with self.throttler:
                async with session.get(url, timeout=aiohttp.ClientTimeout(total=10)) as response:
                    response_time = time.time() - start_time
                    self.response_times.append(response_time)
                    
                    self.total_requests += 1
                    
                    if response.status == 200:
                        self.successful_requests += 1
                        return True
                    elif response.status == 404:
                        self.not_found_requests += 1
                        return False
                    else:
                        self.failed_requests += 1
                        text = await response.text()
                        print(f"Request failed with status {response.status} for record {record_number} (ID: {record_id}): {text}")
                        return False
        
        except asyncio.TimeoutError:
            response_time = time.time() - start_time
            self.response_times.append(response_time)
            self.total_requests += 1
            self.failed_requests += 1
            print(f"Request timeout for record {record_number} (ID: {record_id})")
            return False
        
        except Exception as e:
            response_time = time.time() - start_time
            self.response_times.append(response_time)
            self.total_requests += 1
            self.failed_requests += 1
            print(f"Request error for record {record_number} (ID: {record_id}): {e}")
            return False

    def print_stats(self):
        """Print current statistics"""
        current_time = time.time()
        elapsed = current_time - self.start_time
        
        # Calculate rates
        actual_rps = self.total_requests / elapsed if elapsed > 0 else 0
        success_rate = (self.successful_requests / self.total_requests * 100) if self.total_requests > 0 else 0
        
        # Calculate response time stats
        if self.response_times:
            avg_response_time = sum(self.response_times) / len(self.response_times) * 1000  # ms
            self.response_times.clear()  # Clear for next period
        else:
            avg_response_time = 0
        
        print(f"\r⚡ Read Load Test - ID: {self.current_id} | "
              f"RPS: {actual_rps:.1f} | "
              f"Total: {self.total_requests} | "
              f"Success: {self.successful_requests} ({success_rate:.1f}%) | "
              f"Not Found: {self.not_found_requests} | "
              f"Failed: {self.failed_requests} | "
              f"Avg RT: {avg_response_time:.1f}ms", end='', flush=True)

    async def worker(self, session: aiohttp.ClientSession):
        """Worker coroutine that makes requests"""
        while self.running:
            # Check duration limit
            if self.duration and (time.time() - self.start_time) >= self.duration:
                self.running = False
                break
            
            await self.make_request(session, self.current_id)
            self.current_id += 1
            
            # Print stats every second
            current_time = time.time()
            if current_time - self.last_stats_time >= 1.0:
                self.print_stats()
                self.last_stats_time = current_time

    async def run(self):
        """Run the read load test"""
        # Setup signal handlers
        signal.signal(signal.SIGINT, self.signal_handler)
        signal.signal(signal.SIGTERM, self.signal_handler)
        
        print(f"🚀 Starting Read Load Test:")
        print(f"   Target: {self.base_url}")
        print(f"   RPS: {self.rps}")
        print(f"   Start ID: {self.start_id}")
        print(f"   Duration: {'Infinite' if not self.duration else f'{self.duration}s'}")
        print(f"   Press Ctrl+C to stop\n")
        
        self.start_time = time.time()
        self.last_stats_time = self.start_time
        
        connector = aiohttp.TCPConnector(limit=100, limit_per_host=100)
        timeout = aiohttp.ClientTimeout(total=30)
        
        try:
            async with aiohttp.ClientSession(connector=connector, timeout=timeout) as session:
                # Start multiple workers to achieve desired RPS
                workers_count = min(50, self.rps // 4)  # Adjust based on RPS
                if workers_count < 1:
                    workers_count = 1
                
                tasks = [self.worker(session) for _ in range(workers_count)]
                await asyncio.gather(*tasks)
        
        except Exception as e:
            print(f"\nUnexpected error: {e}")
        
        finally:
            self.print_final_stats()

    def print_final_stats(self):
        """Print final statistics"""
        elapsed = time.time() - self.start_time
        actual_rps = self.total_requests / elapsed if elapsed > 0 else 0
        success_rate = (self.successful_requests / self.total_requests * 100) if self.total_requests > 0 else 0
        
        print(f"\n\n📊 Final Statistics:")
        print(f"   Duration: {elapsed:.1f}s")
        print(f"   Total Requests: {self.total_requests}")
        print(f"   Successful: {self.successful_requests} ({success_rate:.1f}%)")
        print(f"   Not Found: {self.not_found_requests}")
        print(f"   Failed: {self.failed_requests}")
        print(f"   Actual RPS: {actual_rps:.1f}")
        print(f"   Last ID reached: {self.current_id - 1}")
        print(f"   Range tested: {self.start_id} - {self.current_id - 1}")


async def main():
    """Main function"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Read Load Testing Script')
    parser.add_argument('--url', default='http://localhost:8080', 
                      help='Base URL (default: http://localhost:8080)')
    parser.add_argument('--rps', type=int, default=200, 
                      help='Requests per second (default: 200)')
    parser.add_argument('--start-id', type=int, default=1, 
                      help='Starting ID (default: 1)')
    parser.add_argument('--duration', type=int, 
                      help='Test duration in seconds (default: infinite)')
    
    args = parser.parse_args()
    
    # Validate parameters
    if args.rps <= 0:
        print("Error: RPS must be positive")
        sys.exit(1)
    
    if args.start_id < 1:
        print("Error: Start ID must be >= 1")
        sys.exit(1)
    
    # Test connectivity
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(f"{args.url}/health", timeout=aiohttp.ClientTimeout(total=5)) as response:
                if response.status != 200:
                    print(f"Warning: Health check failed with status {response.status}")
    except Exception as e:
        print(f"Error: Cannot connect to {args.url}: {e}")
        print("Make sure the MIT service is running")
        sys.exit(1)
    
    # Run the test
    tester = ReadLoadTester(
        base_url=args.url,
        rps=args.rps,
        start_id=args.start_id,
        duration=args.duration
    )
    
    await tester.run()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
    except Exception as e:
        print(f"\nUnexpected error: {e}")
        sys.exit(1)
