import httpx
from typing import Optional


class ProxyMeshPeer:
    def __init__(self, base_url: str = "http://localhost:8000/api/peer", token: Optional[str] = None, node_id: Optional[str] = None):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.node_id = node_id

    def set_auth(self, token: str, node_id: str) -> None:
        self.token = token
        self.node_id = node_id

    def clear_auth(self) -> None:
        self.token = None
        self.node_id = None

    async def _request(self, path: str, method: str = "GET", json_data: Optional[dict] = None) -> dict:
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["X-Peer-Token"] = self.token

        async with httpx.AsyncClient() as client:
            res = await client.request(
                method,
                f"{self.base_url}{path}",
                json=json_data,
                headers=headers,
            )
            data = res.json()
            if res.status_code >= 400:
                raise RuntimeError(data.get("error", f"HTTP {res.status_code}"))
            return data

    async def login(self, node_id: str) -> dict:
        data = await self._request("/auth", "POST", {"node_id": node_id})
        self.set_auth(data["token"], data["node_id"])
        return data

    async def disconnect(self) -> None:
        await self._request("/disconnect", "POST", {})
        self.clear_auth()

    async def get_status(self) -> dict:
        return await self._request("/status")

    async def get_bandwidth(self) -> dict:
        return await self._request("/bandwidth")

    async def get_earnings(self) -> dict:
        return await self._request("/earnings")

    async def get_health(self) -> dict:
        data = await self._request("/health")
        return data.get("score", {})

    async def set_consent(self, enabled: bool) -> dict:
        return await self._request("/consent", "POST", {"enabled": enabled})

    async def update_node(self, updates: dict) -> dict:
        return await self._request("/node", "PATCH", updates)
