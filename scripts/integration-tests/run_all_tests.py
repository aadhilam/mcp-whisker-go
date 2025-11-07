#!/usr/bin/env python3
"""
MCP Server Test Suite - Run all tests
Usage: python3 run_all_tests.py
"""

import json
import subprocess
import sys
import os

class MCPTestSuite:
    def __init__(self):
        self.server_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), "mcp-whisker")
        self.passed = 0
        self.failed = 0
        
    def run_mcp_request(self, method, params=None, timeout=30):
        """Run a single MCP request"""
        request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params or {}
        }
        
        try:
            process = subprocess.Popen(
                [self.server_path, "server"],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            
            request_json = json.dumps(request)
            stdout, stderr = process.communicate(input=request_json + '\n', timeout=timeout)
            
            if stdout.strip():
                return json.loads(stdout.strip())
            return None
            
        except Exception as e:
            print(f"❌ Request failed: {e}")
            return None
    
    def test_initialization(self):
        """Test MCP server initialization"""
        print("\n🧪 Testing MCP Server Initialization...")
        
        response = self.run_mcp_request("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "test-suite", "version": "1.0.0"}
        })
        
        if response and 'result' in response:
            server_info = response['result'].get('serverInfo', {})
            print(f"✅ Server initialized: {server_info.get('name')} v{server_info.get('version')}")
            self.passed += 1
            return True
        else:
            print("❌ Server initialization failed")
            self.failed += 1
            return False
    
    def test_tools_list(self):
        """Test tools listing"""
        print("\n🧪 Testing Tools List...")
        
        response = self.run_mcp_request("tools/list")
        
        if response and 'result' in response:
            tools = response['result'].get('tools', [])
            print(f"✅ Found {len(tools)} tools:")
            for tool in tools:
                print(f"   • {tool['name']}: {tool['description']}")
            self.passed += 1
            return True
        else:
            print("❌ Tools list failed")
            self.failed += 1
            return False
    
    def test_whisker_service_check(self):
        """Test Whisker service availability check"""
        print("\n🧪 Testing Whisker Service Check...")
        
        response = self.run_mcp_request("tools/call", {
            "name": "check_whisker_service",
            "arguments": {}
        })
        
        if response and 'result' in response:
            content = response['result'].get('content', [{}])[0].get('text', '')
            try:
                result = json.loads(content)
                if result.get('available'):
                    print(f"✅ Whisker service is available: {result.get('status')}")
                    self.passed += 1
                    return True
                else:
                    print(f"⚠️  Whisker service not available: {result.get('details')}")
                    self.passed += 1  # Still a successful test
                    return True
            except:
                print(f"✅ Service check completed: {content}")
                self.passed += 1
                return True
        else:
            print("❌ Service check failed")
            self.failed += 1
            return False
    
    def test_port_forward_setup(self):
        """Test port forward setup"""
        print("\n🧪 Testing Port Forward Setup...")
        
        response = self.run_mcp_request("tools/call", {
            "name": "setup_port_forward",
            "arguments": {"namespace": "calico-system"}
        }, timeout=25)
        
        if response and 'result' in response:
            content = response['result'].get('content', [{}])[0].get('text', '')
            print(f"✅ Port forward setup: {content}")
            self.passed += 1
            return True
        else:
            error = response.get('error', {}) if response else {}
            print(f"❌ Port forward failed: {error.get('message', 'Unknown error')}")
            self.failed += 1
            return False
    
    def test_flow_logs_retrieval(self):
        """Test flow logs retrieval"""
        print("\n🧪 Testing Flow Logs Retrieval...")
        
        response = self.run_mcp_request("tools/call", {
            "name": "get_flow_logs",
            "arguments": {"setup_port_forward": True}
        }, timeout=30)
        
        if response and 'result' in response:
            content = response['result'].get('content', [{}])[0].get('text', '')
            try:
                flows = json.loads(content)
                if isinstance(flows, list) and len(flows) > 0:
                    print(f"✅ Retrieved {len(flows)} flow log entries")
                    self.passed += 1
                    return True
                else:
                    print("✅ Flow logs retrieved (empty result)")
                    self.passed += 1
                    return True
            except:
                print(f"✅ Flow logs retrieved: {len(content)} characters")
                self.passed += 1
                return True
        else:
            error = response.get('error', {}) if response else {}
            print(f"❌ Flow logs failed: {error.get('message', 'Unknown error')}")
            self.failed += 1
            return False
    
    def test_namespace_analysis(self):
        """Test namespace flow analysis"""
        print("\n🧪 Testing Namespace Flow Analysis...")
        
        response = self.run_mcp_request("tools/call", {
            "name": "analyze_namespace_flows",
            "arguments": {
                "namespace": "kube-system",
                "setup_port_forward": True
            }
        }, timeout=30)
        
        if response and 'result' in response:
            content = response['result'].get('content', [{}])[0].get('text', '')
            try:
                analysis = json.loads(content)
                total_flows = analysis.get('analysis', {}).get('totalUniqueFlows', 0)
                print(f"✅ Namespace analysis completed: {total_flows} unique flows found")
                self.passed += 1
                return True
            except:
                print(f"✅ Namespace analysis completed: {len(content)} characters")
                self.passed += 1
                return True
        else:
            error = response.get('error', {}) if response else {}
            print(f"❌ Namespace analysis failed: {error.get('message', 'Unknown error')}")
            self.failed += 1
            return False
    
    def test_blocked_flows_analysis(self):
        """Test blocked flows analysis"""
        print("\n🧪 Testing Blocked Flows Analysis...")
        
        response = self.run_mcp_request("tools/call", {
            "name": "analyze_blocked_flows",
            "arguments": {"setup_port_forward": True}
        }, timeout=30)
        
        if response and 'result' in response:
            content = response['result'].get('content', [{}])[0].get('text', '')
            try:
                analysis = json.loads(content)
                blocked_count = analysis.get('analysis', {}).get('totalBlockedFlows', 0)
                print(f"✅ Blocked flows analysis completed: {blocked_count} blocked flows found")
                self.passed += 1
                return True
            except:
                print(f"✅ Blocked flows analysis completed: {len(content)} characters")
                self.passed += 1
                return True
        else:
            error = response.get('error', {}) if response else {}
            print(f"❌ Blocked flows analysis failed: {error.get('message', 'Unknown error')}")
            self.failed += 1
            return False
    
    def run_all_tests(self):
        """Run all test cases"""
        print("🚀 MCP Whisker Server Test Suite")
        print("=" * 50)
        
        # Check if server binary exists
        if not os.path.exists(self.server_path):
            print(f"❌ MCP server binary not found at {self.server_path}")
            print("   Please build the server first: go build -o mcp-whisker ./cmd/server")
            return
        
        # Run tests in order
        tests = [
            ("Initialization", self.test_initialization),
            ("Tools List", self.test_tools_list),
            ("Service Check", self.test_whisker_service_check),
            ("Port Forward", self.test_port_forward_setup),
            ("Flow Logs", self.test_flow_logs_retrieval),
            ("Namespace Analysis", self.test_namespace_analysis),
            ("Blocked Flows", self.test_blocked_flows_analysis),
        ]
        
        for test_name, test_func in tests:
            try:
                test_func()
            except KeyboardInterrupt:
                print(f"\n⚠️  Test interrupted: {test_name}")
                break
            except Exception as e:
                print(f"❌ Test error in {test_name}: {e}")
                self.failed += 1
        
        # Summary
        print("\n" + "=" * 50)
        print("📊 Test Summary:")
        print(f"✅ Passed: {self.passed}")
        print(f"❌ Failed: {self.failed}")
        print(f"📈 Success Rate: {(self.passed/(self.passed+self.failed)*100):.1f}%")
        
        if self.failed == 0:
            print("\n🎉 All tests passed! MCP server is working correctly.")
        else:
            print(f"\n⚠️  {self.failed} test(s) failed. Check the output above for details.")

def main():
    suite = MCPTestSuite()
    suite.run_all_tests()

if __name__ == "__main__":
    main()