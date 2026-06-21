import sys
import os
import requests
import json
import time
import subprocess

# Add current folder to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from test_auth_helper import get_oauth_init_url, get_istio_headers, decode_jwt_payload

BASE_URL = "http://localhost:8000"

def verify_user_in_db(user_id):
    try:
        cmd = f'docker exec -i postgres psql -U postgres -d identity_service -t -c "SELECT COUNT(*) FROM user_accounts WHERE id = \'{user_id}\';"'
        res = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        return int(res) > 0
    except Exception as e:
        print(f"[ERROR] Failed to query Postgres: {e}")
        return False

def main():
    print("=============================================================")
    print("   LUỒNG 1: XÁC THỰC GOOGLE OAUTH & DUYỆT NÂNG CẤP COMPANION ")
    print("=============================================================\n")

    # 1. Sinh 2 link đăng nhập Google OAuth
    print("[Bước 1] Khởi tạo các liên kết đăng nhập Google OAuth thực tế...")
    try:
        client_auth_url = get_oauth_init_url()
        candidate_auth_url = get_oauth_init_url()
    except Exception as e:
        print(f"❌ Không thể kết nối với dịch vụ identity-service qua API Gateway: {e}")
        sys.exit(1)

    print("\n-------------------------------------------------------------")
    print("👉 Hãy copy link này dán vào trình duyệt để đăng nhập tài khoản CLIENT:")
    print(f"👉 Link CLIENT: {client_auth_url}")
    print("-------------------------------------------------------------")

    print("\n-------------------------------------------------------------")
    print("👉 Hãy copy link này dán vào trình duyệt để đăng nhập tài khoản CANDIDATE (Companion tương lai):")
    print(f"👉 Link CANDIDATE: {candidate_auth_url}")
    print("-------------------------------------------------------------\n")

    print("⚠️  Chú ý: Sau khi đăng nhập thành công ở mỗi link, trình duyệt của bạn sẽ hiển thị kết quả dạng JSON.")
    print("Hãy copy chuỗi 'accessToken' từ kết quả JSON và dán vào bên dưới.\n")

    client_token = input("🔑 Nhập Access Token của CLIENT: ").strip()
    candidate_token = input("🔑 Nhập Access Token của CANDIDATE: ").strip()

    if not client_token or not candidate_token:
        print("❌ Lỗi: Bạn chưa cung cấp đầy đủ tokens.")
        sys.exit(1)

    # 2. Giải mã JWT và trích xuất thông tin
    print("\n[Bước 2] Giải mã phần payload của JWT tokens...")
    try:
        client_claims = decode_jwt_payload(client_token)
        candidate_claims = decode_jwt_payload(candidate_token)
        
        client_id = client_claims.get("sub")
        client_email = client_claims.get("email")
        
        candidate_id = candidate_claims.get("sub")
        candidate_email = candidate_claims.get("email")
        
        print(f"-> CLIENT: ID={client_id}, Email={client_email}")
        print(f"-> CANDIDATE: ID={candidate_id}, Email={candidate_email}")
    except Exception as e:
        print(f"❌ Lỗi giải mã JWT token: {e}")
        sys.exit(1)

    # 3. Kiểm tra dữ liệu được thêm vào DB của identity-service
    print("\n[Bước 3] Kiểm tra tài khoản đã tồn tại trong DB của identity-service...")
    if verify_user_in_db(client_id) and verify_user_in_db(candidate_id):
        print("-> [SUCCESS] Cả hai tài khoản đã được lưu thành công vào cơ sở dữ liệu identity-service!")
    else:
        print("❌ Lỗi: Tài khoản chưa được tạo trong DB của identity-service.")
        sys.exit(1)

    # 4. Kiểm tra việc tự động onboard ví (wallet) của các tài khoản
    print("\n[Bước 4] Xác thực ví tiền tự động onboard thành công qua Kafka...")
    print("Chờ 3 giây để đảm bảo sự kiện Kafka đã được xử lý hoàn toàn...")
    time.sleep(3)
    
    # Query Client Wallet
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={client_id}", timeout=5)
        resp.raise_for_status()
        client_wallet = resp.json()
        print(f"-> Ví CLIENT hoạt động: Khả dụng={client_wallet['availableBalance']}, Đóng băng={client_wallet['frozenBalance']}")
        
        # Query Candidate Wallet
        resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={candidate_id}", timeout=5)
        resp.raise_for_status()
        candidate_wallet = resp.json()
        print(f"-> Ví CANDIDATE hoạt động: Khả dụng={candidate_wallet['availableBalance']}, Đóng băng={candidate_wallet['frozenBalance']}")
        print("-> [SUCCESS] Các ví tiền đã được khởi tạo tự động hoàn chỉnh!")
    except Exception as e:
        print(f"❌ Lỗi kiểm tra ví tiền: {e}")
        sys.exit(1)

    # 5. Client Candidate gửi yêu cầu nâng cấp Companion
    print("\n[Bước 5] Candidate gửi yêu cầu nâng cấp Companion...")
    candidate_headers = get_istio_headers(candidate_token)
    payload = {"reason": "Tôi muốn đăng ký làm companion chuyên nghiệp."}
    
    resp = requests.post(f"{BASE_URL}/api/v1/upgrade-requests", json=payload, headers=candidate_headers, timeout=5)
    if resp.status_code in (200, 201):
        print("-> [SUCCESS] Candidate gửi yêu cầu nâng cấp thành công!")
    else:
        print(f"❌ Lỗi gửi yêu cầu nâng cấp (Status {resp.status_code}): {resp.text}")
        sys.exit(1)

    # Kiểm tra database để đảm bảo yêu cầu nâng cấp đã được lưu với trạng thái PENDING
    try:
        cmd = f'docker exec -i postgres psql -U postgres -d identity_service -t -c "SELECT status FROM upgrade_requests WHERE user_id = \'{candidate_id}\' ORDER BY created_at DESC LIMIT 1;"'
        db_status = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_status == "PENDING":
            print("-> [DATABASE SUCCESS] Đã xác nhận yêu cầu nâng cấp trạng thái PENDING trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Trạng thái yêu cầu trong DB không hợp lệ: mong đợi PENDING, nhưng nhận được: {db_status}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Lỗi khi kết nối kiểm tra Database: {e}")
        sys.exit(1)

    # 6. Admin liệt kê danh sách yêu cầu và phê duyệt
    print("\n[Bước 6] Admin duyệt yêu cầu nâng cấp...")
    admin_headers = {
        "Content-Type": "application/json",
        "user-id": "a0000000-0000-0000-0000-000000000003",
        "user-role": "ADMIN",
        "user-email": "admin@rentgf.com",
        "user-status": "ACTIVE"
    }
    
    # Get requests list
    resp = requests.get(f"{BASE_URL}/api/v1/admin/upgrade-requests", headers=admin_headers, timeout=5)
    resp.raise_for_status()
    reqs_data = resp.json()
    items = reqs_data.get("data", [])
    
    # Find candidate request (using camelCase 'userId' and supporting 'UPGRADE_STATUS_PENDING' or 'PENDING')
    target_req = next((item for item in items if item.get("userId") == candidate_id and item.get("status") in ("PENDING", "UPGRADE_STATUS_PENDING")), None)
    if not target_req:
        print(f"❌ Lỗi: Không tìm thấy yêu cầu nâng cấp PENDING của Candidate ID={candidate_id} trong danh sách.")
        sys.exit(1)
        
    request_id = target_req["id"]
    print(f"-> Tìm thấy yêu cầu nâng cấp: ID={request_id}")
    
    # Approve request
    resp = requests.put(f"{BASE_URL}/api/v1/admin/upgrade-requests/{request_id}/approve", json={}, headers=admin_headers, timeout=5)
    if resp.status_code == 200:
        print("-> [SUCCESS] Admin phê duyệt yêu cầu thành công!")
    else:
        print(f"❌ Lỗi phê duyệt (Status {resp.status_code}): {resp.text}")
        sys.exit(1)

    # Kiểm tra database để đảm bảo yêu cầu đã chuyển thành APPROVED
    try:
        cmd = f'docker exec -i postgres psql -U postgres -d identity_service -t -c "SELECT status FROM upgrade_requests WHERE id = \'{request_id}\';"'
        db_status = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_status == "APPROVED":
            print("-> [DATABASE SUCCESS] Đã xác nhận yêu cầu nâng cấp được phê duyệt thành APPROVED trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Trạng thái yêu cầu trong DB không hợp lệ: mong đợi APPROVED, nhưng nhận được: {db_status}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Lỗi khi kết nối kiểm tra Database: {e}")
        sys.exit(1)

    # 7. Xác nhận vai trò mới của Candidate là COMPANION
    print("\n[Bước 7] Xác nhận vai trò tài khoản sau phê duyệt...")
    resp = requests.get(f"{BASE_URL}/api/v1/admin/accounts/{candidate_id}", headers=admin_headers, timeout=5)
    resp.raise_for_status()
    account_info = resp.json()
    new_role = account_info.get("role")
    print(f"-> Vai trò tài khoản hiện tại: {new_role}")
    if new_role in ("COMPANION", "ACCOUNT_ROLE_COMPANION"):
        print("-> [SUCCESS] Vai trò tài khoản chuyển sang COMPANION thành công!")
    else:
        print(f"❌ Lỗi: Tài khoản chưa được cập nhật vai trò.")
        sys.exit(1)

    # Kiểm tra database để đảm bảo vai trò của tài khoản trong bảng user_accounts đã cập nhật
    try:
        cmd = f'docker exec -i postgres psql -U postgres -d identity_service -t -c "SELECT role FROM user_accounts WHERE id = \'{candidate_id}\';"'
        db_role = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_role == "COMPANION":
            print("-> [DATABASE SUCCESS] Đã xác nhận vai trò tài khoản chuyển thành COMPANION trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Vai trò trong DB không hợp lệ: mong đợi COMPANION, nhưng nhận được: {db_role}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Lỗi khi kết nối kiểm tra Database: {e}")
        sys.exit(1)

    # 8. Lưu trạng thái token để các luồng sau tái sử dụng
    tokens_filepath = os.path.join(os.path.dirname(os.path.abspath(__file__)), "active_tokens.json")
    state_data = {
        "client_token": client_token,
        "candidate_token": candidate_token,
        "client_id": client_id,
        "client_email": client_email,
        "candidate_id": candidate_id,
        "candidate_email": candidate_email
    }
    with open(tokens_filepath, "w", encoding="utf-8") as f:
        json.dump(state_data, f, indent=2, ensure_ascii=False)
    print(f"\n💾 Lưu thông tin phiên kiểm thử vào {tokens_filepath}")
    
    print("\n=============================================================")
    print("   [SUCCESS] LUỒNG 1 HOÀN THÀNH THÀNH CÔNG!                  ")
    print("=============================================================\n")

if __name__ == "__main__":
    main()
