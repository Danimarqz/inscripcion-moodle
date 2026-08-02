interface ExamOfficialNoticeProps {
  hasOfficialMerits: boolean;
}

export default function ExamOfficialNotice({ hasOfficialMerits }: ExamOfficialNoticeProps) {
  return (
    <p className="mt-6 rounded-lg border border-brand-blue/50 bg-brand-blue/10 p-4 text-sm text-brand-blue">
      PUBLICADAS NOTAS OFICIALES <br /><br />
      {hasOfficialMerits ? '' : 'Completa tus MÉRITOS y veras tu POSICION en relacion al resto de opositores.'}
    </p>
  );
}
