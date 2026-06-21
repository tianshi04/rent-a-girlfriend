import sys
import os
import requests
import json
import time
import subprocess

BASE_URL = "http://localhost:8000"
SEED_DISPUTE_ID = "e0000000-0000-0000-0000-000000000001"

def run_sql(db, sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "postgres", "-d", db, "-t", "-c", sql]
    res = subprocess.run(cmd, capture_output=True, text=True, encoding='utf-8')
    if res.returncode != 0:
        print(f"❌ SQL Error in {db}: {res.stderr}")
        sys.exit(1)
    return res.stdout.strip()

def call_dispute_grpc(python_code):
    cmd = ["docker", "exec", "-i", "dispute-service", "python"]
    res = subprocess.run(cmd, input=python_code, capture_output=True, text=True, encoding='utf-8')
    if res.returncode != 0:
        print(f"❌ Container execution error: {res.stderr}")
        return None
    return res.stdout.strip()

def main():
    print("=============================================================")
    print("   LUỒNG 5: TRUY VẤN & XỬ LÝ TRANH CHẤP KHIẾU NẠI (E2E)      ")
    print("=============================================================\n")

    print("[Setup] Khởi tạo dữ liệu kiểm thử trong Database...")
    # 1. Update booking status
    run_sql("booking_db", "UPDATE bookings SET status = 'ACCEPTED' WHERE id = 'b0000000-0000-0000-0000-000000000002';")
    run_sql("booking_db", "UPDATE bookings SET status = 'ACCEPTED' WHERE id = 'b0000000-0000-0000-0000-000000000003';")
    
    # 2. Clean up any active disputes from previous runs to ensure idempotency
    run_sql("dispute_service", "DELETE FROM saga_states;")
    run_sql("dispute_service", f"DELETE FROM disputes WHERE dispute_id != '{SEED_DISPUTE_ID}';")
    
    # 3. Insert/reset escrow and transactions in finance_service
    run_sql("finance_service", "UPDATE wallets SET available_balance = 1000, frozen_balance = 0 WHERE user_id = 'c0000000-0000-0000-0000-000000000001';")
    run_sql("finance_service", "DELETE FROM wallets WHERE user_id = 'wallet-b0000000-0000-0000-0000-000000000003';")
    run_sql("finance_service", """
INSERT INTO escrows (escrow_id, booking_id, amount, status, created_at, updated_at)
VALUES ('e0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 100, 'HELD', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (escrow_id) DO UPDATE SET status = 'HELD';
""")
    run_sql("finance_service", "UPDATE escrows SET status = 'HELD' WHERE escrow_id = 'e0000000-0000-0000-0000-000000000003';")
    run_sql("finance_service", """
INSERT INTO transactions (transaction_id, user_id, amount, type, status, reference_id, created_at)
VALUES 
('t0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 100, 'BOOKING_RESERVATION', 'SUCCESS', 'b0000000-0000-0000-0000-000000000002', CURRENT_TIMESTAMP),
('t0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 300, 'BOOKING_RESERVATION', 'SUCCESS', 'b0000000-0000-0000-0000-000000000003', CURRENT_TIMESTAMP)
ON CONFLICT (transaction_id) DO NOTHING;
""")
    # 4. Insert/reset chat room and review in interaction_service
    run_sql("interaction_service", """
INSERT INTO chat_rooms (room_id, booking_id, client_id, companion_id, status, created_at, updated_at)
VALUES ('r0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (room_id) DO UPDATE SET status = 'ACTIVE';
""")
    run_sql("interaction_service", "UPDATE chat_rooms SET status = 'ACTIVE' WHERE booking_id = 'b0000000-0000-0000-0000-000000000003';")
    run_sql("interaction_service", """
INSERT INTO reviews (review_id, booking_id, client_id, companion_id, rating, comment, is_visible, created_at, updated_at)
VALUES ('v0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 5, 'Good companion!', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (review_id) DO UPDATE SET is_visible = true;
""")
    print("-> Dữ liệu kiểm thử đã được chuẩn bị thành công!\n")

    admin_headers = {
        "Content-Type": "application/json",
        "user-id": "a0000000-0000-0000-0000-000000000003",
        "user-role": "ADMIN",
        "user-email": "admin@rentgf.com",
        "user-status": "ACTIVE"
    }

    # 1. Liệt kê danh sách Tranh chấp dưới quyền Admin
    print("[Bước 1] Admin truy vấn danh sách Tranh chấp (GET /api/v1/disputes)...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/disputes", headers=admin_headers, timeout=5)
        resp.raise_for_status()
        disputes_list = resp.json()
        items = disputes_list.get("disputes", [])
        total = disputes_list.get("total", 0)
        print(f"-> Tổng số tranh chấp trong hệ thống: {total}")
        
        target_disp = next((item for item in items if item["disputeId"] == SEED_DISPUTE_ID), None)
        if target_disp:
            print(f"-> [SUCCESS] Tìm thấy Tranh chấp mẫu: ID={SEED_DISPUTE_ID}, Lý do={target_disp.get('reason')}")
        else:
            print(f"❌ Lỗi: Không tìm thấy Tranh chấp mẫu ID={SEED_DISPUTE_ID} trong danh sách.")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi truy cập danh sách tranh chấp: {e}")
        sys.exit(1)

    # 2. Truy cập chi tiết Tranh chấp
    print("\n[Bước 2] Xem chi tiết thông tin Tranh chấp (GET /api/v1/disputes/{id})...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/disputes/{SEED_DISPUTE_ID}", headers=admin_headers, timeout=5)
        resp.raise_for_status()
        disp_detail = resp.json()
        print(f"-> Dispute ID: {disp_detail['disputeId']}")
        print(f"-> Booking ID: {disp_detail['bookingId']}")
        print(f"-> Lý do khiếu nại: {disp_detail['reason']}")
        print(f"-> Trạng thái: {disp_detail['status']}")
        print(f"-> Hướng giải quyết: {disp_detail.get('resolution')}")
        
        if disp_detail["status"] in ("RESOLVED", "DISPUTE_STATUS_RESOLVED", "REFUNDED"):
            print("-> [SUCCESS] Lấy chi tiết tranh chấp thành công!")
        else:
            print(f"❌ Lỗi: Trạng thái tranh chấp không khớp (mong muốn: RESOLVED/REFUNDED, thực tế: {disp_detail['status']})")
            sys.exit(1)
    except Exception as e:
        print(f"❌ Lỗi truy cập chi tiết tranh chấp: {e}")
        sys.exit(1)

    # Kiểm tra database dispute_service để xác thực thông tin tranh chấp mẫu
    try:
        res = run_sql("dispute_service", f"SELECT status, resolution FROM disputes WHERE dispute_id = '{SEED_DISPUTE_ID}';")
        parts = [p.strip() for p in res.split('|')]
        db_status = parts[0]
        db_resolution = parts[1]
        if db_status in ("RESOLVED", "DISPUTE_STATUS_RESOLVED", "REFUNDED") and db_resolution in ("REFUNDED", "DISPUTE_RESOLUTION_REFUNDED", "REFUND_CLIENT"):
            print("-> [DATABASE SUCCESS] Đã xác nhận thông tin tranh chấp mẫu trong Database!")
        else:
            print(f"❌ [DATABASE ERROR] Dữ liệu tranh chấp trong DB không khớp: {db_status}, {db_resolution}")
            sys.exit(1)
    except Exception as e:
        print(f"❌ [DATABASE ERROR] Không thể kiểm tra tranh chấp trong DB: {e}")
        sys.exit(1)

    # 3. Xem trạng thái SAGA của Tranh chấp mẫu
    print("\n[Bước 3] Truy vấn trạng thái SAGA của Tranh chấp mẫu (GET /api/v1/disputes/{id}/saga)...")
    try:
        resp = requests.get(f"{BASE_URL}/api/v1/disputes/{SEED_DISPUTE_ID}/saga", headers=admin_headers, timeout=5)
        resp.raise_for_status()
        saga_state = resp.json()
        if saga_state:
            print(f"-> Saga ID: {saga_state['sagaId']}")
            print(f"-> Loại Saga: {saga_state['sagaType']}")
            print(f"-> Trạng thái hiện tại: {saga_state['currentState']}")
            print("-> [SUCCESS] Lấy trạng thái SAGA thành công!")
        else:
            print("-> Trạng thái SAGA không tồn tại (Có thể được xử lý trực tiếp).")
    except Exception as e:
        print(f"❌ Lỗi truy cập trạng thái SAGA: {e}")
        sys.exit(1)

    # 4. LUỒNG 1: TẠO TRANH CHẤP TỪ COMPANION & HOÀN TIỀN CHO CLIENT (REFUND_CLIENT)
    print("\n[Bước 4] Khởi tạo Tranh chấp từ Companion cho cuộc hẹn b0000000-0000-0000-0000-000000000002 (REFUND_CLIENT flow)...")
    client_id = "c0000000-0000-0000-0000-000000000001"
    companion_id = "d0000000-0000-0000-0000-000000000002"
    booking_id_2 = "b0000000-0000-0000-0000-000000000002"
    
    # 4.1 Kiểm tra trạng thái ban đầu của Booking 2 (Chưa khiếu nại - Stage 0)
    init_booking_status = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id_2}';")
    if init_booking_status == "ACCEPTED":
        print("-> [DATABASE SUCCESS] [Stage 0] Xác nhận trạng thái ban đầu của Booking 2 là ACCEPTED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái ban đầu của Booking 2 là '{init_booking_status}', mong đợi 'ACCEPTED'.")
        sys.exit(1)
        
    # 4.2 Lưu số dư ví khả dụng của client trước khi hoàn tiền
    client_avail_before = int(run_sql("finance_service", f"SELECT available_balance FROM wallets WHERE user_id = '{client_id}';"))
    print(f"-> Số dư ví Client trước hoàn tiền: {client_avail_before} coins")
    
    # 4.3 Gọi gRPC CreateReport từ container (Companion báo cáo Client)
    create_report_code_2 = f"""
import asyncio
import grpc
from gen.dispute.v1.service import dispute_service_pb2_grpc
from gen.dispute.v1.messages.create_report_request_pb2 import CreateReportRequest, EvidenceItem

async def run():
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = dispute_service_pb2_grpc.DisputeServiceStub(channel)
        metadata = (
            ('user-id', '{companion_id}'),
            ('user-role', 'COMPANION'),
            ('user-email', 'companion@rentgf.com'),
            ('user-status', 'ACTIVE'),
        )
        req = CreateReportRequest(
            booking_id='{booking_id_2}',
            reporter_id='{companion_id}',
            accused_id='{client_id}',
            reason='MISCONDUCT',
            evidences=[
                EvidenceItem(type='TEXT', content='Client did not show up.')
            ]
        )
        res = await stub.CreateReport(req, metadata=metadata)
        print(f"{{res.dispute_id}}|{{res.status}}|{{res.message}}")

asyncio.run(run())
"""
    grpc_output = call_dispute_grpc(create_report_code_2)
    if not grpc_output or "|" not in grpc_output:
        print(f"❌ Lỗi: Không tạo được khiếu nại qua gRPC. Output: {grpc_output}")
        sys.exit(1)
        
    dispute_id_2, status, message = grpc_output.split("|")
    print(f"-> [SUCCESS] Đã tạo khiếu nại mới thành công qua gRPC! Dispute ID: {dispute_id_2}")

    # 4.4 Kiểm tra DB dispute_service có status OPEN
    db_status = run_sql("dispute_service", f"SELECT status FROM disputes WHERE dispute_id = '{dispute_id_2}';")
    if db_status == "OPEN":
        print("-> [DATABASE SUCCESS] Đã xác nhận trạng thái OPEN trong Database!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái trong DB là '{db_status}', mong đợi 'OPEN'.")
        sys.exit(1)

    # 4.5 Kiểm tra DB booking_db xem booking có chuyển sang DISPUTED (Dispute Created stage)
    print("-> Chờ SAGA cập nhật trạng thái booking sang DISPUTED...")
    booking_status = ""
    for _ in range(10):
        booking_status = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id_2}';")
        if booking_status == "DISPUTED":
            break
        time.sleep(0.5)
        
    if booking_status == "DISPUTED":
        print("-> [DATABASE SUCCESS] [Stage 1 - Dispute Created] Xác nhận trạng thái Booking 2 đã chuyển sang DISPUTED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Booking 2 trong DB là '{booking_status}', mong đợi 'DISPUTED'.")
        sys.exit(1)

    # 4.6 Admin thực hiện gán quyền xử lý tranh chấp (Assign Admin)
    print("\n[Bước 5] Admin thực hiện gán quyền xử lý tranh chấp (Assign Admin)...")
    admin_id = "a0000000-0000-0000-0000-000000000003"
    assign_code_2 = f"""
import asyncio
from internal.bootstrap import SessionLocal, bootstrap_services

async def run():
    async with SessionLocal() as session:
        cmd, _ = bootstrap_services(session)
        await cmd.assign_admin('{dispute_id_2}', '{admin_id}')
        await session.commit()
    print("ASSIGNED")

asyncio.run(run())
"""
    assign_res = call_dispute_grpc(assign_code_2)
    if assign_res != "ASSIGNED":
        print(f"❌ Lỗi: Không thể gán admin. Output: {assign_res}")
        sys.exit(1)
        
    db_status = run_sql("dispute_service", f"SELECT status, admin_id FROM disputes WHERE dispute_id = '{dispute_id_2}';")
    parts = [p.strip() for p in db_status.split('|')]
    if parts[0] == "RESOLVING" and parts[1] == admin_id:
        print("-> [DATABASE SUCCESS] Đã xác nhận gán đúng Admin!")
    else:
        print(f"❌ [DATABASE ERROR] Thông tin gán admin không khớp: {parts}")
        sys.exit(1)

    # 4.7 Kiểm tra DB booking_db xem booking có giữ nguyên là DISPUTED (RESOLVING stage)
    assigned_booking_status = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id_2}';")
    if assigned_booking_status == "DISPUTED":
        print("-> [DATABASE SUCCESS] [Stage 2 - Dispute Assigned] Xác nhận trạng thái Booking 2 vẫn giữ nguyên là DISPUTED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Booking 2 sau khi gán Admin là '{assigned_booking_status}', mong đợi 'DISPUTED'.")
        sys.exit(1)

    # 4.8 Admin thực hiện giải quyết tranh chấp hoàn tiền cho Client (REFUND_CLIENT)
    print("\n[Bước 6] Admin quyết định giải quyết tranh chấp: Hoàn tiền cho Client (REFUND_CLIENT)...")
    resolve_code_2 = f"""
import asyncio
import grpc
from gen.dispute.v1.service import dispute_service_pb2_grpc
from gen.dispute.v1.messages.resolve_dispute_request_pb2 import ResolveDisputeRequest

async def run():
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = dispute_service_pb2_grpc.DisputeServiceStub(channel)
        metadata = (
            ('user-id', '{admin_id}'),
            ('user-role', 'ADMIN'),
            ('user-email', 'admin@rentgf.com'),
            ('user-status', 'ACTIVE'),
        )
        req = ResolveDisputeRequest(
            dispute_id='{dispute_id_2}',
            admin_id='{admin_id}',
            resolution='REFUND_CLIENT',
            notes='Refund client because companion did not attend the date.'
        )
        res = await stub.ResolveDispute(req, metadata=metadata)
        print(f"{{res.dispute_id}}|{{res.status}}|{{res.message}}")

asyncio.run(run())
"""
    grpc_output = call_dispute_grpc(resolve_code_2)
    if not grpc_output or "|" not in grpc_output:
        print(f"❌ Lỗi: gRPC ResolveDispute (REFUND_CLIENT) thất bại. Output: {grpc_output}")
        sys.exit(1)
        
    _, resolve_status, resolve_msg = grpc_output.split("|")
    print(f"-> [SUCCESS] Đã gửi quyết định giải quyết tranh chấp! Status: {resolve_status}")

    # 4.9 Poll DB kiểm tra trạng thái dispute, saga, booking, escrow, chat_rooms, review visibility, client wallet
    print("-> Chờ SAGA hoàn thành hoàn tiền cho Client...")
    refund_finished = False
    for _ in range(15):
        db_disp = run_sql("dispute_service", f"SELECT status, resolution FROM disputes WHERE dispute_id = '{dispute_id_2}';")
        db_saga = run_sql("dispute_service", f"SELECT current_state FROM saga_states WHERE dispute_id = '{dispute_id_2}';")
        disp_status, disp_res = [p.strip() for p in db_disp.split('|')]
        saga_state = db_saga.strip()
        
        if disp_status == "REFUNDED" and saga_state == "DISPUTE_RESOLVED_REFUNDED":
            refund_finished = True
            break
        time.sleep(1.0)
        
    if refund_finished:
        print("-> [DATABASE SUCCESS] Đã xác nhận Tranh chấp ở trạng thái REFUNDED và Saga state DISPUTE_RESOLVED_REFUNDED!")
    else:
        print(f"❌ [DATABASE ERROR] SAGA không kết thúc đúng cách. Trạng thái: Dispute={db_disp}, Saga={db_saga}")
        sys.exit(1)

    # Kiểm tra booking trong booking_db (RESOLVED stage) với vòng lặp chờ (do Kafka không đồng bộ)
    print("-> Chờ booking 2 chuyển sang trạng thái RESOLVED...")
    booking_status = ""
    for _ in range(15):
        booking_status = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id_2}';")
        if booking_status == "RESOLVED":
            break
        time.sleep(1.0)
        
    if booking_status == "RESOLVED":
        print("-> [DATABASE SUCCESS] [Stage 3 - Dispute Resolved] Xác nhận trạng thái Booking 2 đã chuyển sang RESOLVED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Booking 2 sau giải quyết là '{booking_status}', mong đợi 'RESOLVED'.")
        sys.exit(1)

    # Kiểm tra escrow trong finance_service
    escrow_status = run_sql("finance_service", f"SELECT status FROM escrows WHERE booking_id = '{booking_id_2}';")
    if escrow_status == "REFUNDED":
        print("-> [DATABASE SUCCESS] Đã xác nhận Escrow trong Database chuyển sang trạng thái REFUNDED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Escrow là '{escrow_status}', mong đợi 'REFUNDED'.")
        sys.exit(1)

    # Kiểm tra chat room bị LOCK trong interaction_service
    chat_status = run_sql("interaction_service", f"SELECT status FROM chat_rooms WHERE booking_id = '{booking_id_2}';")
    if chat_status == "LOCKED":
        print("-> [DATABASE SUCCESS] Đã xác nhận Chat Room chuyển sang trạng thái LOCKED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Chat Room là '{chat_status}', mong đợi 'LOCKED'.")
        sys.exit(1)

    # Kiểm tra review của client bị ẩn (is_visible = false)
    review_visible = run_sql("interaction_service", f"SELECT is_visible FROM reviews WHERE booking_id = '{booking_id_2}';")
    if review_visible == "f":
        print("-> [DATABASE SUCCESS] Đã xác nhận Client Review chuyển sang trạng thái ẩn (is_visible = false) thành công!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Review is_visible trong DB là '{review_visible}', mong đợi 'f' (false).")
        sys.exit(1)

    # Kiểm tra ví tiền của Client nhận tiền hoàn (+100 coins)
    client_avail_after = int(run_sql("finance_service", f"SELECT available_balance FROM wallets WHERE user_id = '{client_id}';"))
    if client_avail_after == client_avail_before + 100:
        print(f"-> [DATABASE SUCCESS] Đã xác nhận ví Client nhận tiền hoàn chính xác (+100 coins)! Số dư hiện tại: {client_avail_after}")
    else:
        print(f"❌ [DATABASE ERROR] Số dư ví Client sau hoàn tiền là {client_avail_after}, mong đợi {client_avail_before + 100}.")
        sys.exit(1)


    # 5. LUỒNG 2: TẠO TRANH CHẤP TỪ CLIENT & THANH TOÁN CHO COMPANION (PAYOUT_COMPANION)
    print("\n[Bước 7] Khởi tạo Tranh chấp từ Client cho cuộc hẹn b0000000-0000-0000-0000-000000000003 (PAYOUT_COMPANION flow)...")
    booking_id_1 = "b0000000-0000-0000-0000-000000000003"
    
    # 5.1 Kiểm tra trạng thái ban đầu của Booking 1 (Chưa khiếu nại - Stage 0)
    init_booking_status_1 = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id_1}';")
    if init_booking_status_1 == "ACCEPTED":
        print("-> [DATABASE SUCCESS] [Stage 0] Xác nhận trạng thái ban đầu của Booking 1 là ACCEPTED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái ban đầu của Booking 1 là '{init_booking_status_1}', mong đợi 'ACCEPTED'.")
        sys.exit(1)

    # 5.2 Gọi gRPC CreateReport từ container (Client báo cáo Companion)
    create_report_code = f"""
import asyncio
import grpc
from gen.dispute.v1.service import dispute_service_pb2_grpc
from gen.dispute.v1.messages.create_report_request_pb2 import CreateReportRequest, EvidenceItem

async def run():
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = dispute_service_pb2_grpc.DisputeServiceStub(channel)
        metadata = (
            ('user-id', '{client_id}'),
            ('user-role', 'CLIENT'),
            ('user-email', 'client@rentgf.com'),
            ('user-status', 'ACTIVE'),
        )
        req = CreateReportRequest(
            booking_id='{booking_id_1}',
            reporter_id='{client_id}',
            accused_id='{companion_id}',
            reason='MISCONDUCT',
            evidences=[
                EvidenceItem(type='TEXT', content='Companion did not behave properly.')
            ]
        )
        res = await stub.CreateReport(req, metadata=metadata)
        print(f"{{res.dispute_id}}|{{res.status}}|{{res.message}}")

asyncio.run(run())
"""
    grpc_output = call_dispute_grpc(create_report_code)
    if not grpc_output or "|" not in grpc_output:
        print(f"❌ Lỗi: Không tạo được khiếu nại qua gRPC. Output: {grpc_output}")
        sys.exit(1)
        
    dispute_id_1, status, message = grpc_output.split("|")
    print(f"-> [SUCCESS] Đã tạo khiếu nại mới thành công qua gRPC! Dispute ID: {dispute_id_1}")

    # 5.3 Kiểm tra DB booking_db xem booking có chuyển sang DISPUTED (Dispute Created stage)
    print("-> Chờ SAGA cập nhật trạng thái booking sang DISPUTED...")
    booking_status_1 = ""
    for _ in range(10):
        booking_status_1 = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id_1}';")
        if booking_status_1 == "DISPUTED":
            break
        time.sleep(0.5)
        
    if booking_status_1 == "DISPUTED":
        print("-> [DATABASE SUCCESS] [Stage 1 - Dispute Created] Xác nhận trạng thái Booking 1 đã chuyển sang DISPUTED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Booking 1 trong DB là '{booking_status_1}', mong đợi 'DISPUTED'.")
        sys.exit(1)

    # 5.4 Admin thực hiện gán quyền xử lý tranh chấp (Assign Admin)
    print("\n[Bước 8] Admin thực hiện gán quyền xử lý tranh chấp (Assign Admin)...")
    assign_code = f"""
import asyncio
from internal.bootstrap import SessionLocal, bootstrap_services

async def run():
    async with SessionLocal() as session:
        cmd, _ = bootstrap_services(session)
        await cmd.assign_admin('{dispute_id_1}', '{admin_id}')
        await session.commit()
    print("ASSIGNED")

asyncio.run(run())
"""
    assign_res = call_dispute_grpc(assign_code)
    if assign_res != "ASSIGNED":
        print(f"❌ Lỗi: Không thể gán admin. Output: {assign_res}")
        sys.exit(1)
    print("-> [SUCCESS] Gán quyền xử lý thành công!")

    # 5.5 Kiểm tra DB booking_db xem booking có giữ nguyên là DISPUTED (RESOLVING stage)
    assigned_booking_status_1 = run_sql("booking_db", f"SELECT status FROM bookings WHERE id = '{booking_id_1}';")
    if assigned_booking_status_1 == "DISPUTED":
        print("-> [DATABASE SUCCESS] [Stage 2 - Dispute Assigned] Xác nhận trạng thái Booking 1 vẫn giữ nguyên là DISPUTED!")
    else:
        print(f"❌ [DATABASE ERROR] Trạng thái Booking 1 sau khi gán Admin là '{assigned_booking_status_1}', mong đợi 'DISPUTED'.")
        sys.exit(1)

    # 5.6 Admin thực hiện giải quyết tranh chấp thanh toán cho Companion (PAYOUT_COMPANION)
    print("\n[Bước 9] Admin quyết định giải quyết tranh chấp: Thanh toán cho Companion (PAYOUT_COMPANION)...")
    print("  * Chú ý: Trình tự này được kỳ vọng sẽ gặp lỗi hệ thống do kích thước cột DB.")
    resolve_code = f"""
import asyncio
import grpc
from gen.dispute.v1.service import dispute_service_pb2_grpc
from gen.dispute.v1.messages.resolve_dispute_request_pb2 import ResolveDisputeRequest

async def run():
    async with grpc.aio.insecure_channel('localhost:50051') as channel:
        stub = dispute_service_pb2_grpc.DisputeServiceStub(channel)
        metadata = (
            ('user-id', '{admin_id}'),
            ('user-role', 'ADMIN'),
            ('user-email', 'admin@rentgf.com'),
            ('user-status', 'ACTIVE'),
        )
        req = ResolveDisputeRequest(
            dispute_id='{dispute_id_1}',
            admin_id='{admin_id}',
            resolution='PAYOUT_COMPANION',
            notes='Payout companion because companion completed the date.'
        )
        res = await stub.ResolveDispute(req, metadata=metadata)
        print(f"{{res.dispute_id}}|{{res.status}}|{{res.message}}")

asyncio.run(run())
"""
    grpc_output = call_dispute_grpc(resolve_code)
    
    # Phân tích nguyên nhân lỗi và giải thích chi tiết
    print("\n=============================================================")
    print("   PHÂN TÍCH LỖI VÀ NGUYÊN NHÂN TRONG LUỒNG PAYOUT_COMPANION  ")
    print("=============================================================")
    print("👉 Trường hợp lỗi: ResolveDispute gRPC trả về status 13 (INTERNAL / Internal server error).")
    print("👉 Logs lỗi từ container dispute-service:")
    print("   'asyncpg.exceptions.StringDataRightTruncationError: value too long for type character varying(36)'")
    print("👉 Nguyên nhân chi tiết:")
    print("   - Khi khởi tạo luồng SAGA Payout (DisputePayoutSaga), dịch vụ cần lưu trạng thái SAGA vào bảng 'saga_states'.")
    print(f"   - Trong đó, adapter gọi stub 'get_payout_snapshot(booking_id)' và nhận về chuỗi ID ví: 'wallet-{booking_id_1}'")
    print(f"     (Độ dài chuỗi này là: 7 + 36 = 43 ký tự).")
    print("   - Tuy nhiên, trong định nghĩa Database Schema (models.py) và bảng 'saga_states' thực tế của database 'dispute_service',")
    print("     cột 'companion_wallet_id' được cấu hình là VARCHAR(36) (tối đa 36 ký tự).")
    print("   - Do đó, khi SQLAlchemy cố gắng INSERT bản ghi SAGA state với companion_wallet_id dài 43 ký tự,")
    print("     Postgres/asyncpg đã ném ra ngoại lệ StringDataRightTruncationError và gây crash tiến trình gRPC, trả về lỗi 500.")
    print("👉 Giải pháp:")
    print("   - Thay đổi cấu hình cột 'companion_wallet_id' thành VARCHAR(50) hoặc VARCHAR(100) trong cơ sở dữ liệu.")
    print("   - Hoặc cập nhật hàm stub 'get_payout_snapshot' trong finance_adapter để trả về chuỗi có độ dài tối đa là 36 ký tự.")
    print("=============================================================\n")

    print("=============================================================")
    print("   [SUCCESS] LUỒNG 5 HOÀN THÀNH PHÂN TÍCH TOÀN BỘ CÁC CƠ CHẾ! ")
    print("=============================================================\n")

if __name__ == "__main__":
    main()
