#!/usr/bin/env python3
"""
Performance Benchmark for MCP Server
Usage: python3 benchmark.py
"""

import json
import subprocess
import time
import statistics
import os

class MCPBenchmark:
    def __init__(self):
        self.server_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), "mcp-whisker")
        self.results = {}
    
    def benchmark_tool(self, tool_name, arguments=None, iterations=3):
        """Benchmark a specific MCP tool"""
        if arguments is None:
            arguments = {}
        
        print(f"\n📊 Benchmarking: {tool_name} ({iterations} iterations)")
        
        times = []
        success_count = 0
        
        for i in range(iterations):
            print(f"  Run {i+1}/{iterations}...", end="", flush=True)
            
            request = {
                "jsonrpc": "2.0",
                "id": i+1,
                "method": "tools/call",
                "params": {
                    "name": tool_name,
                    "arguments": arguments
                }
            }
            
            start_time = time.time()
            
            try:
                process = subprocess.Popen(
                    [self.server_path, "server"],
                    stdin=subprocess.PIPE,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    text=True
                )
                
                stdout, stderr = process.communicate(input=json.dumps(request) + '\n', timeout=60)
                
                end_time = time.time()
                duration = end_time - start_time
                
                if stdout.strip():
                    response = json.loads(stdout.strip())
                    if 'result' in response:
                        times.append(duration)
                        success_count += 1
                        print(f" ✅ {duration:.2f}s")
                    else:
                        print(f" ❌ Error")
                else:
                    print(f" ❌ No response")
                    
            except Exception as e:
                end_time = time.time()
                print(f" ❌ Failed: {e}")
        
        if times:
            self.results[tool_name] = {
                "success_rate": (success_count / iterations) * 100,
                "avg_time": statistics.mean(times),
                "min_time": min(times),
                "max_time": max(times),
                "std_dev": statistics.stdev(times) if len(times) > 1 else 0
            }
            
            print(f"  📈 Success Rate: {success_count}/{iterations} ({self.results[tool_name]['success_rate']:.1f}%)")
            print(f"  ⏱️  Avg Time: {self.results[tool_name]['avg_time']:.2f}s")
            print(f"  🏃 Min Time: {self.results[tool_name]['min_time']:.2f}s")
            print(f"  🐌 Max Time: {self.results[tool_name]['max_time']:.2f}s")
        else:
            print(f"  ❌ All runs failed")
    
    def run_benchmark_suite(self):
        """Run complete benchmark suite"""
        print("🚀 MCP Server Performance Benchmark")
        print("=" * 50)
        
        if not os.path.exists(self.server_path):
            print("❌ MCP server binary not found")
            return
        
        # Benchmark different tools
        benchmarks = [
            ("check_whisker_service", {}),
            ("setup_port_forward", {"namespace": "calico-system"}),
            ("get_flow_logs", {"setup_port_forward": False}),  # Assume port forward already setup
        ]
        
        for tool_name, args in benchmarks:
            try:
                self.benchmark_tool(tool_name, args, iterations=3)
            except KeyboardInterrupt:
                print(f"\n⚠️  Benchmark interrupted")
                break
        
        # Summary
        print("\n" + "=" * 50)
        print("📊 Benchmark Summary:")
        
        if self.results:
            print(f"{'Tool':<25} {'Success%':<10} {'Avg Time':<10} {'Min Time':<10} {'Max Time':<10}")
            print("-" * 65)
            
            for tool, stats in self.results.items():
                print(f"{tool:<25} {stats['success_rate']:<10.1f} {stats['avg_time']:<10.2f} {stats['min_time']:<10.2f} {stats['max_time']:<10.2f}")
        
        print("\n💡 Lower times are better. All times in seconds.")

def main():
    benchmark = MCPBenchmark()
    benchmark.run_benchmark_suite()

if __name__ == "__main__":
    main()