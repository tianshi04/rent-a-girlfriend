from test_auth_helper import decode_jwt_payload
import sys
import os
import requests
import json
import time

# Add current folder to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from test_auth_helper import get_istio_headers

BASE_URL = "http://localhost:8000"

def main():
    print("=============================================================")
    print("   LUỒNG 4: ĐẶT LỊCH HẸN, THỰC THI SAGA & CHAT INTERACTION    ")
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
    candidate_token = state["candidate_token"]
    candidate_id = state["candidate_id"] # upgraded to Companion

    client_headers = get_istio_headers(client_token)
    companion_headers = get_istio_headers(candidate_token)

    # 2. Truy vấn số dư ví Client trước khi đặt
    print("[Bước 1] Kiểm tra số dư ví khả dụng và đóng băng trước khi đặt...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={client_id}", timeout=5)
        resp.raise_for_status()
        wallet_before = resp.json()
        avail_before = wallet_before["availableBalance"]
        frozen_before = wallet_before["frozenBalance"]
        print(f"-> Ví Client trước đặt: Khả dụng={avail_before}, Đóng băng={frozen_before}")
    except Exception as e:
        print(f"❌ Lỗi truy cập ví tiền: {e}")
        sys.exit(1)

    # 3. Thực hiện đặt lịch hẹn thực tế (Scenario giá 200 coins)
    print("\n[Bước 2] Client gửi yêu cầu đặt lịch hẹn Companion (POST /api/v1/bookings)...")
    booking_payload = {
        "clientId": client_id,
        "companionId": candidate_id,
        "scenarioId": "s0000000-0000-0000-0000-000000000003",
        "startTime": "2026-06-28T10:00:00Z"
    }
    try:
        resp = requests.post(f"{BASE_URL}/api/v1/bookings", json=booking_payload, headers=client_headers, timeout=5)
        resp.raise_for_status()
        booking_data = resp.json()
        booking_id = booking_data["bookingId"]
        print(f"-> [SUCCESS] Đặt lịch thành công! Booking ID: {booking_id}, Trạng thái: {booking_data.get('status')}")
    except Exception as e:
        print(f"❌ Lỗi gửi đặt lịch: {e}")
        sys.exit(1)

    # Kiểm tra database booking_db để xác thực booking được tạo ở trạng thái PENDING hoặc PENDING_RESERVING
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d booking_db -t -c "SELECT status FROM bookings WHERE id = \'{booking_id}\';"'
        db_status = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_status in ("PENDING", "PENDING_RESERVING"):
            print(f"-> [DATABASE SUCCESS] Đã xác nhận booking trạng thái {db_status} được tạo trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Trạng thái booking trong DB không khớp: mong đợi PENDING hoặc PENDING_RESERVING, nhận được: {db_status}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra booking trong DB: {e}")
        sys.exit(1)

    # 4. Xác minh số dư bị đóng băng
    print("\n[Bước 3] Xác minh SAGA đóng băng 200 coins của Client...")
    print("Chờ 3 giây để SAGA xử lý outbox và cập nhật số dư...")
    time.sleep(3)
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={client_id}", timeout=5)
        resp.raise_for_status()
        wallet_after = resp.json()
        avail_after = wallet_after["availableBalance"]
        frozen_after = wallet_after["frozenBalance"]
        print(f"-> Ví Client sau đặt: Khả dụng={avail_after}, Đóng băng={frozen_after}")
        
        if avail_after == avail_before - 200 and frozen_after == frozen_before + 200:
            print("-> [SUCCESS] SAGA đóng băng chính xác 200 coins của Client!")
        else:
            print("❌ Lỗi: Phân bổ số dư ví sau khi đặt lịch không chính xác.")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi kiểm tra ví sau đặt: {e}")
        sys.exit(1)

    # Kiểm tra database của finance_service để xác thực số dư ví của Client
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d finance_service -t -c "SELECT available_balance, frozen_balance FROM wallets WHERE user_id = \'{client_id}\';"'
        res = subprocess.check_output(cmd, shell=True).decode('utf-8').strip().split('|')
        db_avail = int(res[0].strip())
        db_frozen = int(res[1].strip())
        if db_avail == avail_before - 200 and db_frozen == frozen_before + 200:
            print("-> [DATABASE SUCCESS] Đã xác nhận phân bổ ví (Khả dụng/Đóng băng) chính xác trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Số dư ví trong DB không khớp: mong đợi ({avail_before - 200}, {frozen_before + 200}), nhận được: ({db_avail}, {db_frozen})")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra số dư ví trong DB: {e}")
        sys.exit(1)

    # 5. Companion chấp nhận cuộc hẹn
    print("\n[Bước 4] Companion chấp nhận cuộc hẹn (PUT /api/v1/bookings/{id}/accept)...")
    try:
        resp = requests.put(f"{BASE_URL}/api/v1/bookings/{booking_id}/accept", json={}, headers=companion_headers, timeout=5)
        resp.raise_for_status()
        print("-> [SUCCESS] Companion đã gửi chấp nhận cuộc hẹn!")
    except Exception as e:
        print(f"❌ Lỗi chấp nhận cuộc hẹn: {e}")
        sys.exit(1)

    # Chờ 3 giây để SAGA xử lý chấp nhận cuộc hẹn và cập nhật trạng thái sang ACCEPTED
    print("Chờ SAGA xử lý chấp nhận cuộc hẹn...")
    db_status = "PENDING"
    for _ in range(10):
        try:
            import subprocess
            cmd = f'docker exec -i postgres psql -U postgres -d booking_db -t -c "SELECT status FROM bookings WHERE id = \'{booking_id}\';"'
            db_status = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
            if db_status in ("ACCEPTED", "BOOKING_STATUS_ACCEPTED"):
                break
        except Exception:
            pass
        time.sleep(0.5)

    if db_status in ("ACCEPTED", "BOOKING_STATUS_ACCEPTED"):
        print("-> [DATABASE SUCCESS] Đã xác nhận booking chuyển sang trạng thái ACCEPTED trong Database!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái booking trong DB không khớp: mong đợi ACCEPTED, nhận được: {db_status}")
        sys.exit(1)

    # 6. Xác minh tiền chuyển sang Escrow và chat được kích hoạt
    print("\n[Bước 5] Xác minh SAGA giải phóng tiền đóng băng sang Escrow và kích hoạt Chat...")
    print("Chờ 3 giây để SAGA xử lý chuyển tiền và tạo chat room...")
    time.sleep(3)
    try:
        # Check booking status
        resp = requests.get(f"{BASE_URL}/api/v1/bookings/{booking_id}", headers=client_headers, timeout=5)
        resp.raise_for_status()
        booking_detail = resp.json()
        print(f"-> Trạng thái booking hiện tại: {booking_detail.get('status')}")
        if booking_detail.get("status") not in ("ACCEPTED", "BOOKING_STATUS_ACCEPTED"):
            print(f"❌ Lỗi: Trạng thái cuộc hẹn không phải ACCEPTED (Thực tế: {booking_detail.get('status')})")
            sys.exit(1)
            
        # Check wallet frozen balance cleared
        resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={client_id}", timeout=5)
        resp.raise_for_status()
        wallet_final = resp.json()
        print(f"-> Ví Client cuối: Khả dụng={wallet_final['availableBalance']}, Đóng băng={wallet_final['frozenBalance']}")
        if wallet_final["frozenBalance"] != frozen_before:
            print("❌ Lỗi: Tiền đóng băng chưa được giải phóng sang Escrow.")
            sys.exit(1)
        print("-> [SUCCESS] Tiền đã được chuyển hoàn toàn sang Escrow quỹ trung gian!")
    except Exception as e:
        print(f"❌ Lỗi xác minh sau chấp nhận: {e}")
        sys.exit(1)

    # Kiểm tra database để xác nhận Escrow của giao dịch này đã được tạo
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d finance_service -t -c "SELECT amount, status FROM escrows WHERE booking_id = \'{booking_id}\';"'
        res = subprocess.check_output(cmd, shell=True).decode('utf-8').strip().split('|')
        db_amount = int(res[0].strip())
        db_status = res[1].strip()
        if db_amount == 200 and db_status == "HELD":
            print("-> [DATABASE SUCCESS] Đã xác nhận Escrow 200 coins ở trạng thái HELD được tạo trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Dữ liệu Escrow trong DB không khớp: mong đợi (200, HELD), nhận được: ({db_amount}, {db_status})")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra Escrow trong DB: {e}")
        sys.exit(1)

    # Kiểm tra database để xác nhận ví của Client đã giải phóng 200 coins đóng băng
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d finance_service -t -c "SELECT frozen_balance FROM wallets WHERE user_id = \'{client_id}\';"'
        db_frozen = int(subprocess.check_output(cmd, shell=True).decode('utf-8').strip())
        if db_frozen == frozen_before:
            print("-> [DATABASE SUCCESS] Đã xác nhận ví Client giải phóng tiền đóng băng thành công trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Số dư đóng băng trong DB không khớp: mong đợi {frozen_before}, nhận được: {db_frozen}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra ví Client trong DB: {e}")
        sys.exit(1)

    # 7. Kiểm tra kết nối chat
    print("\n[Bước 6] Client gửi tin nhắn vào phòng chat mới tạo...")
    
    # Lấy room_id từ database interaction_service bằng booking_id
    try:
        import subprocess
        cmd = f'docker exec -i postgres psql -U postgres -d interaction_service -t -c "SELECT room_id FROM chat_rooms WHERE booking_id = \'{booking_id}\';"'
        room_id = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if not room_id:
            print("❌ Lỗi: Không tìm thấy room_id trong database cho booking này.")
            sys.exit(1)
        print(f"-> [DATABASE SUCCESS] Đã xác định room_id: {room_id}")
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể lấy room_id từ DB: {e}")
        sys.exit(1)

    message_payload = {
        "text": "Chào Companion! Rất mong chờ buổi gặp mặt xem phim cùng bạn."
    }
    try:
        resp = requests.post(f"{BASE_URL}/api/v1/interaction/rooms/{room_id}/messages", json=message_payload, headers=client_headers, timeout=5)
        if resp.status_code in (200, 201):
            print("-> [SUCCESS] Gửi tin nhắn chat thành công! Phòng chat hoạt động hoàn hảo.")
            chat_msg = resp.json()
            print(f"-> Nội dung tin nhắn đã gửi: '{chat_msg.get('content')}'")
        else:
            print(f"❌ Lỗi gửi tin nhắn (Status {resp.status_code}): {resp.text}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi gửi tin nhắn: {e}")
        sys.exit(1)

    # Kiểm tra database interaction_service để xác thực tin nhắn được lưu thành công
    try:
        cmd = f'docker exec -i postgres psql -U postgres -d interaction_service -t -c "SELECT content FROM chat_messages WHERE room_id = \'{room_id}\' ORDER BY created_at DESC LIMIT 1;"'
        db_content = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_content == "Chào Companion! Rất mong chờ buổi gặp mặt xem phim cùng bạn.":
            print("-> [DATABASE SUCCESS] Đã xác nhận nội dung tin nhắn được lưu chính xác trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Nội dung tin nhắn trong DB không khớp: nhận được '{db_content}'")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra tin nhắn trong DB: {e}")
        sys.exit(1)

    # 8. Kiểm thử luồng lỗi
    print("\n[Bước 7] Kiểm thử các luồng lỗi nghiệp vụ...")

    # A. Trùng lịch hẹn của Client
    print("-> A. Client đặt trùng lịch hẹn khác chồng chéo thời gian...")
    overlapping_payload = {
        "clientId": client_id,
        "companionId": candidate_id,
        "scenarioId": "s0000000-0000-0000-0000-000000000003",
        "startTime": "2026-06-28T10:30:00Z" # Trùng khoảng 60 phút của cuộc hẹn trước
    }
    resp = requests.post(f"{BASE_URL}/api/v1/bookings", json=overlapping_payload, headers=client_headers, timeout=5)
    print(f"   * Status trả về: {resp.status_code}, Nội dung: {resp.text}")
    if resp.status_code >= 400:
        print("-> [SUCCESS] Hệ thống đã phát hiện trùng lịch Client và từ chối đặt lịch thành công!")
    else:
        print("❌ Lỗi: Hệ thống không phát hiện trùng lịch Client.")
        sys.exit(1)

    # B. Tài khoản không đủ số dư đặt lịch
    print("-> B. Tài khoản không đủ tiền khả dụng cố đặt lịch kịch bản đắt tiền...")
    try:
        # Cập nhật số dư ví w0000000-0000-0000-0000-000000000002 của user d0000000-0000-0000-0000-000000000002 về 0 để test không đủ số dư
        try:
            cmd = 'docker exec -i postgres psql -U postgres -d finance_service -c "UPDATE wallets SET available_balance = 0 WHERE wallet_id = \'w0000000-0000-0000-0000-000000000002\';"'
            subprocess.check_call(cmd, shell=True)
            print("-> [DATABASE SUCCESS] Đã cập nhật số dư ví w0000000-0000-0000-0000-000000000002 về 0 coins.")
        except Exception as e:
            print(f"❌ [DATABASE ERROR] Không thể cập nhật ví trong database: {e}")
            sys.exit(1)

        poor_headers = {
            "Content-Type": "application/json",
            "user-id": "d0000000-0000-0000-0000-000000000002",
            "user-role": "CLIENT",
            "user-email": "companion@rentgf.com",
            "user-status": "ACTIVE"
        }

        poor_payload = {
            "clientId": "d0000000-0000-0000-0000-000000000002",
            "companionId": candidate_id,
            "scenarioId": "s0000000-0000-0000-0000-000000000003",
            "startTime": "2026-06-29T14:00:00Z"
        }
        resp = requests.post(f"{BASE_URL}/api/v1/bookings", json=poor_payload, headers=poor_headers, timeout=5)
        resp.raise_for_status()
        poor_booking_id = resp.json()["bookingId"]
        print(f"-> [SUCCESS] Đã gửi yêu cầu đặt lịch thành công! Booking ID: {poor_booking_id}")

        print("Chờ 3 giây để SAGA đóng băng tiền xử lý và thất bại do thiếu số dư...")
        time.sleep(3)

        # Kiểm tra database để xác định booking tự động bị hủy (CANCELLED) do thiếu số dư
        cmd_check = f'docker exec -i postgres psql -U postgres -d booking_db -t -c "SELECT status FROM bookings WHERE id = \'{poor_booking_id}\';"'
        db_status = subprocess.check_output(cmd_check, shell=True).decode('utf-8').strip()
        if db_status in ("CANCELLED", "BOOKING_STATUS_CANCELLED"):
            print("-> [SUCCESS] Hệ thống phát hiện không đủ số dư và tự động hủy booking (CANCELLED) thành công!")
        else:
            print(f"❌ Lỗi: Booking không được hủy tự động, trạng thái hiện tại: {db_status}")
            
            # Khôi phục ví trước khi thoát lỗi
            try:
                cmd_restore = 'docker exec -i postgres psql -U postgres -d finance_service -c "UPDATE wallets SET available_balance = 500 WHERE wallet_id = \'w0000000-0000-0000-0000-000000000002\';"'
                subprocess.check_call(cmd_restore, shell=True)
            except Exception:
                pass
            sys.exit(1)

        # Khôi phục số dư ví w0000000-0000-0000-0000-000000000002 về 500 coins
        try:
            cmd_restore = 'docker exec -i postgres psql -U postgres -d finance_service -c "UPDATE wallets SET available_balance = 500 WHERE wallet_id = \'w0000000-0000-0000-0000-000000000002\';"'
            subprocess.check_call(cmd_restore, shell=True)
            print("-> [DATABASE SUCCESS] Đã khôi phục số dư ví về 500 coins.")
        except Exception as e:
            print(f"❌ [DATABASE ERROR] Không thể khôi phục ví trong database: {e}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi kiểm thử tài khoản poor client: {e}")
        # Đảm bảo khôi phục số dư nếu có bất kỳ lỗi nào xảy ra
        try:
            cmd_restore = 'docker exec -i postgres psql -U postgres -d finance_service -c "UPDATE wallets SET available_balance = 500 WHERE wallet_id = \'w0000000-0000-0000-0000-000000000002\';"'
            subprocess.check_call(cmd_restore, shell=True)
        except Exception:
            pass
        sys.exit(1)

    # Lưu thêm booking_id vào active_tokens.json để luồng sau (dispute) tái sử dụng
    try:
        with open(tokens_filepath, "r", encoding="utf-8") as f:
            state_data = json.load(f)
        state_data["booking_id"] = booking_id
        with open(tokens_filepath, "w", encoding="utf-8") as f:
            json.dump(state_data, f, indent=2, ensure_ascii=False)
        print("💾 Đã lưu booking_id vào active_tokens.json")
    except Exception as e:
        print(f"⚠️ Không thể lưu booking_id vào active_tokens.json: {e}")

    print("\n=============================================================")
    print("   [SUCCESS] LUỒNG 4 HOÀN THÀNH THÀNH CÔNG!                  ")
    print("=============================================================\n")

if __name__ == "__main__":
    main()
