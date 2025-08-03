from pydantic import BaseModel

class AdminLogin(BaseModel):
    username: str
    password: str

class AdminCreate(AdminLogin):
    pass

class TokenResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
