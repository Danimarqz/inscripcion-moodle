from fpdf import FPDF
from datetime import datetime
import base64
from io import BytesIO
from PIL import Image
import tempfile
import os
from registerdata import RegisterData

MESES = {
    'JANUARY': 'ENERO', 'FEBRUARY': 'FEBRERO', 'MARCH': 'MARZO', 'APRIL': 'ABRIL',
    'MAY': 'MAYO', 'JUNE': 'JUNIO', 'JULY': 'JULIO', 'AUGUST': 'AGOSTO',
    'SEPTEMBER': 'SEPTIEMBRE', 'OCTOBER': 'OCTUBRE', 'NOVEMBER': 'NOVIEMBRE', 'DECEMBER': 'DICIEMBRE'
}

class PDF(FPDF):
    def header(self):
        if os.path.exists("assets/opositalogo.png"):
            self.image("assets/opositalogo.png", x=10, y=8, w=10)
        self.set_font('Arial', 'B', 14)
        self.set_text_color(0, 51, 102)
        self.cell(0, 10, 'FORMULARIO DE SUSCRIPCIÓN', ln=True, align='C')
        self.set_draw_color(0, 51, 102)
        self.set_line_width(0.5)
        self.line(10, 22, 200, 22)
        self.ln(5)

    def footer(self):
        self.set_y(-15)
        self.set_font('Arial', 'I', 8)
        self.set_text_color(108)
        self.cell(0, 10, f'Página {self.page_no()}', 0, 0, 'C')

def generate_pdf(data: RegisterData) -> str:
    pdf = PDF()
    pdf.add_page()
    pdf.set_auto_page_break(auto=True, margin=15)
    pdf.add_font('DejaVu', '', 'assets/DejaVuSans.ttf', uni=True)
    pdf.add_font('DejaVu', 'B', 'assets/DejaVuSans-Bold.ttf', uni=True)
    pdf.set_font('DejaVu', size=10)
    pdf.set_text_color(0)

    def add_section_title(title):
        pdf.ln(5)
        pdf.set_font("DejaVu", style='B', size=10)
        pdf.set_fill_color(230, 230, 250)
        pdf.cell(0, 10, title, ln=True, fill=True)
        pdf.set_font("DejaVu", size=10)

    def labeled_line(label, value, width_label=50, width_value=140):
        pdf.set_font("DejaVu", style='B', size=10)
        pdf.cell(width_label, 10, label, border=0)
        pdf.set_font("DejaVu", style='', size=10)
        pdf.cell(width_value, 10, value, ln=True)

    # Fecha de solicitud
    labeled_line("Fecha de solicitud:", datetime.strptime(data["requestdate"], "%Y-%m-%d").strftime("%d-%m-%Y"))

    # Datos personales
    add_section_title("Datos personales")
    labeled_line("Nombre:", data["name"])
    labeled_line("Apellido:", data["surname"])

    # Fecha de nacimiento + DNI
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(55, 10, "Fecha de nacimiento:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(35, 10, datetime.strptime(data["dob"], "%Y-%m-%d").strftime("%d-%m-%Y"), border=0)
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(20, 10, "DNI:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(70, 10, data["dni"], ln=True)

    # Dirección (línea única)
    labeled_line("Dirección:", data["address"])

    # Ciudad + Localidad
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(40, 10, "Ciudad:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(50, 10, data["city"], border=0)
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(30, 10, "Localidad:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(70, 10, data["locality"], ln=True)

    # Código postal + País
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(40, 10, "Código postal:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(50, 10, data["postalcode"], border=0)
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(20, 10, "País:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(70, 10, data["country"], ln=True)

    # Teléfono + Email
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(40, 10, "Teléfono:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(50, 10, data["phone"], border=0)
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(20, 10, "Email:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(70, 10, data["email"], ln=True)



    # Datos académicos
    add_section_title("Datos académicos")
    labeled_line("Curso:", data["course"])

    # Modalidad + Forma de pago
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(40, 10, "Modalidad:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(50, 10, data["modality"], border=0)
    pdf.set_font("DejaVu", style='B', size=10)
    pdf.cell(40, 10, "Forma de pago:", border=0)
    pdf.set_font("DejaVu", style='', size=10)
    pdf.cell(10, 10, data["payment"], ln=True)

    labeled_line("IBAN:", data["iban"])
    # Obtener sólo el mes en mayúsculas de la fecha de inicio

    try:
        mes_en = datetime.strptime(data["startdate"], "%Y-%m").strftime("%B").upper()
        mes_inicio = MESES.get(mes_en, mes_en)
    except Exception:
        mes_inicio = data["startdate"]  # fallback si el formato no es válido

    labeled_line("Mes de inicio:", mes_inicio)

    pdf.set_font("Arial", style='I', size=10)
    pdf.multi_cell(0, 7, "Los cursos que se pagan mensualmente se abonarán obligatoriamente a través de domiciliación bancaria.")

    # Consentimiento
    add_section_title("Consentimiento")
    pdf.set_font("DejaVu", size=10)
    pdf.cell(0, 10, "[X] He leído y acepto los términos y condiciones.", ln=True)
    pdf.cell(0, 10, "[X] He leído y acepto la política de privacidad.", ln=True)

    # Firma
    add_section_title("Firma")

    # Procesar imagen base64
    signature_data = base64.b64decode(data["signature"].split(",")[1])
    image = Image.open(BytesIO(signature_data))
    if image.mode in ("RGBA", "LA"):
        background = Image.new("RGB", image.size, (255, 255, 255))
        background.paste(image, mask=image.split()[-1])
        image = background
    else:
        image = image.convert("RGB")

    with tempfile.NamedTemporaryFile(suffix=".jpg", delete=False) as tmp_file:
        image_path = tmp_file.name
        image.save(image_path, format="JPEG")

    pdf.image(image_path, x=10, w=64)
    # Exportar como bytes
    pdf_bytes = pdf.output(dest='S').encode('latin1')

    # Guardar el PDF en el servidor
    output_dir = "generated_pdfs"
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)
    
    # Formato de fecha para el nombre del archivo
    date_str = datetime.now().strftime("%Y%m%d")
    # Limpiar el correo electrónico para usarlo en el nombre del archivo
    clean_email = "".join(c for c in data["email"] if c.isalnum() or c in ['.', '_', '-']).replace('@', '_at_')
    
    file_name = f"{clean_email}_{date_str}.pdf"
    output_path = os.path.join(output_dir, file_name)
    
    with open(output_path, "wb") as f:
        f.write(pdf_bytes)
        
    return pdf_bytes
