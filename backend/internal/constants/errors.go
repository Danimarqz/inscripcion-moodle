package constants

const (
	InvalidCredentials = "credenciales invalidas"
	AdminAlreadyExists = "administrador ya existe"

	InvalidRequest          = "invalid request"
	FailedToLoadExams       = "failed to load exams"
	InvalidExamID           = "invalid exam id"
	ExamNotFound            = "exam not found"
	FailedToLoadQuestions   = "failed to load exam questions"
	FailedToVerifyResult  = "failed to verify official result"
	EmailAndDNIAreRequired  = "email and dni are required"
	DNIRequired             = "dni is required"
	AllFieldsRequired       = "all fields are required"
	UsernameAndPassword     = "username and password are required"
	SpamDetected            = "spam detected"
	FailedToRegister        = "failed to register submission"
	FailedToVerifyAdmin     = "failed to verify existing admin"
	FailedToHashPassword    = "failed to hash password"
	FailedToCreateAdmin     = "failed to create admin user"
	AdminCreated            = `{"message":"Administrador creado con exito"}`
	FailedToLookupAdmin     = "failed to lookup admin"
	FailedToCreateToken     = "failed to create token"
	MissingAuthHeader       = "missing authorization header"
	InvalidAuthHeader       = "invalid authorization header format"
	InvalidToken            = "invalid token"
	Unauthorized            = "unauthorized"
	TokenIsValid            = `{"detail": "Token valido", "user": "%s"}`
)
