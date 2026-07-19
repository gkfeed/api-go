from fastapi import APIRouter

from ..auth import router as auth_router
from .feeds import router as feeds_router
from .items import router as items_router

api_router = APIRouter(prefix="/api/v2")
api_router.include_router(auth_router)
api_router.include_router(feeds_router)
api_router.include_router(items_router)

__all__ = ["api_router"]
