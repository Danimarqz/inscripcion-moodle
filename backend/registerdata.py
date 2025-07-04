from pydantic import BaseModel, EmailStr

class RegisterData(BaseModel):
    dob: str
    dni: str
    address: str
    city: str
    locality: str
    postalcode: str
    country: str
    phone: str
    email: EmailStr
    course: str
    modality: str
    payment: str
    iban: str
    startdate: str
    signature: str
    requestdate: str
    name: str
    surname: str
    discover: str