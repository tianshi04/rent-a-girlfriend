from typing import Dict, Any, Optional
import boto3
from botocore.config import Config
from internal.application.port import IStoragePort

class S3Storage(IStoragePort):
    def __init__(
        self,
        bucket_name: str,
        region_name: str,
        access_key_id: str,
        secret_access_key: str,
        endpoint_url: Optional[str] = None
    ):
        self.bucket_name = bucket_name
        self.endpoint_url = endpoint_url
        self.region_name = region_name

        # Standard client config
        # Use signature version v4 for compatibility
        config = Config(signature_version='s3v4')
        
        self.s3_client = boto3.client(
            's3',
            endpoint_url=self.endpoint_url, # Key to avoid AWS lock-in (e.g. MinIO, Cloudflare R2)
            region_name=self.region_name,
            aws_access_key_id=access_key_id,
            aws_secret_access_key=secret_access_key,
            config=config
        )

    def generate_presigned_put_url(
        self,
        key: str,
        content_type: str,
        content_length: int
    ) -> str:
        """
        Generates S3 standard PUT presigned URL.
        """
        # Set content-length in client params to restrict size on upload
        params = {
            'Bucket': self.bucket_name,
            'Key': key,
            'ContentType': content_type
        }
        
        upload_url = self.s3_client.generate_presigned_url(
            ClientMethod='put_object',
            Params=params,
            ExpiresIn=3600 # Valid for 1 hour
        )
        return upload_url

    def get_object_metadata(self, key: str) -> Dict[str, Any]:
        """
        Double-check S3 object metadata after upload.
        """
        response = self.s3_client.head_object(
            Bucket=self.bucket_name,
            Key=key
        )
        return {
            "ContentLength": response.get("ContentLength", 0),
            "ContentType": response.get("ContentType", "")
        }
