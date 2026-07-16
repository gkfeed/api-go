from fastapi import Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import PlainTextResponse


async def validation_error_handler(
    _request: Request, error: RequestValidationError
) -> PlainTextResponse:
    first_error = error.errors()[0]
    location = first_error.get("loc", ())
    if location and location[0] == "query":
        message = f"Invalid {location[-1]}"
    else:
        message = "Invalid request body"
    return PlainTextResponse(message, status_code=400)
