from pydantic import BaseModel, Field


class User(BaseModel):
    id: int
    name: str
    hashed_password: str = Field(exclude=True)
