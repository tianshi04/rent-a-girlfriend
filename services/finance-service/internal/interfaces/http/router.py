from fastapi import APIRouter, Depends, HTTPException, Request, Query, status
from pydantic import BaseModel, Field, ConfigDict
from pydantic.alias_generators import to_camel

from internal.bootstrap import get_finance_cmd
from internal.application.command.finance import FinanceCommandService

router = APIRouter(prefix="/api/v1/finance", tags=["Finance"])


class TopupRequest(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    user_id: str = Field(..., description="The ID of the user topping up")
    amount: int = Field(
        ..., gt=0, description="Amount of Kano-Coins to buy (1 Coin = 1,000 VND)"
    )


class TopupResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    payment_url: str = Field(..., description="VNPay Sandbox checkout redirect URL")


class WalletResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    wallet_id: str
    user_id: str
    available_balance: int
    frozen_balance: int


@router.post(
    "/topup", response_model=TopupResponse, status_code=status.HTTP_201_CREATED
)
async def initiate_topup(
    request: Request,
    payload: TopupRequest,
    finance_cmd: FinanceCommandService = Depends(get_finance_cmd),
):
    """
    Initiates a wallet top-up by creating a pending transaction and returning a signed VNPay URL.
    """
    client_ip = request.client.host if request.client else "127.0.0.1"
    try:
        payment_url = await finance_cmd.initiate_topup(
            user_id=payload.user_id,
            amount_coins=payload.amount,
            client_ip=client_ip,
        )
        return TopupResponse(payment_url=payment_url)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to initiate top-up: {str(e)}",
        )


@router.get("/vnpay-ipn")
async def vnpay_ipn(
    request: Request,
    finance_cmd: FinanceCommandService = Depends(get_finance_cmd),
):
    """
    VNPay IPN Webhook callback. Handles secure verification, credits wallet, and commits ledger.
    Ensures absolute transaction Idempotency.
    """
    params = dict(request.query_params)
    try:
        response = await finance_cmd.process_vnpay_ipn(params)
        return response
    except Exception as e:
        # According to VNPay standard, even if exception occurs, we should return error format
        return {"RspCode": "99", "Message": f"Internal system error: {str(e)}"}


