"""
conftest.py — Root test configuration for Finance Service.

Sets TESTING=1 in os.environ BEFORE any internal module is imported.
This ensures:
1. internal.bootstrap uses sqlite+aiosqlite:///:memory: (not Postgres)
2. bootstrap skips asyncio.run(init_db()) at module load time
3. The circular import bootstrap <-> router is resolved because bootstrap
   is always loaded from this conftest before any test file touches it.
"""

import os

os.environ["TESTING"] = "1"
