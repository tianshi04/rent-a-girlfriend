import sys
import os
import requests
import json
import urllib.parse
import hmac
import hashlib

# Add current folder to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from test_auth_helper import get_istio_headers

BASE_URL = "http://localhost:8000"
HASH_SECRET = "SHA512SECRETKEYEXAMPLE"

def main():
    print("=============================================================")
    print("   LUỒNG 3: NẠP TIỀN QUÁ VNPAY WEBHOOK (VNPAY-IPN)           ")
    print("=============================================================\n")

    # 1. Load active tokens
    tokens_filepath = os.path.join(os.path.dirname(os.path.abspath(__file__)), "active_tokens.json")
    if not os.path.exists(tokens_filepath):
        print(f"❌ Lỗi: Không tìm thấy file {tokens_filepath}. Hãy chạy test_flow_1_auth_upgrade.py trước.")
        sys.exit(1)
        
    with open(tokens_filepath, "r", encoding="utf-8") as f:
        state = json.load(f)
        
    client_token = state["client_token"]
    client_id = state["client_id"]
    client_headers = get_istio_headers(client_token)

    # 2. Truy vấn số dư ví trước khi nạp
    print("[Bước 1] Kiểm tra số dư ví ban đầu của Client...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={client_id}", timeout=5)
        resp.raise_for_status()
        wallet_before = resp.json()
        initial_balance = wallet_before["availableBalance"]
        print(f"-> Số dư khả dụng hiện tại: {initial_balance} coins")
    except Exception as e:
        print(f"❌ Lỗi kiểm tra ví tiền: {e}")
        sys.exit(1)

    # 3. Khởi tạo yêu cầu nạp tiền 500 coins qua API REST thực tế
    print("\n[Bước 2] Khởi tạo yêu cầu nạp tiền qua VNPay (POST /api/v1/finance/topup)...")
    payload = {
        "userId": client_id,
        "amount": 500
    }
    try:
        resp = requests.post(f"{BASE_URL}/api/v1/finance/topup", json=payload, headers=client_headers, timeout=5)
        resp.raise_for_status()
        topup_data = resp.json()
        payment_url = topup_data["paymentUrl"]
        print(f"-> [SUCCESS] Đã sinh payment URL thành công!")
        
        # Trích xuất txn_ref
        parsed = urllib.parse.urlparse(payment_url)
        params_dict = urllib.parse.parse_qs(parsed.query)
        txn_ref = params_dict["vnp_TxnRef"][0]
        print(f"-> Mã giao dịch trích xuất (vnp_TxnRef): {txn_ref}")
    except Exception as e:
        print(f"❌ Lỗi khởi tạo nạp tiền: {e}")
        sys.exit(1)

    # Kiểm tra database của finance_service để xác thực giao dịch ở trạng thái PENDING
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d finance_service -t -c "SELECT status FROM transactions WHERE transaction_id = \'{txn_ref}\';"'
        db_status = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_status == "PENDING":
            print("-> [DATABASE SUCCESS] Đã xác nhận giao dịch PENDING được tạo trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Trạng thái giao dịch trong DB không khớp: mong đợi PENDING, nhận được: {db_status}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra giao dịch trong DB: {e}")
        sys.exit(1)

    # 4. Gửi webhook IPN lỗi (chữ ký số sai)
    print("\n[Bước 3] Kiểm tra bảo mật: Gửi webhook IPN sai chữ ký...")
    bad_params = {
        "vnp_TmnCode": "DEMO2019",
        "vnp_Amount": "50000000", # 500 coins * 1000 RATE * 100 cents
        "vnp_TxnRef": txn_ref,
        "vnp_ResponseCode": "00",
        "vnp_SecureHash": "TAMPERED_SIGNATURE_HASH_EXAMPLE"
    }
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/vnpay-ipn", params=bad_params, timeout=5)
        resp.raise_for_status()
        res_json = resp.json()
        print(f"-> Kết quả từ service: {res_json}")
        if res_json.get("RspCode") == "97":
            print("-> [SUCCESS] Hệ thống đã phát hiện chữ ký sai và trả về mã lỗi 97 (Chữ ký không hợp lệ)!")
        else:
            print(f"❌ Lỗi: Hệ thống không phát hiện chữ ký sai (RspCode: {res_json.get('RspCode')})")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi gửi webhook IPN sai: {e}")
        sys.exit(1)

    # 5. Ký số HMAC-SHA512 thực tế và gửi webhook IPN thành công
    print("\n[Bước 4] Ký số HMAC-SHA512 và gửi webhook IPN thành công...")
    ipn_params = {
        "vnp_TmnCode": "DEMO2019",
        "vnp_Amount": "50000000",
        "vnp_TxnRef": txn_ref,
        "vnp_ResponseCode": "00",
        "vnp_TransactionNo": "999999",
        "vnp_OrderInfo": f"Topup Kano-Coin for transaction {txn_ref}"
    }
    
    # Sắp xếp tham số, urlencode và tính chữ ký HMAC-SHA512
    sorted_params = sorted(ipn_params.items())
    query_string = urllib.parse.urlencode(sorted_params)
    secure_hash = hmac.new(
        HASH_SECRET.encode("utf-8"),
        query_string.encode("utf-8"),
        hashlib.sha512
    ).hexdigest()
    ipn_params["vnp_SecureHash"] = secure_hash

    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/vnpay-ipn", params=ipn_params, timeout=5)
        resp.raise_for_status()
        res_json = resp.json()
        print(f"-> Kết quả từ service: {res_json}")
        if res_json.get("RspCode") == "00":
            print("-> [SUCCESS] Webhook xử lý nạp tiền thành công và trả về mã 00!")
        else:
            print(f"❌ Lỗi xử lý webhook nạp tiền (RspCode: {res_json.get('RspCode')})")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi gửi webhook IPN thành công: {e}")
        sys.exit(1)

    # Kiểm tra database của finance_service để xác thực giao dịch chuyển sang SUCCESS
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d finance_service -t -c "SELECT status FROM transactions WHERE transaction_id = \'{txn_ref}\';"'
        db_status = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_status == "SUCCESS":
            print("-> [DATABASE SUCCESS] Đã xác nhận giao dịch chuyển sang trạng thái SUCCESS trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Trạng thái giao dịch trong DB không khớp: mong đợi SUCCESS, nhận được: {db_status}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra giao dịch trong DB: {e}")
        sys.exit(1)

    # 6. Gửi trùng lặp IPN (Duplicate IPN Check)
    print("\n[Bước 5] Kiểm tra Idempotency: Gửi lại trùng lặp webhook IPN vừa xử lý...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/vnpay-ipn", params=ipn_params, timeout=5)
        resp.raise_for_status()
        res_json = resp.json()
        print(f"-> Kết quả từ service: {res_json}")
        if res_json.get("RspCode") == "02":
            print("-> [SUCCESS] Hệ thống phát hiện nạp trùng lặp và trả về mã lỗi 02 (Giao dịch đã xác nhận)!")
        else:
            print(f"❌ Lỗi: Hệ thống không chặn nạp lặp (RspCode: {res_json.get('RspCode')})")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi gửi trùng lặp IPN: {e}")
        sys.exit(1)

    # 7. Kiểm tra ví khả dụng đã được cộng tiền
    print("\n[Bước 6] Xác minh số dư ví khả dụng sau khi nạp thành công...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={client_id}", timeout=5)
        resp.raise_for_status()
        wallet_after = resp.json()
        new_balance = wallet_after["availableBalance"]
        print(f"-> Số dư ví sau khi nạp: {new_balance} coins")
        if new_balance == initial_balance + 500:
            print("-> [SUCCESS] Ví khả dụng tăng chính xác thêm 500 coins!")
        else:
            print(f"❌ Lỗi: Số dư ví không chính xác (mong muốn: {initial_balance + 500}, thực tế: {new_balance})")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi kiểm tra ví tiền sau nạp: {e}")
        sys.exit(1)

    # Kiểm tra database của finance_service để xác thực số dư ví đã tăng thêm 500 coins
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d finance_service -t -c "SELECT available_balance FROM wallets WHERE user_id = \'{client_id}\';"'
        db_balance = int(subprocess.check_output(cmd, shell=True).decode('utf-8').strip())
        if db_balance == initial_balance + 500:
            print("-> [DATABASE SUCCESS] Đã xác nhận số dư ví tăng chính xác thêm 500 coins trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Số dư ví trong DB không khớp: mong đợi {initial_balance + 500}, nhận được: {db_balance}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra số dư ví trong DB: {e}")
        sys.exit(1)

    print("\n=============================================================")
    print("   [SUCCESS] LUỒNG 3 HOÀN THÀNH THÀNH CÔNG!                  ")
    print("=============================================================\n")

if __name__ == "__main__":
    main()
