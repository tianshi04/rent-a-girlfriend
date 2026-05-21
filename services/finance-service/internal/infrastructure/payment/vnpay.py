import hashlib
import hmac
import urllib.parse
from datetime import datetime


class VNPayAdapter:
    def __init__(
        self, tmn_code: str, hash_secret: str, payment_url: str, return_url: str
    ):
        self.tmn_code = tmn_code
        self.hash_secret = hash_secret
        self.payment_url = payment_url
        self.return_url = return_url

    def generate_payment_url(
        self, txn_ref: str, amount_vnd: int, ip_address: str
    ) -> str:
        """
        Generates a signed VNPay payment URL for local or production sandbox.
        """
        create_date = datetime.now().strftime("%Y%m%d%H%M%S")

        # VNPay Standard Parameters (V2.1.0)
        vnp_params = {
            "vnp_Version": "2.1.0",
            "vnp_Command": "pay",
            "vnp_TmnCode": self.tmn_code,
            "vnp_Amount": str(amount_vnd * 100),  # VNPay requires multiplying by 100
            "vnp_CurrCode": "VND",
            "vnp_TxnRef": txn_ref,
            "vnp_OrderInfo": f"Topup Kano-Coin for transaction {txn_ref}",
            "vnp_OrderType": "other",
            "vnp_Locale": "vn",
            "vnp_ReturnUrl": self.return_url,
            "vnp_IpAddr": ip_address,
            "vnp_CreateDate": create_date,
        }

        # Sort parameters alphabetically
        sorted_params = sorted(vnp_params.items())

        # Build query string
        query_string = urllib.parse.urlencode(sorted_params)

        # Calculate SecureHash using HMAC-SHA512
        secure_hash = hmac.new(
            self.hash_secret.encode("utf-8"),
            query_string.encode("utf-8"),
            hashlib.sha512,
        ).hexdigest()

        # Return final signed URL
        return f"{self.payment_url}?{query_string}&vnp_SecureHash={secure_hash}"

    def validate_ipn_signature(self, vnp_params: dict) -> bool:
        """
        Validates HMAC-SHA512 signature from VNPay IPN callback.
        """
        # Extract received hash
        received_hash = vnp_params.get("vnp_SecureHash")
        if not received_hash:
            return False

        # Remove hash fields before sorting
        clean_params = {
            k: v
            for k, v in vnp_params.items()
            if k not in ["vnp_SecureHash", "vnp_SecureHashType"]
        }

        # Sort and encode parameters
        sorted_params = sorted(clean_params.items())

        # Build raw query string
        # Note: VNPay requires using urllib.parse.quote_plus for values to exactly match signature input
        # standard urlencode works in most sandbox implementations, but sorting is crucial
        query_string = urllib.parse.urlencode(sorted_params)

        # Calculate local hash
        calculated_hash = hmac.new(
            self.hash_secret.encode("utf-8"),
            query_string.encode("utf-8"),
            hashlib.sha512,
        ).hexdigest()

        return calculated_hash.lower() == received_hash.lower()
