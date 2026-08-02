interface ExamSubmitSectionProps {
  hasPreviousSubmission: boolean;
  hasOfficialScore: boolean;
  eligibilityAllowed: boolean;
  eligibilityChecking: boolean;
  isSubmitting: boolean;
  formError: string | null;
  onFormErrorDismiss?: () => void;
}

export default function ExamSubmitSection({
  hasPreviousSubmission,
  hasOfficialScore,
  eligibilityAllowed,
  eligibilityChecking,
  isSubmitting,
  formError,
}: ExamSubmitSectionProps) {
  if (hasPreviousSubmission) return null;

  const buttonText = eligibilityChecking
    ? 'Comprobando...'
    : isSubmitting
      ? 'Enviando...'
      : hasOfficialScore
        ? 'Confirmar mis datos'
        : 'Entregar Examen';

  return (
    <>
      <button
        type="submit"
        className="btn-brand w-full text-lg mt-4 disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer"
        disabled={!eligibilityAllowed || eligibilityChecking || isSubmitting}
      >
        {buttonText}
      </button>
      {formError && (
        <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">{formError}</p>
      )}
    </>
  );
}
