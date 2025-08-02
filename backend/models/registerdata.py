from pydantic import BaseModel, EmailStr

class RegisterData(BaseModel):
    address: str
    city: str
    country: str
    course: str
    discover: str
    dni: str
    dob: str
    email: EmailStr
    iban: str
    locality: str
    modality: str
    name: str
    payment: str
    phone: str
    postalcode: str
    signature: str
    startdate: str
    surname: str
    website: str