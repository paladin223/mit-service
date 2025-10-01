#!/usr/bin/env python3
"""
Database Population Script
Populate database with 100k records before load testing
"""

import asyncio
import aiohttp
import hashlib
import time
import sys


class DatabasePopulator:
    def __init__(self, base_url: str, total_records: int = 100000, batch_size: int = 100, start_id: int = 1):
        self.base_url = base_url.rstrip('/')
        self.total_records = total_records
        self.batch_size = batch_size
        self.start_id = start_id
        self.inserted = 0
        self.errors = 0
    
    def generate_record_id(self, record_number: int) -> str:
        """Generate record ID as MD5(abcdefg + number) starting from start_id"""
        record_id = self.start_id + record_number  # Start from start_id
        base_string = f"abcdefg{record_id}"
        return hashlib.md5(base_string.encode()).hexdigest()
    
    def generate_record_data(self, record_number: int) -> dict:
        """Generate test data for record"""
        return {
            "name": f"Test User {record_number}",
            "email": f"user{record_number}@example.com",
            "age": 20 + (record_number % 60),
            "city": f"City {record_number % 100}",
            "created_at": time.strftime("%Y-%m-%d %H:%M:%S"),
            "record_number": record_number,
            "department": f"Dept {record_number % 10}",
            "salary": 30000 + (record_number % 70000),
            "metadata": {
                "populated": True,
                "batch": record_number // self.batch_size,
                "test_data": True
            }
        }
    
    async def insert_record(self, session: aiohttp.ClientSession, record_number: int):
        """Insert single record"""
        record_id = self.generate_record_id(record_number)
        data = self.generate_record_data(record_number)
        
        payload = {
            "id": record_id,
            "value": data
        }
        
        try:
            async with session.post(f"{self.base_url}/insert", json=payload) as response:
                if 200 <= response.status < 300:
                    self.inserted += 1
                    return True
                else:
                    self.errors += 1
                    return False
        except Exception:
            self.errors += 1
            return False
    
    async def populate_batch(self, session: aiohttp.ClientSession, start_number: int, batch_size: int):
        """Populate batch of records"""
        tasks = []
        for i in range(start_number, min(start_number + batch_size, self.total_records)):
            task = self.insert_record(session, i)
            tasks.append(task)
        
        results = await asyncio.gather(*tasks, return_exceptions=True)
        return sum(1 for r in results if r is True)
    
    async def populate_database(self):
        """Main population function"""
        end_id = self.start_id + self.total_records - 1
        print(f"🚀 Starting database population:")
        print(f"   Target: {self.total_records:,} records")
        print(f"   ID range: {self.start_id} to {end_id}")
        print(f"   Batch size: {self.batch_size}")
        print(f"   URL: {self.base_url}")
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
        
        start_time = time.time()
        connector = aiohttp.TCPConnector(limit=50)
        
        async with aiohttp.ClientSession(
            connector=connector,
            timeout=aiohttp.ClientTimeout(total=30)
        ) as session:
            
            batch_number = 0
            for start in range(0, self.total_records, self.batch_size):
                batch_number += 1
                batch_start = time.time()
                
                # Process batch
                batch_success = await self.populate_batch(session, start, self.batch_size)
                
                batch_time = time.time() - batch_start
                progress = (start + self.batch_size) / self.total_records * 100
                
                # Progress update
                if batch_number % 10 == 0 or start + self.batch_size >= self.total_records:
                    elapsed = time.time() - start_time
                    rate = self.inserted / elapsed if elapsed > 0 else 0
                    eta = (self.total_records - self.inserted) / rate if rate > 0 else 0
                    
                    print(f"Batch {batch_number:4d}: {progress:6.1f}% | "
                          f"Inserted: {self.inserted:6,} | "
                          f"Rate: {rate:6.1f}/s | "
                          f"ETA: {eta:4.0f}s | "
                          f"Batch time: {batch_time:.2f}s")
        
        total_time = time.time() - start_time
        self.print_summary(total_time)
    
    def print_summary(self, total_time: float):
        """Print population summary"""
        print("\n" + "="*80)
        print("📊 DATABASE POPULATION SUMMARY")
        print("="*80)
        print(f"Total time: {total_time:.2f} seconds")
        print(f"Records inserted: {self.inserted:,}")
        print(f"Errors: {self.errors:,}")
        print(f"Success rate: {(self.inserted / self.total_records * 100):.2f}%")
        print(f"Average rate: {(self.inserted / total_time):.1f} records/second")
        
        if self.inserted > 0:
            start_hash = self.generate_record_id(0)[:8]
            end_hash = self.generate_record_id(self.total_records-1)[:8] 
            end_id = self.start_id + self.total_records - 1
            print(f"\n✅ Database populated successfully!")
            print(f"📝 Records with IDs from: {start_hash}... (abcdefg{self.start_id})")
            print(f"📝                    to: {end_hash}... (abcdefg{end_id})")
            print(f"\n🔥 Ready for load testing!")
        else:
            print(f"\n❌ Population failed!")


async def main():
    import argparse
    
    parser = argparse.ArgumentParser(description='Populate database with test records')
    parser.add_argument('--url', default='http://localhost:8080', 
                       help='Service URL (default: http://localhost:8080)')
    parser.add_argument('--records', type=int, default=100000,
                       help='Number of records to insert (default: 100,000)')
    parser.add_argument('--batch', type=int, default=100,
                       help='Batch size (default: 100)')
    parser.add_argument('--start', type=int, default=1,
                       help='Start ID number (default: 1)')
    
    args = parser.parse_args()
    
    if args.records <= 0 or args.batch <= 0 or args.start < 0:
        print("❌ Records and batch size must be positive, start ID must be >= 0")
        sys.exit(1)
    
    populator = DatabasePopulator(
        base_url=args.url,
        total_records=args.records,
        batch_size=args.batch,
        start_id=args.start
    )
    
    try:
        await populator.populate_database()
    except KeyboardInterrupt:
        print(f"\n\n⚠️  Population interrupted by user")
        print(f"📊 Inserted {populator.inserted:,} out of {args.records:,} records")
    except Exception as e:
        print(f"\n\n❌ Error during population: {e}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
