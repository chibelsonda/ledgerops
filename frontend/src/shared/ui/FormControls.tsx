import { Field, Input as HeadlessInput, Label, Select as HeadlessSelect, Textarea as HeadlessTextarea } from '@headlessui/react'
import { ChevronDown } from 'lucide-react'
import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'

const controlClass = 'w-full rounded-lg border border-stone-300 bg-white px-3 py-2.5 text-xs leading-5 text-stone-900 outline-none transition placeholder:text-stone-500 focus:border-emerald-600 focus:ring-3 focus:ring-emerald-700/10'

type InputProps = InputHTMLAttributes<HTMLInputElement> & { label: string; icon?: ReactNode; hideLabel?: boolean; error?: string }

export function Input({ label, icon, hideLabel, error, className = '', ...props }: InputProps) {
  return (
    <Field className="relative w-full">
      <Label className={hideLabel ? 'sr-only' : 'mb-2 block text-xs font-semibold leading-relaxed text-stone-800'}>{label}</Label>
      {icon ? <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-stone-500">{icon}</span> : null}
      <HeadlessInput {...props} aria-invalid={Boolean(error)} className={`${controlClass} ${icon ? 'pl-9' : ''} ${error ? 'border-red-400' : ''} ${className}`} />
      {error ? <p className="mt-2 text-xs font-medium leading-relaxed text-red-700" role="alert">{error}</p> : null}
    </Field>
  )
}

type SelectProps = SelectHTMLAttributes<HTMLSelectElement> & { label: string; children: ReactNode; hideLabel?: boolean; error?: string }

export function Select({ label, children, hideLabel, error, className = '', ...props }: SelectProps) {
  return (
    <Field className="relative">
      <Label className={hideLabel ? 'sr-only' : 'mb-2 block text-xs font-semibold leading-relaxed text-stone-800'}>{label}</Label>
      <div className="relative">
        <HeadlessSelect {...props} aria-invalid={Boolean(error)} className={`${controlClass} appearance-none pr-9 ${error ? 'border-red-400' : ''} ${className}`}>{children}</HeadlessSelect>
        <ChevronDown className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-stone-500" size={14} />
      </div>
      {error ? <p className="mt-2 text-xs font-medium leading-relaxed text-red-700" role="alert">{error}</p> : null}
    </Field>
  )
}

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & { label: string; error?: string }

export function Textarea({ label, error, className = '', ...props }: TextareaProps) {
  return (
    <Field>
      <Label className="mb-2 block text-xs font-semibold leading-relaxed text-stone-800">{label}</Label>
      <HeadlessTextarea {...props} className={`${controlClass} min-h-28 resize-y leading-relaxed ${className}`} />
      {error ? <p className="mt-2 text-xs font-medium leading-relaxed text-red-700" role="alert">{error}</p> : null}
    </Field>
  )
}
