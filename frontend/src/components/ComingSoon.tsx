export interface ComingSoonProps {
  /** Button label; defaults to the German marker required by AC-19. */
  label?: string;
  /** Optional one-line explanation rendered below the control. */
  note?: string;
}

/**
 * A visibly disabled control labelled "Kommt bald" (AC-19). Use it wherever a
 * feature is not implemented yet so no dead-looking control appears active.
 */
export function ComingSoon({ label = "Kommt bald", note }: ComingSoonProps) {
  return (
    <div className="coming-soon">
      <button
        type="button"
        className="btn btn--secondary"
        disabled
        aria-disabled="true"
      >
        {label}
      </button>
      {note ? <p className="coming-soon__note">{note}</p> : null}
    </div>
  );
}
