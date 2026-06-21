import sys
import os
import requests
import json
import subprocess

BASE_URL = "http://localhost:8000"

def create_companion_profile_db(companion_id):
    try:
        # Insert profile
        cmd_profile = (
            f'docker exec -i postgres psql -U postgres -d profile_service -c '
            f'"INSERT INTO companion_profiles (companion_id, user_id, display_name, intro_text, status, available_cities, avatar_url, created_at, updated_at) '
            f'VALUES (\'{companion_id}\', \'{companion_id}\', \'Companion Candidate\', \'Giới thiệu về Companion Candidate\', \'APPROVED\', \'[\\\"Hanoi\\\"]\', \'http://localhost:9000/avatar.jpg\', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) '
            f'ON CONFLICT (companion_id) DO NOTHING;"'
        )
        subprocess.check_call(cmd_profile, shell=True)

        # Insert scenario (Price: 200 coins, Duration: 60 mins)
        scenario_id = "s0000000-0000-0000-0000-000000000003"
        cmd_scenario = (
            f'docker exec -i postgres psql -U postgres -d profile_service -c '
            f'"INSERT INTO scenarios (scenario_id, companion_id, title, description, price, duration_minutes, status, created_at) '
            f'VALUES (\'{scenario_id}\', \'{companion_id}\', \'Hẹn hò xem phim\', \'Một buổi hẹn hò xem phim lãng mạn\', 200, 60, \'ACTIVE\', CURRENT_TIMESTAMP) '
            f'ON CONFLICT (scenario_id) DO NOTHING;"'
        )
        subprocess.check_call(cmd_scenario, shell=True)
        print("-> [SUCCESS] Đã khởi tạo Companion Profile và Scenario thành công trong Database!")
        return True
    except Exception as e:
        print(f"[ERROR] Failed to insert profile into DB: {e}")
        return False

def main():
    print("=============================================================")
    print("   LUỒNG 2: TRUY VẤN DANH MỤC & HỒ SƠ COMPANION             ")
    print("=============================================================\n")

    # 1. Load active tokens
    tokens_filepath = os.path.join(os.path.dirname(os.path.abspath(__file__)), "active_tokens.json")
    if not os.path.exists(tokens_filepath):
        print(f"❌ Lỗi: Không tìm thấy file {tokens_filepath}. Hãy chạy test_flow_1_auth_upgrade.py trước.")
        sys.exit(1)
        
    with open(tokens_filepath, "r", encoding="utf-8") as f:
        state = json.load(f)
        
    candidate_id = state["candidate_id"]
    candidate_token = state["candidate_token"]

    # 2. Tạo Companion Profile và Scenario cho Candidate
    print("[Bước 1] Khởi tạo Companion Profile & Scenario cho Companion mới nâng cấp...")
    if not create_companion_profile_db(candidate_id):
        print("❌ Lỗi: Không thể khởi tạo Profile trong DB.")
        sys.exit(1)

    # 3. Tìm kiếm trong danh mục Catalogue
    print("\n[Bước 2] Truy vấn danh sách Companion công khai (Catalogue)...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/companions", timeout=5)
        resp.raise_for_status()
        catalogue_data = resp.json()
        items = catalogue_data.get("data", [])
        print(f"-> Tổng số companion tìm thấy: {len(items)}")
        
        my_comp = next((item for item in items if item["companionId"] == candidate_id), None)
        if my_comp:
            print(f"-> [SUCCESS] Tìm thấy Companion mới: {my_comp['displayName']} (Trạng thái: {my_comp.get('status')})")
        else:
            print(f"❌ Lỗi: Không tìm thấy Companion ID={candidate_id} mới nâng cấp trong catalogue.")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi truy cập danh mục: {e}")
        sys.exit(1)

    # 4. Xem chi tiết Profile công khai của Companion
    print("\n[Bước 3] Xem chi tiết hồ sơ công khai của Companion mới nâng cấp...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/companions/{candidate_id}", timeout=5)
        resp.raise_for_status()
        profile_detail = resp.json()
        print(f"-> Tên hiển thị: {profile_detail['displayName']}")
        print(f"-> Giới thiệu: '{profile_detail['introText']}'")
        print(f"-> Thành phố hỗ trợ: {profile_detail['availableCities']}")
        
        scenarios = profile_detail.get("scenarios", [])
        print(f"-> Số lượng kịch bản hẹn hò (Scenarios): {len(scenarios)}")
        for sc in scenarios:
            print(f"   * Kịch bản: {sc['title']} - Giá: {sc['price']} coins - Thời lượng: {sc['duration']} phút")
            
        if len(scenarios) > 0:
            print("-> [SUCCESS] Lấy chi tiết hồ sơ và Scenarios thành công!")
        else:
            print("❌ Lỗi: Không lấy được Scenarios của companion.")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi truy vấn chi tiết hồ sơ: {e}")
        sys.exit(1)

    # 5. Xem thông tin cá nhân của Companion qua /profile/me
    print("\n[Bước 4] Xem thông tin cá nhân của chính Companion...")
    companion_headers = {
        "Content-Type": "application/json",
        "user-id": candidate_id,
        "user-role": "COMPANION",
        "user-email": state["candidate_email"],
        "user-status": "ACTIVE"
    }
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/profile/me", headers=companion_headers, timeout=5)
        resp.raise_for_status()
        my_profile = resp.json()
        print(f"-> [SUCCESS] Lấy hồ sơ cá nhân thành công: ID={my_profile['companionId']}, Tên={my_profile['displayName']}")
    except Exception as e:
        print(f"❌ Lỗi lấy hồ sơ cá nhân: {e}")
        sys.exit(1)

    print("\n=============================================================")
    print("   [SUCCESS] LUỒNG 2 HOÀN THÀNH THÀNH CÔNG!                  ")
    print("=============================================================\n")

if __name__ == "__main__":
    main()
