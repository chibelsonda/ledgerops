import { Button as HeadlessButton, type ButtonProps as HeadlessButtonProps } from '@headlessui/react'
import clsx from 'clsx'

type ButtonProps = HeadlessButtonProps & {
  variant?: 'primary' | 'secondary' | 'ghost'
  full?: boolean
}

export function Button({ variant = 'primary', full, className, ...props }: ButtonProps) {
  return (
    <HeadlessButton
      {...props}
      className={clsx(
        'inline-flex items-center justify-center gap-2 rounded-lg px-3.5 py-2.5 text-xs font-semibold transition enabled:hover:-translate-y-px disabled:cursor-not-allowed disabled:opacity-55',
        variant === 'primary' && 'bg-emerald-700 text-white enabled:hover:bg-emerald-800',
        variant === 'secondary' && 'border border-stone-300 bg-amber-50 text-stone-900 enabled:hover:bg-white',
        variant === 'ghost' && 'bg-transparent text-stone-700 enabled:hover:bg-stone-100',
        full && 'w-full',
        className,
      )}
    />
  )
}
