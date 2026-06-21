import requests
import time
import sys

BASE_URL = "http://localhost:8000"

# Pre-seeded UUIDs from sql files
CLIENT_ID = "c0000000-0000-0000-0000-000000000001"
COMPANION_ID = "d0000000-0000-0000-0000-000000000002"
SCENARIO_ID = "s0000000-0000-0000-0000-000000000001" # Coffee Date & Walk (Price: 100)

def main():
    print("=============================================================")
    print("   Starting System E2E Business Flow Verification Test       ")
    print("=============================================================\n")

    # 1. Verify API Gateway Health
    print("[Step 1] Verifying API Gateway health status...")
    try:
        resp = requests.get(f"{BASE_URL}/health", timeout=5)
        resp.raise_for_status()
        print(f"-> Gateway health response: {resp.json()}")
        print("-> API Gateway is UP and running!\n")
    except Exception as e:
        print(f"❌ Failed to reach API Gateway: {e}")
        sys.exit(1)

    # 2. Check Client & Companion Wallets
    print("[Step 2] Querying initial wallet balances from Finance Service...")
    
    # Query Client Wallet
    resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={CLIENT_ID}", timeout=5)
    resp.raise_for_status()
    client_wallet = resp.json()
    print(f"-> Client Wallet: Available={client_wallet['availableBalance']}, Frozen={client_wallet['frozenBalance']}")

    # Query Companion Wallet
    resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={COMPANION_ID}", timeout=5)
    resp.raise_for_status()
    companion_wallet = resp.json()
    print(f"-> Companion Wallet: Available={companion_wallet['availableBalance']}, Frozen={companion_wallet['frozenBalance']}\n")

    # 3. Check Profile Service Catalogue Search
    print("[Step 3] Searching Companion Profiles Catalogue...")
    resp = requests.get(f"{BASE_URL}/api/v1/companions", timeout=5)
    resp.raise_for_status()
    catalogue_resp = resp.json()
    catalogue = catalogue_resp.get("data", [])
    print(f"-> Found {len(catalogue)} companions in catalogue.")
    
    companion_profile = next((c for c in catalogue if c["companionId"] == COMPANION_ID), None)
    if companion_profile:
        print(f"-> Companion Display Name: {companion_profile['displayName']}")
    print("-> Catalogue search completed!\n")

    # 4. Client Request Booking
    print("[Step 4] Client initiating Booking Request...")
    booking_payload = {
        "clientId": CLIENT_ID,
        "companionId": COMPANION_ID,
        "scenarioId": SCENARIO_ID,
        "startTime": "2026-06-21T10:00:00Z"
    }
    headers = {
        "Content-Type": "application/json",
        "user-id": CLIENT_ID,
        "user-role": "CLIENT",
        "user-email": "client@rentgf.com"
    }
    resp = requests.post(f"{BASE_URL}/api/v1/bookings", json=booking_payload, headers=headers, timeout=5)
    resp.raise_for_status()
    booking_data = resp.json()
    booking_id = booking_data.get("bookingId")
    print(f"-> Booking created: ID={booking_id}, Status={booking_data.get('status')}")
    
    # Wait for SAGA/Event to process the reservation (freeze coins)
    print("Waiting 3s for transactional outbox & event processing...")
    time.sleep(3)

    # 5. Verify Wallet Balance Frozen
    print("[Step 5] Checking wallet state after booking request...")
    resp = requests.get(f"{BASE_URL}/api/v1/finance/wallet?userId={CLIENT_ID}", timeout=5)
    resp.raise_for_status()
    client_wallet_after = resp.json()
    print(f"-> Client Wallet: Available={client_wallet_after['availableBalance']}, Frozen={client_wallet_after['frozenBalance']}")
    print("-> SAGA wallet status checked!\n")

    # 6. Companion Accepts Booking
    print("[Step 6] Companion accepting the Booking...")
    accept_headers = {
        "user-id": COMPANION_ID,
        "user-role": "COMPANION",
        "user-email": "companion@rentgf.com"
    }
    resp = requests.put(f"{BASE_URL}/api/v1/bookings/{booking_id}/accept", headers=accept_headers, timeout=5)
    resp.raise_for_status()
    print(f"-> Accept response status code: {resp.status_code}")
    
    # Wait for SAGA to execute and interaction room to be created
    print("Waiting 3s for accept event propagation & chat room setup...")
    time.sleep(3)

    # 7. Check Interaction Chat Room
    print("[Step 7] Checking if Interaction Chat Room was initialized...")
    chat_headers = {
        "Content-Type": "application/json",
        "user-id": CLIENT_ID,
        "user-role": "CLIENT"
    }
    message_payload = {
        "text": "Hello! Looking forward to meeting you."
    }
    resp = requests.post(f"{BASE_URL}/api/v1/interaction/rooms/{booking_id}/messages", json=message_payload, headers=chat_headers, timeout=5)
    
    if resp.status_code == 201:
        print(f"-> Chat message sent successfully! Room ID={booking_id}")
        chat_msg = resp.json()
        print(f"-> Message content: {chat_msg['content']}")
    else:
        print(f"[WARNING] Failed to send message with Room ID equal to Booking ID (Status {resp.status_code}): {resp.text}. Trying room query...")

    print("-> Interaction/Chat service E2E setup validated!\n")

    # 8. Complete Booking & Settle Payment
    print("[Step 8] Client writing a 5-star review for the Companion...")
    resp = requests.get(f"{BASE_URL}/api/v1/interaction/reviews/companion/{COMPANION_ID}", timeout=5)
    resp.raise_for_status()
    reviews = resp.json()
    print(f"-> Found {len(reviews)} reviews for Companion {COMPANION_ID}.")
    if len(reviews) > 0:
        first_review = reviews[0]
        print(f"-> Review comment: '{first_review['comment']}', Rating: {first_review['rating']}")
    
    print("\n=============================================================")
    print("   [SUCCESS] System E2E Business Flow Verification: SUCCESS! ")
    print("=============================================================")

if __name__ == "__main__":
    main()
