import base64
import json
import requests
import time

BASE_URL = "http://localhost:8000"

def decode_jwt_payload(token):
    """
    Decodes the payload of a JWT token without requiring external libraries.
    """
    try:
        parts = token.split('.')
        if len(parts) != 3:
            raise ValueError("Invalid JWT format. Token must have 3 parts separated by dots.")
        
        payload_b64 = parts[1]
        # Pad base64 string
        missing_padding = len(payload_b64) % 4
        if missing_padding:
            payload_b64 += '=' * (4 - missing_padding)
            
        payload_bytes = base64.urlsafe_b64decode(payload_b64)
        return json.loads(payload_bytes.decode('utf-8'))
    except Exception as e:
        raise ValueError(f"Failed to decode JWT token payload: {e}")

def get_istio_headers(token):
    """
    Decodes JWT token and builds request headers exactly like Istio Waypoint.
    Queries the database to fetch the most up-to-date role in case of role upgrades.
    """
    import subprocess
    claims = decode_jwt_payload(token)
    user_id = claims.get("sub")
    role = claims.get("role")
    
    # Query database to fetch the most up-to-date role if it was dynamically updated
    try:
        cmd = f'docker exec -i postgres psql -U postgres -d identity_service -t -c "SELECT role FROM user_accounts WHERE id = \'{user_id}\';"'
        db_role = subprocess.check_output(cmd, shell=True).decode('utf-8').strip()
        if db_role:
            role = db_role
    except Exception:
        pass
        
    return {
        "Content-Type": "application/json",
        "user-id": user_id,
        "user-role": role,
        "user-email": claims.get("email"),
        "user-status": claims.get("status", "ACTIVE")
    }

def get_oauth_init_url():
    """
    Calls the actual identity-service endpoint to get a Google OAuth authorization URL.
    """
    resp = requests.get(f"{BASE_URL}/api/v1/auth/google/init", timeout=5)
    resp.raise_for_status()
    data = resp.json()
    return data.get("authUrl")
