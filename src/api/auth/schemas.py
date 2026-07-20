from pydantic import BaseModel, ConfigDict

from services.sessions import ACCESS_TOKEN_LIFETIME, REFRESH_IDLE_LIFETIME


class LoginRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    username: str
    password: str


class RefreshRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    refresh_token: str


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "Bearer"
    expires_in: int = int(ACCESS_TOKEN_LIFETIME.total_seconds())
    refresh_expires_in: int = int(REFRESH_IDLE_LIFETIME.total_seconds())
