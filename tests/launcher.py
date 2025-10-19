#!/usr/bin/env python3
"""
MCP Test Suite Launcher - Choose what to test
Usage: python3 launcher.py
"""

import os
import subprocess
import sys

def show_menu():
    """Display test menu options"""
    print("🚀 MCP Whisker Go - Test Suite Launcher")
    print("=" * 50)
    print()
    print("Available Test Options:")
    print("1. 🔍 Quick Connectivity Test (10 seconds)")
    print("2. 🧪 Full Test Suite (comprehensive)")
    print("3. 📊 Performance Benchmark")
    print("4. 🔧 Interactive Tool Testing")
    print("5. 🐛 Debug Mode Testing")
    print("6. 📖 Show Available Tools")
    print("7. ❌ Exit")
    print()

def run_quick_test():
    """Run quick connectivity test"""
    print("\n🔍 Running Quick Connectivity Test...")
    subprocess.run([sys.executable, "quick_test.py"])

def run_full_suite():
    """Run complete test suite"""
    print("\n🧪 Running Full Test Suite...")
    subprocess.run([sys.executable, "run_all_tests.py"])

def run_benchmark():
    """Run performance benchmark"""
    print("\n📊 Running Performance Benchmark...")
    subprocess.run([sys.executable, "benchmark.py"])

def run_interactive():
    """Interactive tool testing"""
    print("\n🔧 Interactive Tool Testing")
    print("Available tools:")
    print("- check_whisker_service")
    print("- setup_port_forward")
    print("- get_flow_logs")
    print("- analyze_namespace_flows")
    print("- analyze_blocked_flows")
    print()
    
    tool = input("Enter tool name (or 'help' for examples): ").strip()
    
    if tool == 'help':
        show_tool_examples()
        return
    elif tool == '':
        print("❌ No tool specified")
        return
    
    args = input("Enter JSON arguments (or press Enter for defaults): ").strip()
    
    if args:
        subprocess.run([sys.executable, "test_tool.py", tool, args])
    else:
        subprocess.run([sys.executable, "test_tool.py", tool])

def run_debug():
    """Run debug mode test"""
    print("\n🐛 Running Debug Mode Test...")
    subprocess.run([sys.executable, "debug_mcp.py"])

def show_tool_examples():
    """Show tool usage examples"""
    print("\n📖 Tool Usage Examples:")
    print("-" * 30)
    print("check_whisker_service:")
    print("  Arguments: (none)")
    print()
    print("setup_port_forward:")
    print("  Arguments: {\"namespace\": \"calico-system\"}")
    print()
    print("analyze_namespace_flows:")
    print("  Arguments: {\"namespace\": \"kube-system\"}")
    print("  Arguments: {\"namespace\": \"production\", \"setup_port_forward\": true}")
    print()
    print("analyze_blocked_flows:")
    print("  Arguments: (none for all namespaces)")
    print("  Arguments: {\"namespace\": \"production\"}")
    print()
    print("get_flow_logs:")
    print("  Arguments: {\"setup_port_forward\": true}")
    print("  Arguments: {\"setup_port_forward\": false}")

def main():
    """Main launcher loop"""
    while True:
        try:
            show_menu()
            choice = input("Select option (1-7): ").strip()
            
            if choice == '1':
                run_quick_test()
            elif choice == '2':
                run_full_suite()
            elif choice == '3':
                run_benchmark()
            elif choice == '4':
                run_interactive()
            elif choice == '5':
                run_debug()
            elif choice == '6':
                show_tool_examples()
            elif choice == '7':
                print("\n👋 Goodbye!")
                break
            else:
                print("❌ Invalid choice. Please select 1-7.")
            
            print("\n" + "="*50)
            input("Press Enter to continue...")
            
        except KeyboardInterrupt:
            print("\n\n👋 Goodbye!")
            break
        except Exception as e:
            print(f"\n❌ Error: {e}")

if __name__ == "__main__":
    # Check if we're in the right directory
    if not os.path.exists("quick_test.py"):
        print("❌ Please run this script from the tests directory")
        sys.exit(1)
    
    main()