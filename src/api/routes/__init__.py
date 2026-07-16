from fastapi import APIRouter

from .feeds import router as feeds_router
from .items import router as items_router

api_router = APIRouter(prefix="/api/v1")
api_router.include_router(feeds_router)
api_router.include_router(items_router)

__all__ = ["api_router"]
