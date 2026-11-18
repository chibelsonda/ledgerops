interface LedgerOpsMarkProps {
  className?: string
}

export function LedgerOpsMark({ className }: LedgerOpsMarkProps) {
  return (
    <svg
      aria-hidden="true"
      className={className}
      fill="none"
      viewBox="0 0 32 32"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M8.5 6.75v18.5h16"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2.25"
      />
      <path
        d="m12.75 19.5 4-4.25 3.25 2.5 4.5-6.5"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2.25"
      />
      <circle cx="24.5" cy="11.25" r="1.5" fill="currentColor" />
    </svg>
  )
}
