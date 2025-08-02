from dotenv import load_dotenv
import os

load_dotenv()

DATABASE_URL = os.getenv("DATABASE_URL")
SMTP_USER = os.getenv("SMTP_USER")
SMTP_PASS = os.getenv("SMTP_PASS")
SMTP_SERVER = os.getenv("SMTP_SERVER")
SMTP_PORT = os.getenv("SMTP_PORT")
MOODLE_TOKEN = os.getenv("MOODLE_TOKEN")
MOODLE_URL = os.getenv("MOODLE_URL")
ADMIN_EMAIL= os.getenv("ADMIN_EMAIL")