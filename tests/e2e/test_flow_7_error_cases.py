import sys
import os
import time
import subprocess
import json
import requests

# Add current folder to path
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from test_auth_helper import get_istio_headers

BASE_URL = "http://localhost:8000"

def run_sql(db, sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "postgres", "-d", db, "-t", "-c", sql]
    res = subprocess.run(cmd, capture_output=True, text=True, encoding='utf-8')
    if res.returncode != 0:
        print(f"❌ SQL Error in {db}: {res.stderr}")
        sys.exit(1)
    return res.stdout.strip()

def run_python_in_dispute_container(code):
    cmd = ["docker", "exec", "-i", "dispute-service", "python"]
    res = subprocess.run(cmd, input=code, capture_output=True, text=True, encoding='utf-8')
    if res.returncode != 0:
        print(f"❌ Container execution error: {res.stderr}")
        return None
    return res.stdout.strip()

def main():
    print("=============================================================")
    print("   LUỒNG 7: KIỂM THỬ CÁC CA LỖI HỆ THỐNG & INVARIANTS (E2E)  ")
    print("=============================================================\n")

    # 1. Load active tokens (chỉ cần thiết cho Client headers xác thực)
    tokens_filepath = os.path.join(os.path.dirname(os.path.abspath(__file__)), "active_tokens.json")
    if not os.path.exists(tokens_filepath):
        print(f"❌ Lỗi: Không tìm thấy file {tokens_filepath}. Hãy chạy test_flow_1_auth_upgrade.py trước.")
        sys.exit(1)
        
    with open(tokens_filepath, "r", encoding="utf-8") as f:
        state = json.load(f)
        
    client_token = state["client_token"]
    client_id = state["client_id"]
    client_headers = get_istio_headers(client_token)

    # -------------------------------------------------------------------------
    # CASE 1: TÀI KHOẢN KHÔNG ĐỦ TIỀN ĐẶT LỊCH (ErrInsufficientFunds)
    # -------------------------------------------------------------------------
    print("👉 [CASE 1] Kiểm tra Client đặt lịch khi số dư ví bằng 0...")
    
    # Reset ví client về 0
    run_sql("finance_service", f"UPDATE wallets SET available_balance = 0, frozen_balance = 0 WHERE user_id = '{client_id}';")
    print("   * Đã reset ví Client về 0 coins.")

    # Gửi yêu cầu đặt lịch (sử dụng gói Chizuru s0000000-0000-0000-0000-000000000001 giá 100 coins)
    booking_payload = {
        "clientId": client_id,
        "companionId": "d0000000-0000-0000-0000-000000000002",
        "scenarioId": "s0000000-0000-0000-0000-000000000001",
        "startTime": "2026-06-29T10:00:00Z"
    }
    
    try:
        resp = requests.post(f"{BASE_URL}/api/v1/bookings", json=booking_payload, headers=client_headers, timeout=5)
        resp.raise_for_status()
        booking_data = resp.json()
        booking_id = booking_data["bookingId"]
        init_status = booking_data.get("status")
        print(f"   * Booking khởi tạo thành công: ID={booking_id}, Trạng thái ban đầu={init_status} (Mong đợi: PENDING_RESERVING)")
        
        # Chờ 3 giây để SAGA chạy ngầm đóng băng tiền và nhận tin thất bại từ finance-service
        print("   * Chờ SAGA đóng băng tiền chạy ngầm...")
        time.sleep(3.0)
        
        # Kiểm tra trạng thái Booking trong DB
        final_status = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id}';")
        print(f"   * Trạng thái Booking cuối cùng trong DB: {final_status} (Mong đợi: CANCELLED)")
        
        if final_status == "CANCELLED":
            print("-> [SUCCESS] Đã xác nhận Booking tự động chuyển sang CANCELLED do không đủ tiền đóng băng!")
        else:
            print(f"❌ Lỗi: Trạng thái Booking là '{final_status}', mong đợi 'CANCELLED'")
            sys.exit(1)
            
    except Exception as e:
        print(f"❌ Lỗi đặt lịch: {e}")
        sys.exit(1)


    # -------------------------------------------------------------------------
    # CASE 2: COMPANION TRÙNG LỊCH HẸN KHI CHẤP NHẬN (ErrCompanionBookingOverlap)
    # -------------------------------------------------------------------------
    print("\n👉 [CASE 2] Kiểm tra lỗi trùng lịch hẹn của Companion khi chấp nhận cuộc hẹn...")
    
    # 2.1 Thiết lập Booking 3 là ACCEPTED
    run_sql("booking_db", "UPDATE bookings SET status = 'ACCEPTED' WHERE id = 'b0000000-0000-0000-0000-000000000003';")
    
    # 2.2 Chèn thủ công một booking b0000000-0000-0000-0000-000000000009 ở trạng thái PENDING bị trùng thời gian với Booking 3
    # Booking 3 trong seeds: start_time = CURRENT_TIMESTAMP + 2 days, duration = 120 minutes (end_time = start_time + 2h)
    # Chèn Booking 9: start_time = CURRENT_TIMESTAMP + 2 days + 30 minutes, duration = 60 minutes.
    insert_booking_9 = """
    INSERT INTO bookings (id, client_id, companion_id, scenario_price, scenario_duration, start_time, end_time, status, version, created_at, updated_at) VALUES
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 100, 60, CURRENT_TIMESTAMP + INTERVAL '2 days' + INTERVAL '30 minutes', CURRENT_TIMESTAMP + INTERVAL '2 days' + INTERVAL '90 minutes', 'PENDING', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    ON CONFLICT (id) DO UPDATE SET status = 'PENDING', start_time = CURRENT_TIMESTAMP + INTERVAL '2 days' + INTERVAL '30 minutes', end_time = CURRENT_TIMESTAMP + INTERVAL '2 days' + INTERVAL '90 minutes';
    """
    run_sql("booking_db", insert_booking_9)
    print("   * Đã chèn booking trùng lịch hẹn (Booking 9) ở trạng thái PENDING.")

    # 2.3 Companion Chizuru chấp nhận cuộc hẹn này
    chizuru_headers = {
        "Content-Type": "application/json",
        "user-id": "d0000000-0000-0000-0000-000000000002",
        "user-role": "COMPANION",
        "user-email": "chizuru@rentgf.com",
        "user-status": "ACTIVE"
    }
    
    print("   * Companion Chizuru gửi yêu cầu chấp nhận Booking 9 bị trùng lịch...")
    resp = requests.put(f"{BASE_URL}/api/v1/bookings/b0000000-0000-0000-0000-000000000009/accept", json={}, headers=chizuru_headers, timeout=5)
    print(f"   * Status Code nhận về: {resp.status_code}")
    print(f"   * Chi tiết lỗi: {resp.text.strip()}")
    
    if resp.status_code == 400 or resp.status_code == 500:
        if "companion already has an accepted booking" in resp.text or "failed precondition" in resp.text or "overlap" in resp.text:
            print("-> [SUCCESS] Đã chặn trùng lịch Companion chính xác!")
        else:
            print(f"❌ Lỗi: Nội dung trả về không khớp lý do trùng lịch. Chi tiết: {resp.text}")
            sys.exit(1)
    else:
        print(f"❌ Lỗi: Kỳ vọng lỗi 400/500 nhưng nhận được {resp.status_code}")
        sys.exit(1)


    # -------------------------------------------------------------------------
    # CASE 3: RÀNG BUỘC DUY NHẤT TRANH CHẤP ([INV-D01])
    # -------------------------------------------------------------------------
    print("\n👉 [CASE 3] Kiểm tra ràng buộc duy nhất tranh chấp active ([INV-D01])...")
    
    # 3.1 Đưa tranh chấp của Booking 2 về trạng thái OPEN
    run_sql("dispute_service", f"DELETE FROM saga_states WHERE booking_id = 'b0000000-0000-0000-0000-000000000002';")
    run_sql("dispute_service", f"DELETE FROM disputes WHERE booking_id = 'b0000000-0000-0000-0000-000000000002';")
    
    # Tạo tranh chấp ban đầu
    insert_dispute_sql = f"""
    INSERT INTO disputes (dispute_id, booking_id, reporter_id, accused_id, reason, status, resolution, notes, admin_id, version, created_at, updated_at)
    VALUES ('e0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'MISCONDUCT', 'OPEN', NULL, NULL, NULL, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
    """
    run_sql("dispute_service", insert_dispute_sql)
    print("   * Đã khởi tạo 1 tranh chấp trạng thái OPEN cho Booking 2.")

    # 3.2 Gọi gRPC CreateReport lần thứ 2 cho Booking 2
    create_report_again_code = f"""
import asyncio
import grpc
import sys
from gen.dispute.v1.service import dispute_service_pb2_grpc
from gen.dispute.v1.messages.create_report_request_pb2 import CreateReportRequest, EvidenceItem

async def run():
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = dispute_service_pb2_grpc.DisputeServiceStub(channel)
        metadata = (
            ('user-id', 'c0000000-0000-0000-0000-000000000001'),
            ('user-role', 'CLIENT'),
            ('user-email', 'client@rentgf.com'),
            ('user-status', 'ACTIVE'),
        )
        req = CreateReportRequest(
            booking_id='b0000000-0000-0000-0000-000000000002',
            reporter_id='c0000000-0000-0000-0000-000000000001',
            accused_id='d0000000-0000-0000-0000-000000000002',
            reason='MISCONDUCT',
            evidences=[EvidenceItem(type='TEXT', content='Duplicate report')]
        )
        try:
            await stub.CreateReport(req, metadata=metadata)
            print("SUCCESS")
        except grpc.RpcError as e:
            print(f"FAILED|{{e.code()}}|{{e.details()}}")

asyncio.run(run())
"""
    grpc_out = run_python_in_dispute_container(create_report_again_code)
    print(f"   * Kết quả gRPC gọi lần 2: {grpc_out}")
    
    if "FAILED" in grpc_out and "INV-D01" in grpc_out:
        print("-> [SUCCESS] Đã chặn tạo trùng khiếu nại đang active ([INV-D01]) chính xác!")
    else:
        print(f"❌ Lỗi: Trực gRPC không chặn trùng báo cáo. Output: {grpc_out}")
        sys.exit(1)


    # -------------------------------------------------------------------------
    # CASE 4: RÀNG BUỘC TÍNH BẤT BIẾN KHI TRANH CHẤP ĐÃ GIẢI QUYẾT ([INV-D02])
    # -------------------------------------------------------------------------
    print("\n👉 [CASE 4] Kiểm tra tính bất biến khi tranh chấp đã được giải quyết ([INV-D02])...")
    
    # 4.1 Cập nhật trạng thái tranh chấp về REFUNDED (trạng thái cuối)
    run_sql("dispute_service", "UPDATE disputes SET status = 'REFUNDED', resolution = 'REFUND_CLIENT' WHERE dispute_id = 'e0000000-0000-0000-0000-000000000002';")
    print("   * Đã cập nhật Tranh chấp sang trạng thái cuối REFUNDED.")

    # 4.2 Gọi gRPC ResolveDispute lần 2 cho tranh chấp này
    resolve_again_code = f"""
import asyncio
import grpc
from gen.dispute.v1.service import dispute_service_pb2_grpc
from gen.dispute.v1.messages.resolve_dispute_request_pb2 import ResolveDisputeRequest

async def run():
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = dispute_service_pb2_grpc.DisputeServiceStub(channel)
        metadata = (
            ('user-id', 'a0000000-0000-0000-0000-000000000003'),
            ('user-role', 'ADMIN'),
            ('user-email', 'admin@rentgf.com'),
            ('user-status', 'ACTIVE'),
        )
        req = ResolveDisputeRequest(
            dispute_id='e0000000-0000-0000-0000-000000000002',
            admin_id='a0000000-0000-0000-0000-000000000003',
            resolution='REFUND_CLIENT',
            notes='Resolve twice'
        )
        try:
            await stub.ResolveDispute(req, metadata=metadata)
            print("SUCCESS")
        except grpc.RpcError as e:
            print(f"FAILED|{{e.code()}}|{{e.details()}}")

asyncio.run(run())
"""
    grpc_out_2 = run_python_in_dispute_container(resolve_again_code)
    print(f"   * Kết quả gRPC gọi giải quyết lần 2: {grpc_out_2}")
    
    if "FAILED" in grpc_out_2 and "INV-D02" in grpc_out_2:
        print("-> [SUCCESS] Đã chặn thay đổi trạng thái tranh chấp cuối ([INV-D02]) chính xác!")
    else:
        print(f"❌ Lỗi: Trực gRPC không chặn giải quyết lại tranh chấp đã đóng. Output: {grpc_out_2}")
        sys.exit(1)

    print("\n=============================================================")
    print("   [SUCCESS] LUỒNG 7 HOÀN THÀNH XÁC MINH CÁC LỖI & INVARIANTS! ")
    print("=============================================================\n")

if __name__ == "__main__":
    main()
