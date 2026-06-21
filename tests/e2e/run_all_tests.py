import sys
import os
import subprocess

def run_db_seeder():
    print("=============================================================")
    print("   [SETUP] Làm sạch và nạp lại cơ sở dữ liệu mẫu...          ")
    print("=============================================================")
    try:
        # Run docker-compose run --rm db-seeder
        print("-> Đang khởi chạy db-seeder container...")
        subprocess.check_call(["docker", "compose", "run", "--rm", "db-seeder"])
        print("-> [SUCCESS] Database đã được phục hồi trạng thái hạt giống sạch sẽ!\n")
        return True
    except Exception as e:
        print(f"❌ Lỗi khi khởi chạy db-seeder: {e}")
        return False

def main():
    # 1. Reset database to fresh seed
    if not run_db_seeder():
        print("❌ Lỗi: Setup database thất bại. Dừng kiểm thử.")
        sys.exit(1)

    # 2. Định nghĩa các script kiểm thử cần chạy
    current_dir = os.path.dirname(os.path.abspath(__file__))
    
    test_flows = [
        ("Flow 1: Google OAuth Init & Companion Upgrade", "test_flow_1_auth_upgrade.py"),
        ("Flow 2: Companion Catalogue & Profile", "test_flow_2_companions.py"),
        ("Flow 3: VNPay Topup & Secure Webhook (IPN)", "test_flow_3_topup.py"),
        ("Flow 4: Booking SAGA flow & Chat Room", "test_flow_4_booking_saga.py"),
        ("Flow 5: Dispute Listing & Saga status", "test_flow_5_disputes.py"),
        ("Flow 6: Dispute Resilience & Idempotency", "test_flow_6_resilience_idempotency.py"),
        ("Flow 7: System Errors & Invariants", "test_flow_7_error_cases.py"),
    ]

    results = {}
    print("=============================================================")
    print("   BẮT ĐẦU CHẠY BỘ KIỂM THỬ E2E TOÀN DIỆN                    ")
    print("=============================================================\n")

    for name, filename in test_flows:
        script_path = os.path.join(current_dir, filename)
        print(f"🚀 Đang chạy: {name} ({filename})...")
        
        # Run script inheriting stdin/stdout/stderr for interactive inputs
        proc = subprocess.run([sys.executable, script_path])
        
        if proc.returncode == 0:
            results[name] = "PASSED"
            print(f"✅ {name}: HOÀN THÀNH THÀNH CÔNG\n")
        else:
            results[name] = "FAILED"
            print(f"❌ {name}: THẤT BẠI (Mã thoát: {proc.returncode})\n")
            # Stop execution of subsequent tests if a preceding one fails
            print("⚠️  Một luồng kiểm thử đã thất bại. Dừng chạy các luồng tiếp theo.")
            break

    # 3. In báo cáo tổng hợp
    print("=============================================================")
    print("   BÁO CÁO KẾT QUẢ KIỂM THỬ E2E TOÀN DIỆN                    ")
    print("=============================================================")
    print(f"{'Luồng Nghiệp vụ Kiểm thử':<50} | Trạng thái")
    print("-" * 65)
    
    passed_all = True
    for name, filename in test_flows:
        status = results.get(name, "SKIPPED")
        if status == "FAILED" or status == "SKIPPED":
            passed_all = False
        color_status = f"🟢 {status}" if status == "PASSED" else (f"🔴 {status}" if status == "FAILED" else f"⚪ {status}")
        print(f"{name:<50} | {color_status}")
    print("-" * 65)

    if passed_all:
        print("🎉 [SUCCESS] TẤT CẢ CÁC LUỒNG THỬ NGHIỆM ĐÃ VƯỢT QUA THÀNH CÔNG!")
        sys.exit(0)
    else:
        print("❌ [FAILURE] CÓ LUỒNG THỬ NGHIỆM BỊ THẤT BẠI HOẶC BỊ BỎ QUA.")
        sys.exit(1)

if __name__ == "__main__":
    main()
