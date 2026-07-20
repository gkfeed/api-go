from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware

from config import Settings, load_settings
from database import Database
from services.sessions import SessionStore

from .errors import validation_error_handler
from .routes import api_router


def create_app(settings: Settings | None = None) -> FastAPI:
    configuration = settings or load_settings()
    app = FastAPI(title="GKFeed API", version="2.0.0")
    app.state.database = Database(configuration.database_path)
    app.state.sessions = SessionStore()
    app.add_middleware(
        CORSMiddleware,
        allow_origins=list(configuration.allowed_origins),
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
    app.add_exception_handler(RequestValidationError, validation_error_handler)
    app.include_router(api_router)
    return app