@router.get("/vnpay-return")
async def vnpay_return(
    request: Request,
    finance_cmd: FinanceCommandService = Depends(get_finance_cmd),
):
    """
    User redirected here after VNPay payment process.
    Verifies signature and returns simple visually premium success/fail HTML.
    """
    params = dict(request.query_params)
    is_valid_sig = finance_cmd.vnpay_adapter.validate_ipn_signature(params)

    response_code = params.get("vnp_ResponseCode")
    txn_ref = params.get("vnp_TxnRef", "Unknown")
    amount_vnd = int(params.get("vnp_Amount", 0)) // 100

    if not is_valid_sig:
        status_text = "FAILED (Invalid Signature)"
        message = (
            "Chữ ký bảo mật không hợp lệ. Vui lòng không thay đổi tham số giao dịch."
        )
        color = "#e53e3e"
    elif response_code == "00":
        status_text = "SUCCESS"
        message = "Giao dịch nạp tiền thành công! Tài khoản của bạn sẽ được cộng Kano-Coin tương ứng sớm nhất."
        color = "#319795"
    else:
        status_text = f"FAILED (Code {response_code})"
        message = "Giao dịch không thành công hoặc đã bị hủy từ phía người dùng."
        color = "#e53e3e"

    from fastapi.responses import HTMLResponse

    html_content = f"""
    <!DOCTYPE html>
    <html lang="vi">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Kết quả giao dịch | Kano-Coin</title>
        <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600;700&display=swap" rel="stylesheet">
        <style>
            body {{
                font-family: 'Outfit', sans-serif;
                background: linear-gradient(135deg, #0f172a 0%, #1e1b4b 100%);
                color: #f8fafc;
                display: flex;
                align-items: center;
                justify-content: center;
                min-height: 100vh;
                margin: 0;
            }}
            .card {{
                background: rgba(30, 41, 59, 0.7);
                backdrop-filter: blur(16px);
                border: 1px solid rgba(255, 255, 255, 0.1);
                border-radius: 24px;
                padding: 40px;
                max-width: 480px;
                width: 100%;
                box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3), 0 10px 10px -5px rgba(0, 0, 0, 0.3);
                text-align: center;
                animation: slideUp 0.6s ease-out;
            }}
            @keyframes slideUp {{
                from {{ opacity: 0; transform: translateY(20px); }}
                to {{ opacity: 1; transform: translateY(0); }}
            }}
            .icon {{
                font-size: 64px;
                margin-bottom: 20px;
            }}
            h2 {{
                margin: 0 0 10px 0;
                font-weight: 700;
                font-size: 24px;
                letter-spacing: -0.5px;
            }}
            .status {{
                display: inline-block;
                padding: 8px 16px;
                border-radius: 9999px;
                font-weight: 600;
                font-size: 14px;
                letter-spacing: 0.5px;
                background-color: {color};
                margin-bottom: 20px;
            }}
            .details {{
                background: rgba(15, 23, 42, 0.4);
                border-radius: 16px;
                padding: 20px;
                margin-bottom: 24px;
                text-align: left;
            }}
            .row {{
                display: flex;
                justify-content: space-between;
                margin-bottom: 10px;
                font-size: 14px;
            }}
            .row:last-child {{
                margin-bottom: 0;
            }}
            .label {{
                color: #94a3b8;
            }}
            .val {{
                font-weight: 600;
                color: #f1f5f9;
            }}
            .desc {{
                font-size: 14px;
                color: #cbd5e1;
                line-height: 1.6;
                margin-bottom: 30px;
            }}
            .btn {{
                display: inline-block;
                background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
                color: white;
                text-decoration: none;
                padding: 12px 32px;
                border-radius: 9999px;
                font-weight: 600;
                transition: transform 0.2s, box-shadow 0.2s;
                box-shadow: 0 4px 14px 0 rgba(99, 102, 241, 0.4);
            }}
            .btn:hover {{
                transform: translateY(-2px);
                box-shadow: 0 6px 20px 0 rgba(99, 102, 241, 0.6);
            }}
        </style>
    </head>
    <body>
        <div class="card">
            <div class="icon">{"✨" if response_code == "00" else "⚠️"}</div>
            <h2>Kết Quả Thanh Toán</h2>
            <div class="status">{status_text}</div>
            
            <div class="details">
                <div class="row">
                    <span class="label">Mã Giao Dịch:</span>
                    <span class="val">{txn_ref}</span>
                </div>
                <div class="row">
                    <span class="label">Số Tiền (VND):</span>
                    <span class="val">{amount_vnd:,.0f} đ</span>
                </div>
            </div>
            
            <p class="desc">{message}</p>
            
            <a href="#" class="btn" onclick="window.close(); return false;">Đóng Cửa Sổ</a>
        </div>
    </body>
    </html>
    """
    return HTMLResponse(content=html_content, status_code=200)


@router.get("/wallet", response_model=WalletResponse)
async def get_wallet(
    user_id: str = Query(
        ..., alias="userId", description="The ID of the user whose wallet is requested"
    ),
    finance_cmd: FinanceCommandService = Depends(get_finance_cmd),
):
    """
    Query current wallet balances. Uses lazy-initialization fallback if the wallet does not exist yet.
    """
    try:
        wallet = await finance_cmd.get_or_create_wallet(user_id)
        return WalletResponse(
            wallet_id=wallet.wallet_id,
            user_id=wallet.user_id,
            available_balance=wallet.available_balance.amount,
            frozen_balance=wallet.frozen_balance.amount,
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to query wallet: {str(e)}",
        )
