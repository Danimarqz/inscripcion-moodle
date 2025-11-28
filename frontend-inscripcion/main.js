document.addEventListener('DOMContentLoaded', function () {
  // --- OBTENER ELEMENTOS DEL DOM ---
  const form = document.getElementById('inscriptionForm');
  const canvas = document.getElementById('signature-pad');
  const clearBtn = document.getElementById('clear');
  const submitBtn = form.querySelector('button[type="submit"]');
  const spinner = submitBtn.querySelector('.spinner-border');
  const responseModal = new bootstrap.Modal(document.getElementById('responseModal'));
  const responseModalLabel = document.getElementById('responseModalLabel');
  const responseModalBody = document.getElementById('responseModalBody');

  // --- INICIALIZAR LA FIRMA Y EL FORMULARIO ---


  const signaturePad = new SignaturePad(canvas);
  setSignaturePadDimensions();

  // Establecer la fecha actual en el campo requestdate
  const requestDateInput = document.getElementById('requestdate');
  if (requestDateInput) {
    const today = new Date();
    const yyyy = today.getFullYear();
    const mm = String(today.getMonth() + 1).padStart(2, '0'); // Enero es 0!
    const dd = String(today.getDate()).padStart(2, '0');
    requestDateInput.value = `${dd}-${mm}-${yyyy}`;
    console.log('requestdate input found and value set to:', requestDateInput.value);
  } else {
    console.log('requestdate input not found!');
  }




  // --- EVENT LISTENERS ---
  clearBtn.addEventListener('click', () => signaturePad.clear());
  window.addEventListener('resize', setSignaturePadDimensions);
  form.addEventListener('submit', handleFormSubmit);

  // --- FUNCIONES ---

  /**
   * Establece las dimensiones del lienzo de la firma en función del tamaño de su contenedor.
   */
  function setSignaturePadDimensions() {
    const ratio = Math.max(window.devicePixelRatio || 1, 1);

    // Obtiene tamaño real en CSS del canvas (considera aspect-ratio)
    const width = canvas.offsetWidth;
    const height = canvas.offsetHeight;

    // Ajusta tamaño físico para resolución retina
    canvas.width = width * ratio;
    canvas.height = height * ratio;

    // Ajusta tamaño CSS igual que el offset para que no haya distorsión
    canvas.style.width = width + 'px';
    canvas.style.height = height + 'px';

    const ctx = canvas.getContext('2d');
    ctx.scale(ratio, ratio);

    signaturePad.clear();
  }



  /**
   * Gestiona el envío del formulario.
   * @param {Event} e - El evento de envío.
   */
  async function handleFormSubmit(e) {
    e.preventDefault();
    if (!validateForm()) return;

    const formData = new FormData(form);
    const data = Object.fromEntries(formData.entries());
    data.signature = signaturePad.toDataURL();
    console.log('Data to send: ', data)

    // Mostrar el mensaje inmediatamente sin esperar a la respuesta.
    showModal('Éxito', 'Su inscripción se está gestionando.');
    form.reset();
    signaturePad.clear();

    // Enviar en segundo plano; solo se loguean errores.
    fetch('/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
      .then(async res => {
        if (!res.ok) {
          const result = await res.json().catch(() => ({}));
          console.error('Error al enviar la inscripción:', result.detail || res.statusText);
        }
      })
      .catch(err => {
        console.error('Error de red o inesperado al enviar la inscripción:', err);
      });
  }

  /**
   * Valida todo el formulario.
   * @returns {boolean} - `true` si el formulario es válido, `false` en caso contrario.
   */
  function validateForm() {
    let firstInvalidField = null;
    resetValidation();

    const fieldsToValidate = [
      'name', 'surname', 'dob', 'address', 'city', 'locality', 'postalcode', 'country',
      'dni', 'email', 'phone', 'iban', 'terms', 'privacy'
    ];

    for (const id of fieldsToValidate) {
      let isValid;
      const input = document.getElementById(id);

      if (input.type === 'checkbox') {
        isValid = validateCheckbox(id);
      } else if (id === 'dni') {
        isValid = validateDNI(id);
      } else if (id === 'email') {
        isValid = validateEmail(id);
      } else if (id === 'phone') {
        isValid = validatePhone(id);
      } else if (id === 'iban') {
        isValid = validateIBAN(id);
      } else {
        isValid = validateRequired(id);
      }

      if (!isValid && !firstInvalidField) {
        firstInvalidField = input;
      }
    }

    if (signaturePad.isEmpty()) {
      alert('Por favor, proporciona una firma.');
      if (!firstInvalidField) {
        firstInvalidField = canvas;
      }
    }

    if (firstInvalidField) {
      firstInvalidField.focus();
      firstInvalidField.scrollIntoView({ behavior: 'smooth', block: 'center' });
      return false;
    }

    return true;
  }

  /**
   * Restablece el estado de validación de todos los campos del formulario.
   */
  function resetValidation() {
    form.querySelectorAll('.is-invalid').forEach(el => el.classList.remove('is-invalid'));
  }

  /**
   * Valida que un campo obligatorio no esté vacío.
   * @param {string} id - El ID del campo.
   * @returns {boolean} - `true` si es válido, `false` en caso contrario.
   */
  function validateRequired(id) {
    const input = document.getElementById(id);
    if (!input.value.trim()) {
      input.classList.add('is-invalid');
      return false;
    }
    return true;
  }

  /**
   * Valida un número de DNI español.
   * @param {string} id - El ID del campo.
   * @returns {boolean} - `true` si es válido, `false` en caso contrario.
   */
  function validateDNI(id) {
    const dniInput = document.getElementById(id);
    let dni = dniInput.value.trim().toUpperCase();
    const dniRegex = /^\d{8}[A-Z]$/;
    const nieRegex = /^[XYZ]\d{7}[A-Z]$/;

    function calcLetter(number) {
      const letters = 'TRWAGMYFPDXBNJZSQVHLCKE';
      return letters.charAt(number % 23);
    }
    if (dniRegex.test(dni)) {
      // DNI normal
      const numbers = parseInt(dni.substring(0, 8), 10);
      const letter = dni.charAt(8);
      if (letter !== calcLetter(numbers)) {
        dniInput.classList.add('is-invalid');
        return false;
      }
      return true;
    } else if (nieRegex.test(dni)) {
      // NIE
      // Sustituir la letra inicial por su valor numérico
      let prefixNumber;
      switch (dni.charAt(0)) {
        case 'X': prefixNumber = '0'; break;
        case 'Y': prefixNumber = '1'; break;
        case 'Z': prefixNumber = '2'; break;
        default:
          dniInput.classList.add('is-invalid');
          return false;
      }
      const numbers = parseInt(prefixNumber + dni.substring(1, 8), 10);
      const letter = dni.charAt(8);
      if (letter !== calcLetter(numbers)) {
        dniInput.classList.add('is-invalid');
        return false;
      }
      return true;
    } else {
      dniInput.classList.add('is-invalid');
      return false;
    }
  }
  /**
   * Valida una dirección de correo electrónico.
   * @param {string} id - El ID del campo.
   * @returns {boolean} - `true` si es válido, `false` en caso contrario.
   */
  function validateEmail(id) {
    const emailInput = document.getElementById(id);
    const email = emailInput.value.trim();
    const emailRegex = /^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$/;
    if (!emailRegex.test(email)) {
      emailInput.classList.add('is-invalid');
      return false;
    }
    return true;
  }

  /**
   * Valida un número de teléfono.
   * @param {string} id - El ID del campo.
   * @returns {boolean} - `true` si es válido, `false` en caso contrario.
   */
  function validatePhone(id) {
    const phoneInput = document.getElementById(id);
    const phone = phoneInput.value.trim();
    const phoneRegex = /^[+]*[(]{0,1}[0-9]{1,4}[)]{0,1}[-\s\./0-9]*$/;
    if (!phoneRegex.test(phone)) {
      phoneInput.classList.add('is-invalid');
      return false;
    }
    return true;
  }

  /**
   * Valida un número de IBAN.
   * @param {string} id - El ID del campo.
   * @returns {boolean} - `true` si es válido, `false` en caso contrario.
   */
  function validateIBAN(id) {
    const ibanInput = document.getElementById(id);
    const iban = ibanInput.value.trim().replace(/\s/g, '');
    const ibanRegex = /^[A-Z]{2}\d{22}$/;
    if (!ibanRegex.test(iban)) {
      ibanInput.classList.add('is-invalid');
      return false;
    }
    return true;
  }

  /**
   * Valida que una casilla de verificación esté marcada.
   * @param {string} id - El ID de la casilla de verificación.
   * @returns {boolean} - `true` si es válida, `false` en caso contrario.
   */
  function validateCheckbox(id) {
    const checkbox = document.getElementById(id);
    if (!checkbox.checked) {
      checkbox.classList.add('is-invalid');
      return false;
    }
    return true;
  }

  /**
   * Muestra un modal con un mensaje.
   * @param {string} title - El título del modal.
   * @param {string} message - El mensaje a mostrar.
   */
  function showModal(title, message) {
    responseModalLabel.textContent = title;
    responseModalBody.textContent = message;
    responseModal.show();
  }

  /**
   * Establece el estado de carga del botón de envío.
   * @param {boolean} isLoading - `true` para mostrar el spinner, `false` para ocultarlo.
   */
  function setLoading(isLoading) {
    // Se elimina el estado de carga para no mostrar spinners durante el envío.
    submitBtn.disabled = false;
    spinner.style.display = 'none';
  }
});
