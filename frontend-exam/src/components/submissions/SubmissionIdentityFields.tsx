import { normalizeDni } from '../../utils/validation';

interface SubmissionIdentityFieldsProps {
  studentName: string;
  studentSurname: string;
  email: string;
  dni: string;
  onNameChange: (value: string) => void;
  onSurnameChange: (value: string) => void;
  onEmailChange: (value: string) => void;
  onDniChange: (value: string) => void;
  onBlurCheck: (event: FocusEvent) => void;
  eligibilityError: string | null;
}

export default function SubmissionIdentityFields({
  studentName,
  studentSurname,
  email,
  dni,
  onNameChange,
  onSurnameChange,
  onEmailChange,
  onDniChange,
  onBlurCheck,
  eligibilityError,
}: SubmissionIdentityFieldsProps) {
  return (
    <>
      <div className="mb-6">
        <label htmlFor="name" className="block font-bold text-brand-pink mb-2">
          Nombre:
        </label>
        <input
          type="text"
          id="name"
          name="name"
          required
          value={studentName}
          onInput={(event) => onNameChange((event.target as HTMLInputElement).value)}
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
        />
      </div>

      <div className="mb-6">
        <label htmlFor="surname" className="block font-bold text-brand-pink mb-2">
          Apellidos:
        </label>
        <input
          type="text"
          id="surname"
          name="surname"
          required
          value={studentSurname}
          onInput={(event) => onSurnameChange((event.target as HTMLInputElement).value)}
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
        />
      </div>

      <div className="mb-6">
        <label htmlFor="email" className="block font-bold text-brand-pink mb-2">
          Email:
        </label>
        <input
          type="email"
          id="email"
          name="email"
          required
          value={email}
          onInput={(event) => onEmailChange((event.target as HTMLInputElement).value)}
          onBlur={onBlurCheck}
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
        />
      </div>

      <div className="mb-6">
        <label htmlFor="dni" className="block font-bold text-brand-pink mb-2">
          DNI/NIE:
        </label>
        <input
          type="text"
          id="dni"
          name="dni"
          required
          value={dni}
          onInput={(event) => onDniChange(normalizeDni((event.target as HTMLInputElement).value))}
          onBlur={onBlurCheck}
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
        />
      </div>

      {eligibilityError && (
        <p className="text-sm text-amber-400 bg-amber-400/10 border border-amber-400/40 p-3 rounded-md mb-4">
          {eligibilityError}
        </p>
      )}
    </>
  );
}
